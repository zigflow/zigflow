/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// The tests below give an activity inputs the workflow already resolved:
// injectedExpr is text a resolved value happened to contain, so it must arrive
// as is, while a deferred inputAttemptExpr is evaluated here.
const (
	injectedExpr     = "${ $env.API_TOKEN }"
	inputAttemptExpr = "${ $data.activity.attempt }"
	commentEnv       = "COMMENT"
	echoCommand      = "echo"
	deferredAttempt  = "deferred attempt"
	nothingDeferred  = "nothing deferred"
)

func resolvedInputsState(deferred ...string) *utils.State {
	state := utils.NewState()
	state.Env = map[string]any{"API_TOKEN": "s3cr3t"}
	state.ActivityInputs = &utils.ActivityInputs{Deferred: deferred}
	return state
}

func TestCallHTTPActivityEvaluatesOnlyDeferredInputs(t *testing.T) {
	var got http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	task := &model.CallHTTP{
		Call: "http",
		With: model.HTTPArguments{
			Method:   http.MethodGet,
			Endpoint: model.NewEndpoint(server.URL),
			Headers: model.NewObjectOrRuntimeExpr(map[string]any{
				"X-Attempt": inputAttemptExpr,
				"X-Comment": injectedExpr,
			}),
		},
	}

	c := &CallHTTP{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(c.CallHTTPActivity)

	_, err := env.ExecuteActivity(c.CallHTTPActivity, task, nil, resolvedInputsState("/headers/X-Attempt"))
	require.NoError(t, err)

	assert.Equal(t, "1", got.Get("X-Attempt"))
	assert.Equal(t, injectedExpr, got.Get("X-Comment"))
}

func TestCallShellActivityEvaluatesOnlyDeferredInputs(t *testing.T) {
	task := &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
		Command:   echoCommand,
		Arguments: &model.RunArguments{Value: []string{injectedExpr, inputAttemptExpr}},
	}}}

	legacy := resolvedInputsState()
	legacy.ActivityInputs = nil

	for name, tc := range map[string]struct {
		state *utils.State
		want  string
	}{
		deferredAttempt: {state: resolvedInputsState("/exec/args/1"), want: injectedExpr + " 1"},
		nothingDeferred: {state: resolvedInputsState(), want: injectedExpr + " " + inputAttemptExpr},
		// Scheduled with unresolved inputs, the activity evaluates them all, as before.
		"unresolved inputs": {state: legacy, want: "s3cr3t 1"},
	} {
		t.Run(name, func(t *testing.T) {
			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()
			env.RegisterActivity(run.CallShellActivity)

			val, err := env.ExecuteActivity(run.CallShellActivity, task, nil, tc.state)
			require.NoError(t, err)
			var out string
			require.NoError(t, val.Get(&out))
			assert.Equal(t, tc.want, out)
		})
	}
}

func TestCallContainerActivityOnDockerEvaluatesOnlyDeferredInputs(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	task := &model.RunTask{Run: model.RunTaskConfiguration{Container: &model.Container{
		Image:       testImageNeverExist,
		Arguments:   []string{injectedExpr, inputAttemptExpr},
		Environment: map[string]string{commentEnv: injectedExpr, "ATTEMPT": inputAttemptExpr},
	}}}

	for name, tc := range map[string]struct {
		state       *utils.State
		wantAttempt string
	}{
		deferredAttempt: {state: resolvedInputsState("/arguments/1", "/environment/ATTEMPT"), wantAttempt: "1"},
		nothingDeferred: {state: resolvedInputsState(), wantAttempt: inputAttemptExpr},
	} {
		t.Run(name, func(t *testing.T) {
			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()
			env.RegisterActivity(run.CallContainerActivity)

			_, err := env.ExecuteActivity(run.CallContainerActivity, task, nil, tc.state, "", ContainerRuntimeDocker, "")
			// The image does not exist, so docker fails; the error carries the command it ran.
			var appErr *temporal.ApplicationError
			require.ErrorAs(t, err, &appErr)
			var details map[string]any
			require.NoError(t, appErr.Details(&details))
			command, ok := details["command"].([]any)
			require.True(t, ok, "the docker error carries the command")

			assert.Contains(t, command, "--env="+commentEnv+"="+injectedExpr)
			assert.Contains(t, command, "--env=ATTEMPT="+tc.wantAttempt)
			assert.Equal(t, []any{testImageNeverExist, injectedExpr, tc.wantAttempt}, command[len(command)-3:])
			assert.NotContains(t, command, "s3cr3t")
		})
	}
}

func TestBuildJobSpecEvaluatesOnlyDeferredInputs(t *testing.T) {
	task := makeContainerTask()
	task.Run.Container.Arguments = []string{injectedExpr, inputAttemptExpr}
	task.Run.Container.Environment = map[string]string{commentEnv: injectedExpr}

	for name, tc := range map[string]struct {
		state       *utils.State
		wantAttempt string
	}{
		deferredAttempt: {state: resolvedInputsState("/arguments/1"), wantAttempt: "1"},
		nothingDeferred: {state: resolvedInputsState(), wantAttempt: inputAttemptExpr},
	} {
		t.Run(name, func(t *testing.T) {
			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()

			var got *batchv1.Job
			testActivity := func(ctx context.Context) error {
				j, err := run.buildJobSpec(ctx, task, testKubeNamespace, testKubeServiceAccount, tc.state)
				if err != nil {
					return err
				}
				if j == nil {
					return errors.New("no job spec")
				}
				got = j
				return nil
			}
			env.RegisterActivity(testActivity)

			_, err := env.ExecuteActivity(testActivity)
			require.NoError(t, err)

			c := got.Spec.Template.Spec.Containers[0]
			assert.Equal(t, []string{injectedExpr, tc.wantAttempt}, c.Args)
			assert.Equal(t, []corev1.EnvVar{{Name: commentEnv, Value: injectedExpr}}, c.Env)
		})
	}
}

// Values whose JSON form is easy to change on the way to the activity: small and large
// integers (2^53 + 1, beyond int64), floats, booleans, null, nested values and numbers
// that already crossed Temporal as workflow input.
var resolvedValueExprs = []string{
	"${ 5 }", "${ 1000000000000000 }", "${ 9007199254740993 }", "${ 1000000000000000000000000000000 }",
	"${ 1.5 }", "${ 1e21 }", "${ true }", "${ null }", "${ {a: 9007199254740993, b: [1, 2.5]} }",
	"${ $input.n }", "${ $input.big }",
}

// numbersState is a workflow's state as Temporal delivers it: its input already decoded from JSON.
func numbersState(t *testing.T) *utils.State {
	state := utils.NewState()
	state.Input = map[string]any{"n": 7, "big": 9007199254740993}
	return throughTemporal(t, state)
}

// throughTemporal returns v as an activity receives it: encoded and decoded by Temporal's
// default JSON data converter.
func throughTemporal[T any](t *testing.T, v T) T {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(v)
	require.NoError(t, err)
	var out T
	require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payload, &out))
	return out
}

func resolvedState(state *utils.State, inputs *utils.ActivityInputs) *utils.State {
	s := state.Clone()
	s.ActivityInputs = inputs
	return s
}

// Resolving in the workflow must send exactly the request the activity sent when it
// evaluated the inputs itself.
func TestCallHTTPActivitySendsTheSameRequestAsBefore(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = fmt.Sprintf("body=%s query=%s header=%s", body, r.URL.RawQuery, r.Header.Get("X-V"))
	}))
	defer server.Close()

	c := &CallHTTP{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(c.CallHTTPActivity)

	for _, expr := range resolvedValueExprs {
		task := &model.CallHTTP{Call: "http", With: model.HTTPArguments{
			Method:   http.MethodPost,
			Endpoint: model.NewEndpoint(server.URL),
			Headers:  model.NewObjectOrRuntimeExpr(map[string]any{"X-V": expr}),
			Query:    model.NewObjectOrRuntimeExpr(map[string]any{"v": expr}),
			Body:     json.RawMessage(`{"v": "` + expr + `"}`),
		}}
		state := numbersState(t)

		_, err := env.ExecuteActivity(c.CallHTTPActivity, task, nil, state)
		require.NoError(t, err, expr)
		before := got

		resolved, inputs, err := ResolveHTTPInputs(task, state)
		require.NoError(t, err, expr)
		_, err = env.ExecuteActivity(c.CallHTTPActivity, throughTemporal(t, resolved), nil, resolvedState(state, inputs))
		require.NoError(t, err, expr)

		assert.Equal(t, before, got, expr)
	}
}

func TestCallGRPCActivityEvaluatesTheSameArgumentsAsBefore(t *testing.T) {
	for _, expr := range resolvedValueExprs {
		task := &model.CallGRPC{Call: "grpc", With: model.GRPCArguments{
			Method:    "Command",
			Arguments: map[string]any{"v": expr},
		}}
		state := numbersState(t)
		tree := func(task *model.CallGRPC) map[string]any {
			return map[string]any{grpcInputArgs: task.With.Arguments, grpcInputMethod: task.With.Method}
		}

		before, err := utils.EvaluateActivityInput("", tree(throughTemporal(t, task)), state)
		require.NoError(t, err, expr)

		resolved, inputs, err := ResolveGRPCInputs(task, state)
		require.NoError(t, err, expr)
		after, err := utils.EvaluateActivityInput("", tree(throughTemporal(t, resolved)), resolvedState(state, inputs))
		require.NoError(t, err, expr)

		// grpcurl sends the arguments as JSON.
		wantJSON, _ := json.Marshal(before)
		gotJSON, _ := json.Marshal(after)
		assert.JSONEq(t, string(wantJSON), string(gotJSON), expr)
		assert.Equal(t, string(wantJSON), string(gotJSON), expr)
	}
}

func TestCallShellActivityPrintsTheSameArgumentsAsBefore(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(run.CallShellActivity)

	for _, expr := range resolvedValueExprs {
		task := &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
			Command: echoCommand, Arguments: &model.RunArguments{Value: []string{expr}},
		}}}
		state := numbersState(t)

		val, err := env.ExecuteActivity(run.CallShellActivity, task, nil, state)
		require.NoError(t, err, expr)
		var before, after string
		require.NoError(t, val.Get(&before))

		resolved, inputs, err := ResolveRunInputs(task, ContainerRuntimeDocker, state)
		require.NoError(t, err, expr)
		val, err = env.ExecuteActivity(run.CallShellActivity, throughTemporal(t, resolved), nil, resolvedState(state, inputs))
		require.NoError(t, err, expr)
		require.NoError(t, val.Get(&after))

		assert.Equal(t, before, after, expr)
	}
}

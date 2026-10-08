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

package tasks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const (
	activityInputDoc  = "wf-activity-input"
	activityInputTask = "fetch"
	attemptExpr       = "${ $data.activity.attempt }"
	injectedExpr      = "${ $env.API_TOKEN }"
	hostExpr          = "${ $env.HOST }"
	commentExpr       = "${ $input.comment }"
	callHTTPType      = "http"
	hostEnv           = "HOST"
	attemptKey        = "attempt"
	host              = "example.com"
)

var activityInputWorkflow = &model.Workflow{Document: model.Document{Name: activityInputDoc}}

// activityInputState holds an input whose comment is text that looks like an
// expression: it must reach the activity as text, never as the token.
func activityInputState() *utils.State {
	state := utils.NewState()
	state.Env = map[string]any{"API_TOKEN": "s3cr3t", hostEnv: host}
	state.Input = map[string]any{"comment": injectedExpr, "id": 2}
	return state
}

// executeTask runs fn in a test workflow with activityFn registered under the
// task's activity name, and returns the workflow error.
func executeTask(
	t *testing.T, fn TemporalWorkflowFunc, activityFn any, setup ...func(*testsuite.TestWorkflowEnvironment),
) error {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(activityFn, activity.RegisterOptions{Name: activityInputDoc + "." + activityInputTask})
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, nil, activityInputState())
	}, workflow.RegisterOptions{Name: "activity-inputs"})
	for _, f := range setup {
		f(env)
	}

	env.ExecuteWorkflow("activity-inputs")
	return env.GetWorkflowError()
}

func newActivityInputHTTPTask() *model.CallHTTP {
	return &model.CallHTTP{
		Call: callHTTPType,
		With: model.HTTPArguments{
			Method:   "post",
			Endpoint: model.NewEndpoint(`${ "https://" + $env.HOST + "/posts/" + ($input.id | tostring) }`),
			Body:     json.RawMessage(`{"comment": "${ $input.comment }", "attempt": "${ $data.activity.attempt }"}`),
		},
	}
}

func buildHTTP(t *testing.T, task *model.CallHTTP) TemporalWorkflowFunc {
	t.Helper()
	b, err := NewCallHTTPTaskBuilder(nil, task, activityInputTask, activityInputWorkflow, testEvents, nil)
	require.NoError(t, err)
	fn, err := b.Build()
	require.NoError(t, err)
	return fn
}

func TestCallHTTPSchedulesResolvedInputs(t *testing.T) {
	task := newActivityInputHTTPTask()

	var scheduled *model.CallHTTP
	var activityState *utils.State
	err := executeTask(t, buildHTTP(t, task), func(task *model.CallHTTP, _ any, state *utils.State) (any, error) {
		scheduled, activityState = task, state
		return map[string]any{}, nil
	})
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/posts/2", scheduled.With.Endpoint.String())
	assert.JSONEq(t, `{"comment": "${ $env.API_TOKEN }", "attempt": "${ $data.activity.attempt }"}`,
		string(scheduled.With.Body))
	assert.Equal(t, &utils.ActivityInputs{Deferred: []string{"/body/attempt"}}, activityState.ActivityInputs)

	// What the activity sends: the comment as the input's text, the attempt from the attempt running.
	attemptState := activityState.Clone().AddData(map[string]any{"activity": map[string]any{attemptKey: 1}})
	args, err := activities.ParseHTTPArguments(scheduled, attemptState)
	require.NoError(t, err)
	assert.JSONEq(t, `{"comment": "${ $env.API_TOKEN }", "attempt": 1}`, string(args.Body))

	assert.Equal(t, `${ "https://" + $env.HOST + "/posts/" + ($input.id | tostring) }`, task.With.Endpoint.String(),
		"the shared task must not be modified")
}

func TestCallGRPCSchedulesResolvedInputs(t *testing.T) {
	task := &model.CallGRPC{
		Call: "grpc",
		With: model.GRPCArguments{
			Proto:   &model.ExternalResource{Endpoint: model.NewEndpoint("file:///tmp/basic.proto")},
			Service: model.GRPCService{Name: "providers.v1.BasicService", Host: hostExpr, Port: 3000},
			Method:  `${ "Command" + ($input.id | tostring) }`,
			Arguments: map[string]any{
				"input":    commentExpr,
				"list":     []any{hostExpr},
				attemptKey: attemptExpr,
			},
		},
	}
	b, err := NewCallGRPCTaskBuilder(nil, task, activityInputTask, activityInputWorkflow, testEvents, nil)
	require.NoError(t, err)
	fn, err := b.Build()
	require.NoError(t, err)

	var scheduled *model.CallGRPC
	var activityState *utils.State
	err = executeTask(t, fn, func(task *model.CallGRPC, _ any, state *utils.State) (any, error) {
		scheduled, activityState = task, state
		return map[string]any{}, nil
	})
	require.NoError(t, err)

	assert.Equal(t, "Command2", scheduled.With.Method)
	// The gRPC activity never adds activity metadata, so nothing is deferred.
	assert.Equal(t, map[string]any{"input": injectedExpr, "list": []any{host}, attemptKey: nil},
		scheduled.With.Arguments)
	assert.Empty(t, activityState.ActivityInputs.Deferred)
	assert.Equal(t, hostExpr, scheduled.With.Service.Host, "service is not evaluated, as before")
	assert.Equal(t, `${ "Command" + ($input.id | tostring) }`, task.With.Method, "the shared task must not be modified")
}

func buildRun(t *testing.T, task *model.RunTask, opts *TaskOpts) TemporalWorkflowFunc {
	t.Helper()
	b, err := NewRunTaskBuilder(nil, task, activityInputTask, activityInputWorkflow, testEvents, opts)
	require.NoError(t, err)
	require.NoError(t, b.PostLoad())
	fn, err := b.Build()
	require.NoError(t, err)
	return fn
}

func scheduleRun(t *testing.T, task *model.RunTask) (*model.RunTask, *utils.State) {
	t.Helper()
	var scheduled *model.RunTask
	var activityState *utils.State
	err := executeTask(t, buildRun(t, task, nil), func(task *model.RunTask, _ any, state *utils.State) (any, error) {
		scheduled, activityState = task, state
		return "", nil
	})
	require.NoError(t, err)
	return scheduled, activityState
}

func TestRunShellSchedulesResolvedInputs(t *testing.T) {
	task := &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
		Command:     "echo",
		Arguments:   &model.RunArguments{Value: []string{"${ $input.id }", attemptExpr, commentExpr}},
		Environment: map[string]string{"TOKEN": injectedExpr, "ATTEMPT": attemptExpr},
	}}}

	scheduled, activityState := scheduleRun(t, task)

	assert.Equal(t, []string{"2", attemptExpr, injectedExpr}, scheduled.Run.Shell.Arguments.AsSlice())
	assert.Equal(t, map[string]string{"TOKEN": "s3cr3t", "ATTEMPT": attemptExpr}, scheduled.Run.Shell.Environment)
	assert.Equal(t, []string{"/exec/args/1", "/exec/env/ATTEMPT"}, activityState.ActivityInputs.Deferred)
	assert.Equal(t, "${ $input.id }", task.Run.Shell.Arguments.AsSlice()[0], "the shared task must not be modified")
}

func TestRunScriptSchedulesResolvedSource(t *testing.T) {
	task := &model.RunTask{Run: model.RunTaskConfiguration{Script: &model.Script{
		Language: "python",
		External: &model.ExternalResource{Endpoint: model.NewEndpoint(`${ "https://" + $env.HOST + "/run.py" }`)},
	}}}

	scheduled, activityState := scheduleRun(t, task)

	assert.Equal(t, "https://example.com/run.py", scheduled.Run.Script.External.Endpoint.String())
	assert.Empty(t, activityState.ActivityInputs.Deferred)
}

func TestRunScriptDefersASourceThatReadsActivityState(t *testing.T) {
	source := `${ "https://example.com/" + ($data.activity.attempt | tostring) + ".py" }`
	task := &model.RunTask{Run: model.RunTaskConfiguration{Script: &model.Script{
		Language: "python",
		External: &model.ExternalResource{Endpoint: model.NewEndpoint(source)},
	}}}

	scheduled, activityState := scheduleRun(t, task)

	assert.Equal(t, source, scheduled.Run.Script.External.Endpoint.String())
	assert.Equal(t, []string{"/source"}, activityState.ActivityInputs.Deferred)
}

func TestRunContainerSchedulesResolvedInputs(t *testing.T) {
	newTask := func() *model.RunTask {
		return &model.RunTask{Run: model.RunTaskConfiguration{Container: &model.Container{
			Image:       `${ "registry.example.com/app:" + ($input.id | tostring) }`,
			Arguments:   []string{commentExpr, attemptExpr},
			Environment: map[string]string{hostEnv: hostExpr},
		}}}
	}

	for name, tc := range map[string]struct {
		runtime   activities.ContainerRuntime
		wantImage string
	}{
		"docker evaluates the image": {runtime: activities.ContainerRuntimeDocker, wantImage: "registry.example.com/app:2"},
		// The Kubernetes runtime never evaluated the image.
		"kubernetes leaves the image": {
			runtime:   activities.ContainerRuntimeKubernetes,
			wantImage: `${ "registry.example.com/app:" + ($input.id | tostring) }`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var scheduled *model.RunTask
			var activityState *utils.State
			fn := buildRun(t, newTask(), &TaskOpts{Run: &RunTaskOpts{Runtime: tc.runtime}})
			err := executeTask(t, fn, func(
				task *model.RunTask, _ any, state *utils.State, _ string, _ activities.ContainerRuntime, _ string,
			) (any, error) {
				scheduled, activityState = task, state
				return "", nil
			})
			require.NoError(t, err)

			assert.Equal(t, tc.wantImage, scheduled.Run.Container.Image)
			assert.Equal(t, []string{injectedExpr, attemptExpr}, scheduled.Run.Container.Arguments)
			assert.Equal(t, map[string]string{hostEnv: host}, scheduled.Run.Container.Environment)
			assert.Equal(t, []string{"/arguments/1"}, activityState.ActivityInputs.Deferred)
		})
	}
}

func TestActivityInputResolutionErrorFailsTheTask(t *testing.T) {
	task := &model.CallHTTP{
		Call: callHTTPType,
		With: model.HTTPArguments{Method: "get", Endpoint: model.NewEndpoint(`${ error("boom") }`)},
	}

	err := executeTask(t, buildHTTP(t, task), func(*model.CallHTTP, any, *utils.State) (any, error) {
		t.Fatal("the activity must not be scheduled")
		return nil, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error resolving activity inputs for task fetch")
	assert.Contains(t, err.Error(), "boom")
}

// An execution that started before inputs were resolved in the workflow has no
// version marker: replay must schedule the task exactly as it was recorded.
func TestExecutionWithoutVersionMarkerSchedulesInputsUnresolved(t *testing.T) {
	task := newActivityInputHTTPTask()

	var scheduled *model.CallHTTP
	var activityState *utils.State
	err := executeTask(t, buildHTTP(t, task), func(task *model.CallHTTP, _ any, state *utils.State) (any, error) {
		scheduled, activityState = task, state
		return map[string]any{}, nil
	}, func(env *testsuite.TestWorkflowEnvironment) {
		env.OnGetVersion(activityInputsVersionChangeID, workflow.DefaultVersion, activityInputsVersion).
			Return(workflow.DefaultVersion)
	})
	require.NoError(t, err)

	assert.Equal(t, task.With.Endpoint.String(), scheduled.With.Endpoint.String())
	assert.JSONEq(t, string(task.With.Body), string(scheduled.With.Body))
	assert.Nil(t, activityState.ActivityInputs)
}

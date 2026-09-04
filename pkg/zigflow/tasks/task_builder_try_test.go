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
	"errors"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type namedTryError struct{}

func (namedTryError) Error() string { return "plain" }

func TestTryTaskBuilderGetTasks(t *testing.T) {
	task := &model.TryTask{
		Try: &model.TaskList{
			&model.TaskItem{Key: "task", Task: &model.SetTask{}},
		},
		Catch: &model.TryTaskCatch{
			Do: &model.TaskList{
				&model.TaskItem{Key: "catch", Task: &model.SetTask{}},
			},
		},
	}

	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			task: task,
		},
	}

	got := builder.getTasks()
	assert.Equal(t, task.Try, got["try"])
	assert.Equal(t, task.Catch.Do, got["catch"])
}

func TestTryTaskBuilderBuildExecutesBodiesInline(t *testing.T) {
	tests := []struct {
		name    string
		tryTask model.Task
		want    map[string]any
	}{
		{
			name: "try succeeds",
			tryTask: &model.SetTask{
				Set: model.NewObjectOrRuntimeExpr(map[string]any{testConstValue: "try"}),
			},
			want: map[string]any{testConstValue: "try"},
		},
		{
			name: "catch handles failure",
			tryTask: &model.RaiseTask{
				Raise: model.RaiseTaskConfiguration{
					Error: model.RaiseTaskError{Definition: &model.Error{
						Type:   model.NewUriTemplate(model.ErrorTypeRuntime),
						Title:  model.NewStringOrRuntimeExpr("try failed"),
						Detail: model.NewStringOrRuntimeExpr("boom"),
					}},
				},
			},
			want: map[string]any{testConstHandledKey: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			events, readEventTypes := newRecordingEvents(t)
			task := &model.TryTask{
				Try: &model.TaskList{
					&model.TaskItem{Key: "try-step", Task: tc.tryTask},
				},
				Catch: &model.TryTaskCatch{Do: &model.TaskList{
					&model.TaskItem{Key: "catch-step", Task: &model.SetTask{
						Set: model.NewObjectOrRuntimeExpr(map[string]any{testConstHandledKey: true}),
					}},
				}},
			}
			builder := &TryTaskBuilder{builder: builder[*model.TryTask]{
				doc:            testWorkflow,
				eventEmitter:   events,
				name:           "try-inline",
				task:           task,
				temporalWorker: new(WorkflowRegistryMock),
			}}

			fn, err := builder.Build()
			require.NoError(t, err)

			state := utils.NewState()
			childStarts := 0
			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()
			env.SetOnChildWorkflowStartedListener(func(*workflow.Info, workflow.Context, converter.EncodedValues) {
				childStarts++
			})
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				return fn(ctx, nil, state)
			}, workflow.RegisterOptions{Name: "try-inline"})

			env.ExecuteWorkflow("try-inline")
			require.NoError(t, env.GetWorkflowError())
			var result map[string]any
			require.NoError(t, env.GetWorkflowResult(&result))
			assert.Equal(t, tc.want, result)
			assert.Zero(t, childStarts)
			assert.Empty(t, state.Data, "inline body mutations must remain isolated from the parent state")

			eventTypes := readEventTypes()
			assert.Contains(t, eventTypes, "dev.zigflow.task.started")
			assert.Contains(t, eventTypes, "dev.zigflow.task.completed")
			assert.NotContains(t, eventTypes, "dev.zigflow.workflow.started")
			assert.NotContains(t, eventTypes, "dev.zigflow.workflow.completed")
		})
	}
}

func TestTryTaskBuilderExecRunsCatchOnError(t *testing.T) {
	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					Do: &model.TaskList{},
				},
			},
		},
	}

	fn, err := builder.exec(
		func(workflow.Context, any, *utils.State) (any, error) {
			return nil, errors.New("boom")
		},
		func(workflow.Context, any, *utils.State) (any, error) {
			return map[string]any{testConstHandledKey: true}, nil
		},
	)
	assert.NoError(t, err)

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "try-exec"})

	env.ExecuteWorkflow("try-exec")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, map[string]any{testConstHandledKey: true}, result)
}

func TestTryTaskBuilderExecCatchStartsFromParentState(t *testing.T) {
	builder := &TryTaskBuilder{builder: builder[*model.TryTask]{
		task: &model.TryTask{Catch: &model.TryTaskCatch{}},
	}}
	parentState := utils.NewState().AddData(map[string]any{"parent": true})
	var caughtData map[string]any
	fn, err := builder.exec(
		func(_ workflow.Context, _ any, state *utils.State) (any, error) {
			state.AddData(map[string]any{"try-only": true})
			return nil, namedTryError{}
		},
		func(_ workflow.Context, _ any, state *utils.State) (any, error) {
			caughtData = state.Data
			return map[string]any{testConstHandledKey: true}, nil
		},
	)
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, parentState)
	}, workflow.RegisterOptions{Name: "try-state-isolation"})
	env.ExecuteWorkflow("try-state-isolation")
	require.NoError(t, env.GetWorkflowError())

	assert.Equal(t, true, caughtData["parent"])
	assert.NotContains(t, caughtData, "try-only")
	assert.NotContains(t, parentState.Data, "try-only")
	assert.NotContains(t, parentState.Data, "error")
}

// TestTryTaskBuilderExecPropagatesEndFromTry proves that `then: end` inside
// the inline try body is not treated as a catchable failure.
func TestTryTaskBuilderExecPropagatesEndFromTry(t *testing.T) {
	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task-end",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					Do: &model.TaskList{},
				},
			},
		},
	}

	state := utils.NewState()
	tryOutput := map[string]any{testConstValue: "end-time-output"}
	catchRan := false
	fn, err := builder.exec(
		func(workflow.Context, any, *utils.State) (any, error) {
			return tryOutput, flow.ErrEnd
		},
		func(workflow.Context, any, *utils.State) (any, error) {
			catchRan = true
			return map[string]any{testConstHandledKey: true}, nil
		},
	)
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	var output any
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		var err error
		output, err = fn(ctx, nil, state)
		return nil, err
	}, workflow.RegisterOptions{Name: "try-exec-end"})

	env.ExecuteWorkflow("try-exec-end")

	// The try task surfaces flow.ErrEnd through the Temporal envelope.
	wErr := env.GetWorkflowError()
	require.Error(t, wErr)
	assert.Contains(t, wErr.Error(), flow.ErrEnd.Error())
	assert.Equal(t, tryOutput, output)
	assert.False(t, catchRan, "catch handler must not run when the try body signalled end")
}

// TestTryTaskBuilderExecPropagatesEndFromCatch is the symmetric case: when
// the inline catch body emits `then: end`, that end propagates unchanged.
func TestTryTaskBuilderExecPropagatesEndFromCatch(t *testing.T) {
	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task-catch-end",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					Do: &model.TaskList{},
				},
			},
		},
	}

	state := utils.NewState()
	catchOutput := map[string]any{testConstValue: "catch-end-output"}
	fn, err := builder.exec(
		func(workflow.Context, any, *utils.State) (any, error) {
			return nil, errors.New("boom from try")
		},
		func(workflow.Context, any, *utils.State) (any, error) {
			return catchOutput, flow.ErrEnd
		},
	)
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	var output any
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		var err error
		output, err = fn(ctx, nil, state)
		return nil, err
	}, workflow.RegisterOptions{Name: "try-exec-catch-end"})

	env.ExecuteWorkflow("try-exec-catch-end")

	wErr := env.GetWorkflowError()
	require.Error(t, wErr)
	assert.Contains(t, wErr.Error(), flow.ErrEnd.Error())
	assert.Equal(t, catchOutput, output)
	assert.NotContains(t, wErr.Error(), "error calling catch workflow",
		"catch-emitted end must not be wrapped as a child-workflow failure")
}

// runCatchAndCaptureState executes a try task whose inline try body fails, then
// returns the $data the inline catch body actually observed alongside the
// parent state the exec function was handed. The catch body records the data
// it receives into a closure-captured map so the test can assert on the exact
// caught-error contract exposed under $data.
func runCatchAndCaptureState(t *testing.T, catchAs string, tryErr error) (caughtData map[string]any, parentState *utils.State) {
	t.Helper()

	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task-capture",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					As: catchAs,
					Do: &model.TaskList{},
				},
			},
		},
	}

	fn, err := builder.exec(
		func(workflow.Context, any, *utils.State) (any, error) {
			return nil, tryErr
		},
		func(_ workflow.Context, _ any, st *utils.State) (any, error) {
			caughtData = st.Data
			return map[string]any{testConstHandledKey: true}, nil
		},
	)
	require.NoError(t, err)

	parentState = utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, parentState)
	}, workflow.RegisterOptions{Name: "try-exec-capture"})

	env.ExecuteWorkflow("try-exec-capture")
	require.NoError(t, env.GetWorkflowError())

	return caughtData, parentState
}

// TestTryTaskBuilderExecExposesErrorUnderDefaultKey proves the inline catch
// body sees the caught error under $data.error when catch.as is unset.
func TestTryTaskBuilderExecExposesErrorUnderDefaultKey(t *testing.T) {
	caughtData, _ := runCatchAndCaptureState(t, "", temporal.NewApplicationError("kaboom", "MyAppError"))

	caughtErr, ok := caughtData["error"].(map[string]any)
	require.True(t, ok, "catch body must see the caught error under $data.error")

	assert.Equal(t, "MyAppError", caughtErr["type"])
	assert.Equal(t, "kaboom", caughtErr["message"])
	assert.NotContains(t, caughtErr, "childWorkflow")
}

// TestTryTaskBuilderExecExposesErrorUnderCustomKey proves the inline catch
// body sees the caught error under $data.<catch.as> when it is configured,
// and that the default "error" key is not used in that case.
func TestTryTaskBuilderExecExposesErrorUnderCustomKey(t *testing.T) {
	const customKey = "failure"

	caughtData, _ := runCatchAndCaptureState(t, customKey, temporal.NewApplicationError("kaboom", "MyAppError"))

	caughtErr, ok := caughtData[customKey].(map[string]any)
	require.True(t, ok, "catch body must see the caught error under the custom $data key")
	assert.Equal(t, "MyAppError", caughtErr["type"])
	assert.Equal(t, "kaboom", caughtErr["message"])

	assert.NotContains(t, caughtData, "error", "default error key must not be set when catch.as is configured")
}

// TestTryTaskBuilderExecDoesNotLeakErrorIntoParentState proves the injected
// caught error lives only on the cloned catch state and never mutates the
// parent state that later tasks observe. This guards Zigflow's explicit state
// propagation model: the error is only carried forward if the catch tasks
// output it.
func TestTryTaskBuilderExecDoesNotLeakErrorIntoParentState(t *testing.T) {
	_, parentState := runCatchAndCaptureState(t, "", temporal.NewApplicationError("kaboom", "MyAppError"))

	assert.NotContains(t, parentState.Data, "error",
		"caught error must not leak back into the parent state after catch completes")
	assert.Empty(t, parentState.Data, "parent state data must be untouched by catch error injection")
}

func TestTryTaskBuilderExecIncludesMetadataForExplicitChildFailure(t *testing.T) {
	const childWorkflowName = "try-explicit-failing-child"

	builder := &TryTaskBuilder{builder: builder[*model.TryTask]{
		task: &model.TryTask{Catch: &model.TryTaskCatch{}},
	}}
	var caughtErr map[string]any
	fn, err := builder.exec(
		func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			return nil, workflow.ExecuteChildWorkflow(ctx, childWorkflowName).Get(ctx, nil)
		},
		func(_ workflow.Context, _ any, state *utils.State) (any, error) {
			caughtErr, _ = state.Data["error"].(map[string]any)
			return nil, nil
		},
	)
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(workflow.Context) error {
		return temporal.NewApplicationError("child failed", "ExplicitChildError")
	}, workflow.RegisterOptions{Name: childWorkflowName})
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: "try-explicit-child-catch"})

	env.ExecuteWorkflow("try-explicit-child-catch")
	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, caughtErr)
	assert.Equal(t, "ExplicitChildError", caughtErr["type"])
	assert.Equal(t, "child failed", caughtErr["message"])

	childMetadata, ok := caughtErr["childWorkflow"].(map[string]any)
	require.True(t, ok, "an actual child workflow failure must retain child metadata")
	assert.Equal(t, childWorkflowName, childMetadata["workflowType"])
	assert.NotEmpty(t, childMetadata["workflowID"])
}

// TestBuildCatchError gives direct, deterministic coverage of the Temporal
// error enrichment without round-tripping every error shape through a workflow.
func TestBuildCatchError(t *testing.T) {
	tb := &TryTaskBuilder{}

	t.Run("application error fields", func(t *testing.T) {
		details := map[string]any{"reason": "quota exceeded"}
		appErr := temporal.NewNonRetryableApplicationError(
			"boom message", "BoomError", errors.New("root cause"), details,
		)

		out := tb.buildCatchError(appErr)

		assert.Equal(t, "BoomError", out["type"])
		assert.Equal(t, "boom message", out["message"])
		assert.Equal(t, true, out["nonRetryable"])
		assert.Equal(t, "root cause", out["cause"])
		assert.Equal(t, details, out["details"])
	})

	t.Run("retryable application error without details", func(t *testing.T) {
		appErr := temporal.NewApplicationError("transient", "TransientError")

		out := tb.buildCatchError(appErr)

		assert.Equal(t, "TransientError", out["type"])
		assert.Equal(t, "transient", out["message"])
		assert.Equal(t, false, out["nonRetryable"])
		assert.NotContains(t, out, "details")
	})

	t.Run("generic error fields", func(t *testing.T) {
		out := tb.buildCatchError(namedTryError{})

		assert.Equal(t, "namedTryError", out["type"])
		assert.Equal(t, "plain", out["message"])
		assert.Equal(t, false, out["nonRetryable"])
		assert.NotContains(t, out, "childWorkflow")
	})
}

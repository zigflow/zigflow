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
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

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

// newInlineTryBuilder builds a TryTaskBuilder wired for inline execution.
// The try and catch bodies are supplied directly to exec as
// TemporalWorkflowFunc closures, so no child workflows are registered.
func newInlineTryBuilder(catchAs string) *TryTaskBuilder {
	return &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					As: catchAs,
					Do: &model.TaskList{},
				},
			},
		},
	}
}

// runInlineTry executes fn inside Temporal's workflow test environment (exec
// needs a real workflow.Context for its logger and state clone) and returns
// the raw output and error the inline function produced, captured before they
// cross the test-environment boundary. Capturing pre-boundary keeps the error
// chain intact so callers can assert with errors.Is.
func runInlineTry(t *testing.T, fn TemporalWorkflowFunc, input any, state *utils.State) (any, error) {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	var (
		gotOutput any
		gotErr    error
	)
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		gotOutput, gotErr = fn(ctx, input, state)
		return gotOutput, gotErr
	}, workflow.RegisterOptions{Name: "try-exec"})

	env.ExecuteWorkflow("try-exec")

	return gotOutput, gotErr
}

// TestTryTaskBuilderExecRunsCatchOnError proves the catch body runs when the
// try body returns a genuine failure, that the try body receives the original
// input and state, and that the catch output becomes the result.
func TestTryTaskBuilderExecRunsCatchOnError(t *testing.T) {
	builder := newInlineTryBuilder("")

	state := utils.NewState()
	state.Input = map[string]any{"in": "put"}

	var (
		tryInput any
		tryState *utils.State
		catchRan bool
	)

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		tryInput = input
		tryState = st
		return nil, errors.New("boom")
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		catchRan = true
		return map[string]any{testConstHandledKey: true}, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineTry(t, fn, state.Input, state)
	require.NoError(t, execErr)

	// The try body must receive the original input and the exact parent state.
	assert.Equal(t, state.Input, tryInput)
	assert.Same(t, state, tryState)

	assert.True(t, catchRan, "catch body must run when the try body fails")
	assert.Equal(t, map[string]any{testConstHandledKey: true}, output)
}

// TestTryTaskBuilderExecSuccessSkipsCatch proves a successful try body returns
// its output unchanged and never invokes the catch body.
func TestTryTaskBuilderExecSuccessSkipsCatch(t *testing.T) {
	builder := newInlineTryBuilder("")

	state := utils.NewState()
	tryOutput := map[string]any{testConstValue: testConstOK}
	catchRan := false

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return tryOutput, nil
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		catchRan = true
		return nil, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineTry(t, fn, nil, state)
	require.NoError(t, execErr)

	assert.False(t, catchRan, "catch body must not run when the try body succeeds")
	assert.Equal(t, tryOutput, output, "successful try output must be returned unchanged")
}

// TestTryTaskBuilderExecWrapsCatchFailure proves a genuine catch-body failure
// is wrapped with the expected contextual error.
func TestTryTaskBuilderExecWrapsCatchFailure(t *testing.T) {
	builder := newInlineTryBuilder("")

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return nil, errors.New("boom from try")
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return nil, errors.New("boom from catch")
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineTry(t, fn, nil, utils.NewState())
	require.Error(t, execErr)
	assert.Nil(t, output)
	assert.Contains(t, execErr.Error(), "error running catch tasks")
	assert.Contains(t, execErr.Error(), "boom from catch")
}

// TestTryTaskBuilderExecPropagatesEndFromTryBody proves a `then: end`
// directive inside the try body (returned inline as flow.ErrEnd) is NOT
// treated as a catchable failure: the carried output survives and exec
// surfaces flow.ErrEnd without running the catch body.
func TestTryTaskBuilderExecPropagatesEndFromTryBody(t *testing.T) {
	builder := newInlineTryBuilder("")

	tryOutput := map[string]any{testConstValue: "end-time-output"}
	catchRan := false

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return tryOutput, flow.ErrEnd
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		catchRan = true
		return nil, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineTry(t, fn, nil, utils.NewState())

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd), "try body end must surface as flow.ErrEnd")
	assert.Equal(t, tryOutput, output, "carried output must be preserved")
	assert.False(t, catchRan, "catch body must not run when the try body signalled end")
}

// TestTryTaskBuilderExecPropagatesEndFromCatchBody is the symmetric case:
// when the try body fails for a real reason and the catch body itself emits
// `then: end` (inline flow.ErrEnd), that end must propagate as flow.ErrEnd
// rather than being wrapped as a generic catch failure.
func TestTryTaskBuilderExecPropagatesEndFromCatchBody(t *testing.T) {
	builder := newInlineTryBuilder("")

	catchOutput := map[string]any{testConstValue: "catch-end-output"}

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return nil, errors.New("boom from try")
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return catchOutput, flow.ErrEnd
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineTry(t, fn, nil, utils.NewState())

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd), "catch body end must surface as flow.ErrEnd")
	assert.Equal(t, catchOutput, output, "carried output must be preserved")
	assert.NotContains(t, execErr.Error(), "error running catch tasks",
		"catch-emitted end must not be wrapped as a catch failure")
}

// TestTryTaskBuilderExecPropagatesEncodedEndCompat proves the retained
// backwards-compatibility path: an encoded Temporal end error (as produced by
// flow.NewEndApplicationError) is still recognised, its carried payload output
// preserved, and the catch body skipped. This is not the primary inline path.
func TestTryTaskBuilderExecPropagatesEncodedEndCompat(t *testing.T) {
	builder := newInlineTryBuilder("")

	encodedOutput := map[string]any{testConstValue: "encoded-end-output"}
	catchRan := false

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return nil, flow.NewEndApplicationError(encodedOutput)
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		catchRan = true
		return nil, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineTry(t, fn, nil, utils.NewState())

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd), "encoded end must surface as flow.ErrEnd")
	assert.Equal(t, encodedOutput, output, "encoded end payload output must be preserved")
	assert.False(t, catchRan, "catch body must not run when the try body signalled end")
}

// runCatchAndCaptureState executes a try task whose try body fails, then
// returns the $data the catch body actually observed alongside the parent
// state exec was handed and the input the catch body received. The catch body
// records what it sees into closure-captured variables so tests can assert the
// exact caught-error contract exposed under $data, all under inline execution.
func runCatchAndCaptureState(t *testing.T, catchAs string, tryErr error) (
	caughtData map[string]any, caughtInput any, parentState *utils.State,
) {
	t.Helper()

	builder := newInlineTryBuilder(catchAs)

	parentState = utils.NewState()
	parentState.Input = map[string]any{"seed": "input"}

	tryFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return nil, tryErr
	}
	catchFn := func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		caughtData = st.Data
		caughtInput = input
		return map[string]any{testConstHandledKey: true}, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	_, execErr := runInlineTry(t, fn, parentState.Input, parentState)
	require.NoError(t, execErr)

	return caughtData, caughtInput, parentState
}

// TestTryTaskBuilderExecExposesErrorUnderDefaultKey proves the catch body sees
// the caught error under $data.error when catch.as is unset, and that inline
// execution no longer decorates it with childWorkflow metadata.
func TestTryTaskBuilderExecExposesErrorUnderDefaultKey(t *testing.T) {
	caughtData, caughtInput, _ := runCatchAndCaptureState(t, "", temporal.NewApplicationError("kaboom", "MyAppError"))

	caughtErr, ok := caughtData["error"].(map[string]any)
	require.True(t, ok, "catch body must see the caught error under $data.error")

	assert.Equal(t, "MyAppError", caughtErr["type"])
	assert.Equal(t, "kaboom", caughtErr["message"])

	// Execution is now inline, so there is no child workflow boundary and no
	// childWorkflow metadata to expose.
	assert.NotContains(t, caughtErr, "childWorkflow",
		"inline execution must not attach childWorkflow metadata")

	// The catch body must receive the cloned catch state's input.
	assert.Equal(t, map[string]any{"seed": "input"}, caughtInput)
}

// TestTryTaskBuilderExecExposesErrorUnderCustomKey proves the catch body sees
// the caught error under $data.<catch.as> when configured, and that the
// default "error" key is not used in that case.
func TestTryTaskBuilderExecExposesErrorUnderCustomKey(t *testing.T) {
	const customKey = "failure"

	caughtData, _, _ := runCatchAndCaptureState(t, customKey, temporal.NewApplicationError("kaboom", "MyAppError"))

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
	_, _, parentState := runCatchAndCaptureState(t, "", temporal.NewApplicationError("kaboom", "MyAppError"))

	assert.NotContains(t, parentState.Data, "error",
		"caught error must not leak back into the parent state after catch completes")
	assert.Empty(t, parentState.Data, "parent state data must be untouched by catch error injection")
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

		// Inline execution means no child workflow boundary is crossed.
		assert.NotContains(t, out, "childWorkflow")
	})

	t.Run("retryable application error without details", func(t *testing.T) {
		appErr := temporal.NewApplicationError("transient", "TransientError")

		out := tb.buildCatchError(appErr)

		assert.Equal(t, "TransientError", out["type"])
		assert.Equal(t, "transient", out["message"])
		assert.Equal(t, false, out["nonRetryable"])
		assert.NotContains(t, out, "details")
	})

	t.Run("plain Go error falls back to message", func(t *testing.T) {
		out := tb.buildCatchError(errors.New("plain"))

		// Inline execution surfaces plain Go errors more often; rather than an
		// empty map, buildCatchError exposes at least the message so the catch
		// tasks always have something interrogable under $data.
		assert.Equal(t, "plain", out["message"])
	})
}

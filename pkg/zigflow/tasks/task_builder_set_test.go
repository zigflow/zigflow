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
	"reflect"
	"strings"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	setTaskName  = "set-task"
	setValueA    = "value-a"
	setValueB    = "value-b"
	setExprEnvA  = "${ $env.A }"
	setExprEnvB  = "${ $env.B }"
	setLiteral   = "literal"
	setKeyOuter  = "outer"
	setKeyList   = "list"
	setKeyNested = "nested"
)

func TestSetTaskBuilderBuild(t *testing.T) {
	task := &model.SetTask{
		Set: model.NewObjectOrRuntimeExpr(map[string]any{
			testConstResult: map[string]any{
				testConstValue: "${ $env.VALUE }",
			},
		}),
	}

	builder, err := NewSetTaskBuilder(nil, task, setTaskName, nil, testEvents, nil)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	state.Env["VALUE"] = testConstOK

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: setTaskName})

	env.ExecuteWorkflow(setTaskName)
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	expected := map[string]any{
		testConstResult: map[string]any{
			testConstValue: testConstOK,
		},
	}

	assert.Equal(t, expected, result)
	assert.Equal(t, expected[testConstResult], state.Data[testConstResult])
}

func TestSetTaskBuilderBuildNonObject(t *testing.T) {
	task := &model.SetTask{
		Set: model.NewObjectOrRuntimeExpr("${ \"not-object\" }"),
	}

	builder, err := NewSetTaskBuilder(nil, task, setTaskName, nil, testEvents, nil)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: setTaskName})

	env.ExecuteWorkflow(setTaskName)

	err = env.GetWorkflowError()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "set must evaluate to an object")
}

// runSetTask executes a set task with the given definition in the Temporal
// test environment and returns the workflow result and error.
func runSetTask(t *testing.T, set *model.ObjectOrRuntimeExpr, state *utils.State) (map[string]any, error) {
	t.Helper()

	builder, err := NewSetTaskBuilder(nil, &model.SetTask{Set: set}, setTaskName, nil, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: setTaskName})

	env.ExecuteWorkflow(setTaskName)
	if err := env.GetWorkflowError(); err != nil {
		return nil, err
	}

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))

	return result, nil
}

func TestSetTaskBuilderBuildMultipleExpressionFields(t *testing.T) {
	state := utils.NewState()
	state.Env["A"] = setValueA
	state.Env["B"] = setValueB
	state.Env["C"] = "value-c"

	result, err := runSetTask(t, model.NewObjectOrRuntimeExpr(map[string]any{
		"a":        setExprEnvA,
		"b":        setExprEnvB,
		"c":        "${ $env.C }",
		setLiteral: "unchanged",
	}), state)
	require.NoError(t, err)

	expected := map[string]any{
		"a":        setValueA,
		"b":        setValueB,
		"c":        "value-c",
		setLiteral: "unchanged",
	}
	assert.Equal(t, expected, result)
	assert.Equal(t, expected, state.Data)
}

func TestSetTaskBuilderBuildNestedExpressionFields(t *testing.T) {
	state := utils.NewState()
	state.Env["A"] = setValueA
	state.Env["B"] = setValueB
	state.Data["count"] = 2

	result, err := runSetTask(t, model.NewObjectOrRuntimeExpr(map[string]any{
		setKeyOuter: map[string]any{
			"inner": map[string]any{
				"a": setExprEnvA,
			},
			setKeyList: []any{
				setExprEnvB,
				map[string]any{"doubled": "${ $data.count * 2 }"},
				setLiteral,
			},
		},
	}), state)
	require.NoError(t, err)

	expectedOuter := map[string]any{
		"inner": map[string]any{"a": setValueA},
		setKeyList: []any{
			setValueB,
			map[string]any{"doubled": float64(4)},
			setLiteral,
		},
	}
	assert.Equal(t, map[string]any{setKeyOuter: expectedOuter}, result)
	assert.Equal(t, expectedOuter, state.Data[setKeyOuter])
	assert.EqualValues(t, 2, state.Data["count"], "existing state data should be preserved")
}

func TestSetTaskBuilderBuildWholeFieldExpressionObject(t *testing.T) {
	state := utils.NewState()
	state.Env["A"] = setValueA

	result, err := runSetTask(t, model.NewObjectOrRuntimeExpr(
		`${ { a: $env.A, nested: { list: [1, $env.A] } } }`,
	), state)
	require.NoError(t, err)

	expected := map[string]any{
		"a": setValueA,
		setKeyNested: map[string]any{
			setKeyList: []any{float64(1), setValueA},
		},
	}
	assert.Equal(t, expected, result)
	assert.Equal(t, setValueA, state.Data["a"])
	assert.Contains(t, state.Data, setKeyNested)
}

func TestSetTaskBuilderBuildWholeFieldExpressionNonObject(t *testing.T) {
	cases := map[string]string{
		"string": `${ "not-object" }`,
		"number": `${ 42 }`,
		"array":  `${ [1, 2] }`,
		"null":   `${ null }`,
	}

	for name, expr := range cases {
		t.Run(name, func(t *testing.T) {
			state := utils.NewState()

			_, err := runSetTask(t, model.NewObjectOrRuntimeExpr(expr), state)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "set must evaluate to an object")
			assert.Empty(t, state.Data, "state must not change when set fails")
		})
	}
}

func TestSetTaskBuilderBuildExpressionError(t *testing.T) {
	state := utils.NewState()
	state.Env["A"] = setValueA

	_, err := runSetTask(t, model.NewObjectOrRuntimeExpr(map[string]any{
		"valid":  setExprEnvA,
		"broken": `${ error("set-expression-failed") }`,
	}), state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error running runtime expression")
	assert.Contains(t, err.Error(), "set-expression-failed")
	assert.Empty(t, state.Data, "state must not change when set fails")
}

// sideEffectRecorder wraps the default data converter used by this project
// (every Zigflow codec wraps converter.GetDefaultDataConverter) and records
// the payload produced for the set task's SideEffect result, so replay tests
// use the exact marker data the implementation writes rather than a
// hand-crafted shape.
type sideEffectRecorder struct {
	converter.DataConverter
	markers []*commonpb.Payloads
}

func newSideEffectRecorder() *sideEffectRecorder {
	return &sideEffectRecorder{DataConverter: converter.GetDefaultDataConverter()}
}

func (r *sideEffectRecorder) ToPayloads(values ...any) (*commonpb.Payloads, error) {
	p, err := r.DataConverter.ToPayloads(values...)
	// The set SideEffect returns a struct carrying the result and error.
	if err == nil && len(values) == 1 {
		if v := reflect.ValueOf(values[0]); v.Kind() == reflect.Struct && v.FieldByName("Err").IsValid() {
			r.markers = append(r.markers, p)
		}
	}
	return p, err
}

// captureSetMarker runs a set task once in the Temporal test environment and
// returns the single SideEffect marker payload it recorded, with the task's
// error from that first execution.
func captureSetMarker(t *testing.T, set *model.ObjectOrRuntimeExpr, state *utils.State) (*commonpb.Payloads, error) {
	t.Helper()

	builder, err := NewSetTaskBuilder(nil, &model.SetTask{Set: set}, setTaskName, nil, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	recorder := newSideEffectRecorder()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(recorder)

	var taskErr error
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		out, err := fn(ctx, nil, state)
		taskErr = err
		return out, err
	}, workflow.RegisterOptions{Name: setTaskName})

	env.ExecuteWorkflow(setTaskName)
	require.Len(t, recorder.markers, 1, "set must record exactly one SideEffect")

	return recorder.markers[0], taskErr
}

// replaySetTask replays the given history through the real workflow replayer
// and returns the set task's output, state and error from the replay, along
// with the replayer's own error.
func replaySetTask(
	t *testing.T, set *model.ObjectOrRuntimeExpr, state *utils.State, history *historypb.History,
) (output any, taskErr, replayErr error) {
	t.Helper()

	builder, err := NewSetTaskBuilder(nil, &model.SetTask{Set: set}, setTaskName, nil, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		output, taskErr = fn(ctx, nil, state)
		return output, taskErr
	}, workflow.RegisterOptions{Name: setReplayWorkflowName})

	replayErr = replayer.ReplayWorkflowHistory(nil, history)
	return output, taskErr, replayErr
}

// TestSetTaskBuilderReplayUsesSingleRecordedSideEffect replays a history
// through the real workflow replayer. The history holds exactly one
// SideEffect marker, captured from a first execution whose env values differ
// from the replay's, so the test proves that:
//
//   - the whole set object is evaluated in one SideEffect, regardless of how
//     many expression fields it contains (the replayer fails if the workflow
//     issues more or fewer SideEffect commands than the history records)
//   - on replay every value comes from the recorded marker and stays under
//     its intended key, without depending on Go map iteration order
func TestSetTaskBuilderReplayUsesSingleRecordedSideEffect(t *testing.T) {
	set := model.NewObjectOrRuntimeExpr(map[string]any{
		"a": "${ $env.A }",
		"b": "${ $env.B }",
		"c": "${ $env.C }",
		setKeyNested: map[string]any{
			"d":        "${ $env.D }",
			setKeyList: []any{"${ $env.E }", setLiteral},
		},
	})

	newState := func(prefix string) *utils.State {
		state := utils.NewState()
		for _, k := range []string{"A", "B", "C", "D", "E"} {
			state.Env[k] = prefix + strings.ToLower(k)
		}
		return state
	}

	marker, err := captureSetMarker(t, set, newState("recorded-"))
	require.NoError(t, err)

	recorded := map[string]any{
		"a": "recorded-a",
		"b": "recorded-b",
		"c": "recorded-c",
		setKeyNested: map[string]any{
			"d":        "recorded-d",
			setKeyList: []any{"recorded-e", setLiteral},
		},
	}

	t.Run("one marker replays with recorded values", func(t *testing.T) {
		// Distinct live values, so a replay that re-evaluated the
		// expressions (or mixed up keys) would produce a different result.
		state := newState("live-")

		output, taskErr, replayErr := replaySetTask(t, set, state, setTaskHistory(t, false, marker))
		require.NoError(t, replayErr)
		require.NoError(t, taskErr)

		assert.Equal(t, recorded, output)
		for k, v := range recorded {
			assert.Equal(t, v, state.Data[k], "state.Data[%s]", k)
		}
	})

	// Guard that the replayer really enforces the marker count, so the
	// success case above is not vacuous. This is also the shape of a history
	// recorded by the previous one-SideEffect-per-field implementation.
	t.Run("per-field markers fail replay", func(t *testing.T) {
		_, _, replayErr := replaySetTask(t, set, newState("live-"), setTaskHistory(t, false, marker, marker))
		require.Error(t, replayErr)
		assert.Contains(t, replayErr.Error(), "missing replay command for MarkerRecorded")
	})
}

// TestSetTaskBuilderReplayFailedExpression checks a set task whose expression
// fails, both inside try/catch on first execution and on history replay.
//
// On replay the SideEffect closure does not run, so the error must come from
// the recorded marker. The replayer does not compare failure messages, so
// the test captures the task's error during replay and compares it with the
// first execution. The parent try task reads the child failure from its own
// history, so the catch branch receives what the try child originally
// failed with; the replayed try child must fail with that same error.
func TestSetTaskBuilderReplayFailedExpression(t *testing.T) {
	const expressionErr = "set-expression-failed"

	set := model.NewObjectOrRuntimeExpr(map[string]any{
		"valid":  setExprEnvA,
		"broken": `${ error("` + expressionErr + `") }`,
	})

	newState := func() *utils.State {
		state := utils.NewState()
		state.Env["A"] = setValueA
		return state
	}

	// First execution: the set task runs as the try child and the catch
	// child captures the error it is given.
	setBuilder, err := NewSetTaskBuilder(nil, &model.SetTask{Set: set}, setTaskName, nil, testEvents, nil)
	require.NoError(t, err)
	setFn, err := setBuilder.Build()
	require.NoError(t, err)

	tryBuilder := &TryTaskBuilder{
		name: "try-set",
		task: &model.TryTask{
			Try:   &model.TaskList{},
			Catch: &model.TryTaskCatch{Do: &model.TaskList{}},
		},
		tryChildWorkflowName:   "try-set-child",
		catchChildWorkflowName: "catch-set-child",
	}
	tryFn, err := tryBuilder.exec()
	require.NoError(t, err)

	recorder := newSideEffectRecorder()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(recorder)

	var firstErr error
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		out, err := setFn(ctx, input, st)
		firstErr = err
		return out, err
	}, workflow.RegisterOptions{Name: tryBuilder.tryChildWorkflowName})

	var caught map[string]any
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		caught, _ = st.Data["error"].(map[string]any)
		return map[string]any{testConstHandledKey: true}, nil
	}, workflow.RegisterOptions{Name: tryBuilder.catchChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return tryFn(ctx, nil, newState())
	}, workflow.RegisterOptions{Name: "try-set"})

	env.ExecuteWorkflow("try-set")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, map[string]any{testConstHandledKey: true}, result, "catch should handle the failed set")

	require.Error(t, firstErr)
	require.Len(t, recorder.markers, 1, "a failed set must still record exactly one SideEffect")
	require.NotNil(t, caught, "catch should receive the error under $data.error")
	assert.Contains(t, caught[testConstMessage], expressionErr, "catch should see the original expression error")

	// Replay the try child from a history holding the recorded marker.
	state := newState()
	_, replayTaskErr, replayErr := replaySetTask(t, set, state, setTaskHistory(t, true, recorder.markers[0]))
	require.NoError(t, replayErr)
	require.Error(t, replayTaskErr, "the replayed try child must still fail so catch behaviour is unchanged")
	assert.Equal(t, firstErr.Error(), replayTaskErr.Error(), "replay must reproduce the first execution error")
	assert.Contains(t, replayTaskErr.Error(), "error running runtime expression")
	assert.Contains(t, replayTaskErr.Error(), expressionErr)
	assert.Empty(t, state.Data, "state must not change when set fails on replay")
}

const setReplayWorkflowName = "set-replay"

// setTaskHistory builds a closed workflow history containing the given
// SideEffect marker payloads, in order. The workflow is recorded as failed
// when failed is true, otherwise as completed.
func setTaskHistory(t *testing.T, failed bool, markers ...*commonpb.Payloads) *historypb.History {
	t.Helper()

	dc := converter.GetDefaultDataConverter()
	taskQueue := &taskqueuepb.TaskQueue{Name: "set-replay-task-queue"}

	events := []*historypb.HistoryEvent{
		{
			EventId:   1,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					WorkflowType: &commonpb.WorkflowType{Name: setReplayWorkflowName},
					TaskQueue:    taskQueue,
				},
			},
		},
		{
			EventId:   2,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{
				WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
					TaskQueue: taskQueue,
				},
			},
		},
		{
			EventId:   3,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{
				WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{},
			},
		},
		{
			EventId:   4,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{
				WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{
					ScheduledEventId: 2,
					StartedEventId:   3,
				},
			},
		},
	}

	for i, data := range markers {
		// Side effect IDs start at 1, matching the SDK's counter.
		id, err := dc.ToPayloads(int64(i + 1))
		require.NoError(t, err)

		events = append(events, &historypb.HistoryEvent{
			EventId:   int64(len(events) + 1),
			EventType: enumspb.EVENT_TYPE_MARKER_RECORDED,
			Attributes: &historypb.HistoryEvent_MarkerRecordedEventAttributes{
				MarkerRecordedEventAttributes: &historypb.MarkerRecordedEventAttributes{
					MarkerName: "SideEffect",
					Details: map[string]*commonpb.Payloads{
						"side-effect-id": id,
						"data":           data,
					},
					WorkflowTaskCompletedEventId: 4,
				},
			},
		})
	}

	closing := &historypb.HistoryEvent{EventId: int64(len(events) + 1)}
	if failed {
		closing.EventType = enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED
		closing.Attributes = &historypb.HistoryEvent_WorkflowExecutionFailedEventAttributes{
			WorkflowExecutionFailedEventAttributes: &historypb.WorkflowExecutionFailedEventAttributes{
				WorkflowTaskCompletedEventId: 4,
			},
		}
	} else {
		closing.EventType = enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED
		closing.Attributes = &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{
			WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{
				WorkflowTaskCompletedEventId: 4,
			},
		}
	}

	return &historypb.History{Events: append(events, closing)}
}

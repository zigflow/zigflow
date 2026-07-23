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
	"fmt"
	"math"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func testInlineForBody(fn func(*utils.State) (any, error)) TemporalWorkflowFunc {
	return func(_ workflow.Context, _ any, state *utils.State) (any, error) {
		output, err := fn(state)
		state.Output = output
		return output, err
	}
}

func TestForTaskBuilderAddIterationResult(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
		response any
	}{
		{
			name:     "adds string response to state data",
			taskName: "my-task",
			response: "some-result",
		},
		{
			name:     "adds map response to state data",
			taskName: "map-task",
			response: map[string]any{"key": testConstValue},
		},
		{
			name:     "adds nil response to state data",
			taskName: "nil-task",
			response: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := utils.NewState()

			b := &ForTaskBuilder{
				builder: builder[*model.ForTask]{
					doc:          testWorkflow,
					eventEmitter: testEvents,
					name:         tc.taskName,
					task: &model.ForTask{
						For: model.ForTaskConfiguration{In: testConstForDataItems},
						Do:  &model.TaskList{},
					},
				},
			}

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			workflowName := "add-iteration-" + tc.name
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
				b.addIterationResult(ctx, state, tc.response)
				return nil
			}, workflow.RegisterOptions{Name: workflowName})

			env.ExecuteWorkflow(workflowName)
			assert.NoError(t, env.GetWorkflowError())

			assert.Equal(t, tc.response, state.Data[tc.taskName])
		})
	}
}

func TestForTaskBuilderCheckWhile(t *testing.T) {
	tests := []struct {
		name        string
		while       string
		stateData   map[string]any
		expect      bool
		expectError bool
	}{
		{
			name:   "empty while defaults to true",
			expect: true,
		},
		{
			name:  "boolean true expression",
			while: testConstDataFlag,
			stateData: map[string]any{
				testConstFlag: true,
			},
			expect: true,
		},
		{
			name:  "boolean false expression",
			while: testConstDataFlag,
			stateData: map[string]any{
				testConstFlag: false,
			},
			expect: false,
		},
		{
			name:  "non boolean resolves to false",
			while: "${ $data.text }",
			stateData: map[string]any{
				"text": "not-bool",
			},
			expect: false,
		},
		{
			name:        "invalid expression returns error",
			while:       "${ $data. }",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := utils.NewState()
			state.AddData(tc.stateData)

			builder := &ForTaskBuilder{
				builder: builder[*model.ForTask]{
					eventEmitter: testEvents,
					name:         "for-task",
					task: &model.ForTask{
						For:   model.ForTaskConfiguration{In: testConstForDataItems},
						While: tc.while,
						Do:    &model.TaskList{},
					},
				},
			}

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			workflowName := "check-" + tc.name
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (bool, error) {
				return builder.checkWhile(ctx, state)
			}, workflow.RegisterOptions{Name: workflowName})

			env.ExecuteWorkflow(workflowName)

			err := env.GetWorkflowError()
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			var res bool
			assert.NoError(t, env.GetWorkflowResult(&res))
			assert.Equal(t, tc.expect, res)
		})
	}
}

func TestForTaskBuilderIterationCount(t *testing.T) {
	tests := []struct {
		name        string
		data        any
		wantCount   int
		wantOK      bool
		wantErr     bool
		errContains string
	}{
		{
			name:      "int passes through unchanged",
			data:      5,
			wantCount: 5,
			wantOK:    true,
		},
		{
			name:      "whole-number float64 is accepted",
			data:      float64(5),
			wantCount: 5,
			wantOK:    true,
		},
		{
			name:        "fractional float64 is rejected",
			data:        5.5,
			wantOK:      true,
			wantErr:     true,
			errContains: "whole number",
		},
		{
			name:        "NaN is rejected",
			data:        math.NaN(),
			wantOK:      true,
			wantErr:     true,
			errContains: "NaN",
		},
		{
			name:        "positive infinity is rejected",
			data:        math.Inf(1),
			wantOK:      true,
			wantErr:     true,
			errContains: "infinite",
		},
		{
			name:        "negative infinity is rejected",
			data:        math.Inf(-1),
			wantOK:      true,
			wantErr:     true,
			errContains: "infinite",
		},
		{
			name:        "value above int range is rejected",
			data:        1e20,
			wantOK:      true,
			wantErr:     true,
			errContains: "out of int range",
		},
		{
			name:        "value below int range is rejected",
			data:        -1e20,
			wantOK:      true,
			wantErr:     true,
			errContains: "out of int range",
		},
		{
			name:   "non-numeric value is not numeric",
			data:   "not-a-number",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &ForTaskBuilder{}
			count, ok, err := b.iterationCount(tc.data)

			assert.Equal(t, tc.wantOK, ok)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.wantCount, count)
		})
	}
}

func TestForTaskBuilderIterator(t *testing.T) {
	state := utils.NewState()
	state.Input = map[string]any{
		testConstRequestID: "abc",
	}

	builder := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "iterate",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstValue,
					At:   testConstIdx,
					In:   testConstForDataItems,
				},
				Do: &model.TaskList{
					&model.TaskItem{Key: "first", Task: &model.DoTask{}},
				},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			return map[string]any{testConstChildValue: st.Data[testConstValue]}, nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state.AddData(map[string]any{
		testConstItems: []any{testConstItemValue},
	})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return builder.iterator(ctx, 0, testConstItemValue, state)
	}, workflow.RegisterOptions{Name: "iterator-test"})

	env.ExecuteWorkflow("iterator-test")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, map[string]any{testConstChildValue: testConstItemValue}, result)
	// iterator() propagates the child's output back onto the working state (the
	// state passed as the workingState parameter, which in this test is state itself).
	assert.Equal(t, map[string]any{testConstChildValue: testConstItemValue}, state.Output)
	// Loop-local variables must not appear on the state passed to iterator()
	// because they are placed on a per-iteration clone (iterState) inside iterator().
	assert.Nil(t, state.Data[testConstValue])
	assert.Nil(t, state.Data[testConstIdx])
}

func TestForTaskBuilderBuildRunsIterationsInline(t *testing.T) {
	events, readEventTypes := newRecordingEvents(t)
	task := &model.ForTask{
		For: model.ForTaskConfiguration{In: testConstForRefDataItems},
		Do: &model.TaskList{
			&model.TaskItem{
				Key: testConstStep,
				Task: &model.SetTask{
					Set: model.NewObjectOrRuntimeExpr(map[string]any{
						testConstProcessed: "${ $data.item }",
					}),
				},
			},
		},
	}

	builder, err := NewForTaskBuilder(nil, task, "inline-for", testWorkflow, events, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"a", "b", "c"}})

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	childStarts := 0
	env.SetOnChildWorkflowStartedListener(
		func(*workflow.Info, workflow.Context, converter.EncodedValues) {
			childStarts++
		},
	)
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "inline-for-outer"})

	env.ExecuteWorkflow("inline-for-outer")
	require.NoError(t, env.GetWorkflowError())
	assert.Zero(t, childStarts)

	var result []any
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Len(t, result, 3)

	eventTypes := readEventTypes()
	assert.Contains(t, eventTypes, "dev.zigflow.task.started")
	assert.Contains(t, eventTypes, "dev.zigflow.task.completed")
	assert.Contains(t, eventTypes, "dev.zigflow.iteration.completed")
	assert.NotContains(t, eventTypes, "dev.zigflow.workflow.started")
	assert.NotContains(t, eventTypes, "dev.zigflow.workflow.completed")
}

// TestForIteratorContextPropagates verifies that $context set by an export in
// iteration N is visible as state.Context when iterator() is called for N+1.
func TestForIteratorContextPropagates(t *testing.T) {
	var receivedContexts []any

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "ctx-prop",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{Each: constDefaultItemVar, At: testConstIdx, In: testConstForDataItems},
				Do:  &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			receivedContexts = append(receivedContexts, st.Context)
			st.Context = map[string]any{testConstLast: st.Data[constDefaultItemVar]}
			return st.Data[constDefaultItemVar], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		if _, err := b.iterator(ctx, 0, "alpha", state); err != nil {
			return err
		}
		if _, err := b.iterator(ctx, 1, "beta", state); err != nil {
			return err
		}
		return nil
	}, workflow.RegisterOptions{Name: "ctx-prop-outer"})

	env.ExecuteWorkflow("ctx-prop-outer")
	assert.NoError(t, env.GetWorkflowError())

	assert.Len(t, receivedContexts, 2)
	// First iteration starts with no exported context.
	assert.Nil(t, receivedContexts[0])
	// Second iteration sees the context exported by the first iteration.
	assert.Equal(t, map[string]any{testConstLast: "alpha"}, receivedContexts[1])
	// Parent state ends with the context from the final iteration.
	assert.Equal(t, map[string]any{testConstLast: "beta"}, state.Context)
}

// TestForIteratorWhileSeesOutput verifies that the while condition for iteration
// N+1 sees the output produced by iteration N.
func TestForIteratorWhileSeesOutput(t *testing.T) {
	callCount := 0

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "while-out",
			task: &model.ForTask{
				// Continue while $output.continue is true.
				While: "${ $output.continue }",
				For:   model.ForTaskConfiguration{Each: constDefaultItemVar, At: testConstIdx, In: testConstForDataItems},
				Do:    &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(*utils.State) (any, error) {
			callCount++
			// The first iteration returns continue=false so the second iteration's
			// while check should stop the loop.
			return map[string]any{testConstFlowContinue: false}, nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	// Pre-seed output so the first while check (before any iteration runs) passes.
	// This also doubles as a check that the pre-loop $output is visible to while.
	state.Output = map[string]any{testConstFlowContinue: true}

	// stoppedAt tracks which iteration caused the loop to stop.
	// 0 = never stopped, 1 = stopped on first, 2 = stopped on second.
	stoppedAt := 0
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		// Iteration 0: state.Output.continue == true (pre-seeded), so while passes.
		res0, err := b.iterator(ctx, 0, "first", state)
		if err != nil {
			if errors.Is(err, errForkIterationStop) {
				stoppedAt = 1
				return nil
			}
			return err
		}
		_ = res0
		// Iteration 1: state.Output.continue == false (from iteration 0), so while stops.
		_, err = b.iterator(ctx, 1, "second", state)
		if err != nil {
			if errors.Is(err, errForkIterationStop) {
				stoppedAt = 2
				return nil
			}
			return err
		}
		return nil
	}, workflow.RegisterOptions{Name: "while-out-outer"})

	env.ExecuteWorkflow("while-out-outer")
	assert.NoError(t, env.GetWorkflowError())
	// The while condition stopped the loop at the second iteration call.
	assert.Equal(t, 2, stoppedAt)
	// The inline body was only invoked once (first iteration ran; second was stopped by while).
	assert.Equal(t, 1, callCount)
}

// TestForExecArrayAccumulatesResults verifies that the for task still returns an
// array of per-iteration results when iterating over an array for.in value.
func TestForExecArrayAccumulatesResults(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "accum",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: constDefaultItemVar,
					At:   testConstIdx,
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			return map[string]any{testConstProcessed: st.Data[constDefaultItemVar]}, nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"x", "y", "z"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "accum-outer"})

	env.ExecuteWorkflow("accum-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result []any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, []any{
		map[string]any{testConstProcessed: "x"},
		map[string]any{testConstProcessed: "y"},
		map[string]any{testConstProcessed: "z"},
	}, result)
}

// TestForExecObjectAccumulatesResults verifies the object for.in variant still
// returns a map keyed by the original object keys.
func TestForExecObjectAccumulatesResults(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "obj-accum",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstVal,
					At:   "key",
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			return st.Data[testConstVal], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: map[string]any{"a": 1, "b": 2}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "obj-accum-outer"})

	env.ExecuteWorkflow("obj-accum-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	// The workflow result still crosses Temporal's serialisation boundary.
	assert.Equal(t, map[string]any{"a": float64(1), "b": float64(2)}, result)
	// Inside the workflow, inline execution preserves the native integer values.
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, state.Output)
}

func TestForExecObjectUsesDeterministicKeyOrder(t *testing.T) {
	iterationOrder := make([]string, 0, 3)
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "obj-order",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstVal,
					At:   "key",
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			key := st.Data["key"].(string)
			iterationOrder = append(iterationOrder, key)
			return key, nil
		}),
	}

	state := utils.NewState()
	state.AddData(map[string]any{
		testConstItems: map[string]any{"z": 1, "a": 2, "m": 3},
	})
	execFn, err := b.exec()
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "obj-order-outer"})

	env.ExecuteWorkflow("obj-order-outer")
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, []string{"a", "m", "z"}, iterationOrder)
}

// TestForExecNumericAccumulatesResults verifies the numeric for.in variant still
// returns a slice with one entry per iteration index.
func TestForExecNumericAccumulatesResults(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "num-accum",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstVal,
					At:   testConstIdx,
					In:   testConstForRefDataCount,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			return st.Data[testConstIdx], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: 3})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "num-accum-outer"})

	env.ExecuteWorkflow("num-accum-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result []any
	assert.NoError(t, env.GetWorkflowResult(&result))

	// The workflow result still crosses Temporal's serialisation boundary.
	assert.Equal(t, []any{float64(0), float64(1), float64(2)}, result)
	// Inside the workflow, inline execution preserves the native integer values.
	assert.Equal(t, []any{0, 1, 2}, state.Output)
}

// TestForExecNumericFloat64Whole verifies that the numeric for.in variant
// accepts a float64 value that represents a whole number (for example 5.0)
// and iterates the expected number of times. Variables resolved through jq
// may arrive as float64 after a JSON round trip even when they were authored
// as integer literals, so this case must be supported.
func TestForExecNumericFloat64Whole(t *testing.T) {
	callCount := 0
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "num-float64-whole",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstVal,
					At:   testConstIdx,
					In:   testConstForRefDataCount,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			callCount++
			return st.Data[testConstIdx], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	// A float64 whole number must be accepted and treated as an iteration count.
	state.AddData(map[string]any{testConstCount: float64(5)})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "num-float64-whole-outer"})

	env.ExecuteWorkflow("num-float64-whole-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result []any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, []any{float64(0), float64(1), float64(2), float64(3), float64(4)}, result)
	assert.Equal(t, []any{0, 1, 2, 3, 4}, state.Output)
	assert.Equal(t, 5, callCount, "inline body must be invoked once per iteration")
}

// TestForExecNumericFloat64Zero verifies that a float64 zero results in no
// iterations rather than an error. Zero is a whole number so the trunc check
// must accept it.
func TestForExecNumericFloat64Zero(t *testing.T) {
	callCount := 0
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "num-float64-zero",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstVal,
					At:   testConstIdx,
					In:   testConstForRefDataCount,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			callCount++
			return st.Data[testConstIdx], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: float64(0)})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "num-float64-zero-outer"})

	env.ExecuteWorkflow("num-float64-zero-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result []any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, []any{}, result, "zero count must produce no iterations")
	assert.Equal(t, 0, callCount, "inline body must not be invoked when count is zero")
}

// TestForExecNumericFloat64Fractional verifies that a for.in expression
// resolving to a non-whole-number float64 fails with a clear, actionable
// error that names the offending value. Silent truncation would violate the
// determinism and explicit-validation principles of the engine.
func TestForExecNumericFloat64Fractional(t *testing.T) {
	callCount := 0
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "num-float64-frac",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: testConstVal,
					At:   testConstIdx,
					In:   testConstForRefDataCount,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			callCount++
			return st.Data[testConstIdx], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: 5.5})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "num-float64-frac-outer"})

	env.ExecuteWorkflow("num-float64-frac-outer")

	wfErr := env.GetWorkflowError()
	assert.Error(t, wfErr)
	assert.Contains(t, wfErr.Error(), "for task numeric iteration value must be a whole number")
	assert.Contains(t, wfErr.Error(), "5.5")
	assert.Equal(t, 0, callCount, "no iterations must run when the for.in value is invalid")
}

// TestForExecLoopVarsDoNotLeakToParent verifies that loop-local variables
// (item, index or custom names) are not present in the parent state's Data
// after the for loop completes.
func TestForExecLoopVarsDoNotLeakToParent(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "leak-check",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: constDefaultItemVar,
					At:   testConstIdx,
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			return st.Data[constDefaultItemVar], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"a", "b"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "leak-check-outer"})

	env.ExecuteWorkflow("leak-check-outer")
	assert.NoError(t, env.GetWorkflowError())

	// Loop-local variables must not appear in parent state after the loop exits.
	assert.Nil(t, state.Data[constDefaultItemVar], "item must not leak to parent state")
	assert.Nil(t, state.Data[testConstIdx], "idx must not leak to parent state")
	// The pre-loop data must be unmodified.
	assert.Equal(t, []any{"a", "b"}, state.Data[testConstItems])
}

// TestForExecContextDoesNotLeakToParent verifies that $context updated by an
// export inside the loop does NOT propagate to the parent state after exec().
// workingState.Context is loop-private and must not overwrite the surrounding
// workflow context. Loop-internal Data changes must also not appear in parent Data.
func TestForExecContextDoesNotLeakToParent(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "ctx-leak",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: constDefaultItemVar,
					At:   testConstIdx,
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(st *utils.State) (any, error) {
			st.Context = map[string]any{testConstLast: st.Data[constDefaultItemVar]}
			return st.Data[constDefaultItemVar], nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"x", "y"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "ctx-leak-outer"})

	env.ExecuteWorkflow("ctx-leak-outer")
	assert.NoError(t, env.GetWorkflowError())

	// workingState.Context is loop-private and must NOT be copied to state.Context.
	assert.Nil(t, state.Context, "loop-private context must not leak to parent state")
	// Loop-internal Data (e.g. taskName -> lastResult set by addIterationResult
	// on workingState) must not appear in the parent state's Data.
	assert.Nil(t, state.Data["ctx-leak"], "accumulated iteration result must not leak to parent Data")
}

// TestForExecOutputIsAggregatedResult verifies that exec() sets state.Output to
// the aggregated per-iteration result (array or object), not to the last
// iteration's internal $output. The per-iteration $output lives only in
// workingState and is used solely for while evaluation between iterations.
func TestForExecOutputIsAggregatedResult(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "no-out-leak",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: constDefaultItemVar,
					At:   testConstIdx,
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(*utils.State) (any, error) {
			return "iteration-output", nil
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"a"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "no-out-leak-outer"})

	env.ExecuteWorkflow("no-out-leak-outer")
	assert.NoError(t, env.GetWorkflowError())

	// exec() sets state.Output to the aggregated loop result, not the last
	// iteration's internal $output. The per-iteration $output lives only in
	// workingState and is used solely for while evaluation between iterations.
	assert.Equal(t, []any{"iteration-output"}, state.Output,
		"exec() must set state.Output to the aggregated loop result")
}

// TestForExecErrorLeavesParentStateUnchanged verifies that when an iteration
// fails the parent state is not modified at all. Context propagation only
// happens after a clean loop exit so that retries and catch handlers see
// the original state.
func TestForExecErrorLeavesParentStateUnchanged(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "err-unchanged",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: constDefaultItemVar,
					At:   testConstIdx,
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(*utils.State) (any, error) {
			return nil, fmt.Errorf("simulated iteration failure")
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.Context = map[string]any{"original": true}
	state.Output = "original-output"
	state.AddData(map[string]any{testConstItems: []any{"a"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "err-unchanged-outer"})

	env.ExecuteWorkflow("err-unchanged-outer")
	// The workflow must have failed due to the body error.
	assert.Error(t, env.GetWorkflowError())

	// Context, Output, and Data must all be unchanged.
	assert.Equal(t, map[string]any{"original": true}, state.Context,
		"exec() must not modify state.Context when an iteration fails")
	assert.Equal(t, "original-output", state.Output,
		"exec() must not modify state.Output when an iteration fails")
	assert.Equal(t, []any{"a"}, state.Data[testConstItems],
		"pre-loop Data must be unchanged when an iteration fails")
	assert.Nil(t, state.Data["err-unchanged"],
		"loop task result must not appear in parent Data when an iteration fails")
	assert.Nil(t, state.Data[constDefaultItemVar],
		"loop-local variable must not leak into parent Data when an iteration fails")
	assert.Nil(t, state.Data[testConstIdx],
		"loop-local variable must not leak into parent Data when an iteration fails")
}

// TestForIteratorBodyEndsPropagatesErrEnd proves that an inline for-loop body
// emitting `then: end` causes the for-task to surface flow.ErrEnd unchanged.
func TestForIteratorBodyEndsPropagatesErrEnd(t *testing.T) {
	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "iter-end",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: constDefaultItemVar,
					At:   testConstIdx,
					In:   testConstForRefDataItems,
				},
				Do: &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
		body: testInlineForBody(func(*utils.State) (any, error) {
			return "final-output", flow.ErrEnd
		}),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"only-item"}})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return b.iterator(ctx, 0, "only-item", state)
	}, workflow.RegisterOptions{Name: "iter-end-outer"})

	env.ExecuteWorkflow("iter-end-outer")

	err := env.GetWorkflowError()
	require.Error(t, err)
	assert.Contains(t, err.Error(), flow.ErrEnd.Error())
	assert.Equal(t, "final-output", state.Output)
}

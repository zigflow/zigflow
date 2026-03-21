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
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

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
			response: map[string]any{"key": "value"},
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
						For: model.ForTaskConfiguration{In: "${ .data.items }"},
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
			while: "${ $data.flag }",
			stateData: map[string]any{
				"flag": true,
			},
			expect: true,
		},
		{
			name:  "boolean false expression",
			while: "${ $data.flag }",
			stateData: map[string]any{
				"flag": false,
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
						For:   model.ForTaskConfiguration{In: "${ .data.items }"},
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

func TestForTaskBuilderIterator(t *testing.T) {
	state := utils.NewState()
	state.Input = map[string]any{
		"request_id": "abc",
	}

	builder := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "iterate",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "value",
					At:   "idx",
					In:   "${ .data.items }",
				},
				Do: &model.TaskList{
					&model.TaskItem{Key: "first", Task: &model.DoTask{}},
				},
			},
		},
		childWorkflowName: utils.GenerateChildWorkflowName("for", "iterate"),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
		return forChildResult{
			Output:  map[string]any{"child_value": st.Data["value"]},
			Context: nil,
		}, nil
	}, workflow.RegisterOptions{Name: builder.childWorkflowName})

	state.AddData(map[string]any{
		"items": []any{"item-value"},
	})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return builder.iterator(ctx, 0, "item-value", state)
	}, workflow.RegisterOptions{Name: "iterator-test"})

	env.ExecuteWorkflow("iterator-test")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, map[string]any{"child_value": "item-value"}, result)
	// iterator() propagates the child's output back onto the working state (the
	// state passed as the workingState parameter, which in this test is state itself).
	assert.Equal(t, map[string]any{"child_value": "item-value"}, state.Output)
	// Loop-local variables must not appear on the state passed to iterator()
	// because they are placed on a per-iteration clone (iterState) inside iterator().
	assert.Nil(t, state.Data["value"])
	assert.Nil(t, state.Data["idx"])
}

// TestForIteratorContextPropagates verifies that $context set by an export in
// iteration N is visible as state.Context when iterator() is called for N+1.
func TestForIteratorContextPropagates(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "ctx-prop")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "ctx-prop",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{Each: "item", At: "idx", In: "${ .data.items }"},
				Do:  &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	// The mock returns a context that records which items have been visited.
	// The second invocation receives the context set by the first invocation.
	var receivedContexts []any
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			receivedContexts = append(receivedContexts, st.Context)
			return forChildResult{
				Output:  st.Data["item"],
				Context: map[string]any{"last": st.Data["item"]},
			}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

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
	assert.Equal(t, map[string]any{"last": "alpha"}, receivedContexts[1])
	// Parent state ends with the context from the final iteration.
	assert.Equal(t, map[string]any{"last": "beta"}, state.Context)
}

// TestForIteratorWhileSeesOutput verifies that the while condition for iteration
// N+1 sees the output produced by iteration N.
func TestForIteratorWhileSeesOutput(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "while-out")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "while-out",
			task: &model.ForTask{
				// Continue while $output.continue is true.
				While: "${ $output.continue }",
				For:   model.ForTaskConfiguration{Each: "item", At: "idx", In: "${ .data.items }"},
				Do:    &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	callCount := 0
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			callCount++
			// The first iteration returns continue=false so the second iteration's
			// while check should stop the loop.
			return forChildResult{Output: map[string]any{"continue": false}, Context: nil}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	// Pre-seed output so the first while check (before any iteration runs) passes.
	// This also doubles as a check that the pre-loop $output is visible to while.
	state.Output = map[string]any{"continue": true}

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
	// Child workflow was only invoked once (first iteration ran; second was stopped by while).
	assert.Equal(t, 1, callCount)
}

// TestForExecArrayAccumulatesResults verifies that the for task still returns an
// array of per-iteration results when iterating over an array for.in value.
func TestForExecArrayAccumulatesResults(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "accum")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "accum",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "item",
					At:   "idx",
					In:   "${ $data.items }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{
				Output:  map[string]any{"processed": st.Data["item"]},
				Context: nil,
			}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.AddData(map[string]any{"items": []any{"x", "y", "z"}})

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
		map[string]any{"processed": "x"},
		map[string]any{"processed": "y"},
		map[string]any{"processed": "z"},
	}, result)
}

// TestForExecObjectAccumulatesResults verifies the object for.in variant still
// returns a map keyed by the original object keys.
func TestForExecObjectAccumulatesResults(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "obj-accum")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "obj-accum",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "val",
					At:   "key",
					In:   "${ $data.items }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{
				Output:  st.Data["val"],
				Context: nil,
			}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.AddData(map[string]any{"items": map[string]any{"a": 1, "b": 2}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "obj-accum-outer"})

	env.ExecuteWorkflow("obj-accum-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	// JSON round-trip through the Temporal child workflow boundary converts integers to float64.
	assert.Equal(t, map[string]any{"a": float64(1), "b": float64(2)}, result)
}

// TestForExecNumericAccumulatesResults verifies the numeric for.in variant still
// returns a slice with one entry per iteration index.
func TestForExecNumericAccumulatesResults(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "num-accum")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "num-accum",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "val",
					At:   "idx",
					In:   "${ $data.count }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{Output: st.Data["idx"], Context: nil}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.AddData(map[string]any{"count": 3})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "num-accum-outer"})

	env.ExecuteWorkflow("num-accum-outer")
	assert.NoError(t, env.GetWorkflowError())

	var result []any
	assert.NoError(t, env.GetWorkflowResult(&result))

	// JSON round-trip through the Temporal child workflow boundary converts integers to float64.
	assert.Equal(t, []any{float64(0), float64(1), float64(2)}, result)
}

// TestForExecLoopVarsDoNotLeakToParent verifies that loop-local variables
// (item, index or custom names) are not present in the parent state's Data
// after the for loop completes.
func TestForExecLoopVarsDoNotLeakToParent(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "leak-check")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "leak-check",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "item",
					At:   "idx",
					In:   "${ $data.items }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{Output: st.Data["item"], Context: nil}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.AddData(map[string]any{"items": []any{"a", "b"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "leak-check-outer"})

	env.ExecuteWorkflow("leak-check-outer")
	assert.NoError(t, env.GetWorkflowError())

	// Loop-local variables must not appear in parent state after the loop exits.
	assert.Nil(t, state.Data["item"], "item must not leak to parent state")
	assert.Nil(t, state.Data["idx"], "idx must not leak to parent state")
	// The pre-loop data must be unmodified.
	assert.Equal(t, []any{"a", "b"}, state.Data["items"])
}

// TestForExecContextDoesNotLeakToParent verifies that $context updated by an
// export inside the loop does NOT propagate to the parent state after exec().
// workingState.Context is loop-private and must not overwrite the surrounding
// workflow context. Loop-internal Data changes must also not appear in parent Data.
func TestForExecContextDoesNotLeakToParent(t *testing.T) {
	childWorkflowName := utils.GenerateChildWorkflowName("for", "ctx-leak")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "ctx-leak",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "item",
					At:   "idx",
					In:   "${ $data.items }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{
				Output:  st.Data["item"],
				Context: map[string]any{"last": st.Data["item"]},
			}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.AddData(map[string]any{"items": []any{"x", "y"}})

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
	childWorkflowName := utils.GenerateChildWorkflowName("for", "no-out-leak")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "no-out-leak",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "item",
					At:   "idx",
					In:   "${ $data.items }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{Output: "iteration-output", Context: nil}, nil
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.AddData(map[string]any{"items": []any{"a"}})

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
	childWorkflowName := utils.GenerateChildWorkflowName("for", "err-unchanged")

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "err-unchanged",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "item",
					At:   "idx",
					In:   "${ $data.items }",
				},
				Do: &model.TaskList{&model.TaskItem{Key: "step", Task: &model.DoTask{}}},
			},
		},
		childWorkflowName: childWorkflowName,
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, st *utils.State) (forChildResult, error) {
			return forChildResult{}, fmt.Errorf("simulated iteration failure")
		},
		workflow.RegisterOptions{Name: childWorkflowName},
	)

	state := utils.NewState()
	state.Context = map[string]any{"original": true}
	state.Output = "original-output"
	state.AddData(map[string]any{"items": []any{"a"}})

	execFn, err := b.exec()
	assert.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return execFn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "err-unchanged-outer"})

	env.ExecuteWorkflow("err-unchanged-outer")
	// The workflow must have failed due to the child error.
	assert.Error(t, env.GetWorkflowError())

	// Context, Output, and Data must all be unchanged.
	assert.Equal(t, map[string]any{"original": true}, state.Context,
		"exec() must not modify state.Context when an iteration fails")
	assert.Equal(t, "original-output", state.Output,
		"exec() must not modify state.Output when an iteration fails")
	assert.Equal(t, []any{"a"}, state.Data["items"],
		"pre-loop Data must be unchanged when an iteration fails")
	assert.Nil(t, state.Data["err-unchanged"],
		"loop task result must not appear in parent Data when an iteration fails")
	assert.Nil(t, state.Data["item"],
		"loop-local variable must not leak into parent Data when an iteration fails")
	assert.Nil(t, state.Data["idx"],
		"loop-local variable must not leak into parent Data when an iteration fails")
}

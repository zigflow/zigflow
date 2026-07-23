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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// newForTestBuilder builds a ForTaskBuilder for the exec/iterator tests. The
// Do list is a placeholder: these tests supply the iteration body directly as a
// TemporalWorkflowFunc closure to exec/iterator, so the body is never built
// from the task list.
func newForTestBuilder(name string, cfg model.ForTaskConfiguration, while string) *ForTaskBuilder {
	return &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         name,
			task: &model.ForTask{
				For:   cfg,
				While: while,
				Do:    &model.TaskList{&model.TaskItem{Key: testConstStep, Task: &model.DoTask{}}},
			},
		},
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

			builder := newForTestBuilder("for-task", model.ForTaskConfiguration{In: testConstForDataItems}, tc.while)

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

// TestForTaskBuilderIterator proves one iteration runs the inline body with the
// current loop-local variables, returns its output, propagates that output onto
// the working state, and keeps the loop-local variables off the working state.
func TestForTaskBuilderIterator(t *testing.T) {
	state := utils.NewState()
	state.Input = map[string]any{testConstRequestID: "abc"}

	builder := newForTestBuilder("iterate", model.ForTaskConfiguration{
		Each: testConstValue,
		At:   testConstIdx,
		In:   testConstForDataItems,
	}, "")

	// The body reads the loop-local item variable and echoes it back under a
	// child key, mirroring what a real do-body would produce.
	var (
		bodyItem  any
		bodyIndex any
	)
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		bodyItem = st.Data[testConstValue]
		bodyIndex = st.Data[testConstIdx]
		return map[string]any{testConstChildValue: st.Data[testConstValue]}, nil
	}

	state.AddData(map[string]any{testConstItems: []any{testConstItemValue}})

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return builder.iterator(ctx, 0, testConstItemValue, state, bodyFn)
	}, workflow.RegisterOptions{Name: "iterator-test"})

	env.ExecuteWorkflow("iterator-test")
	require.NoError(t, env.GetWorkflowError())

	// The body observed the current loop-local variables.
	assert.Equal(t, testConstItemValue, bodyItem)
	assert.Equal(t, 0, bodyIndex)

	// iterator propagates the body's output onto the working state (here the
	// state passed as workingState) with no JSON coercion of the value.
	assert.Equal(t, map[string]any{testConstChildValue: testConstItemValue}, state.Output)

	// Loop-local variables live on the per-iteration clone and must not appear
	// on the working state passed to iterator().
	assert.Nil(t, state.Data[testConstValue])
	assert.Nil(t, state.Data[testConstIdx])
}

// TestForIteratorContextPropagates verifies that $context exported by iteration
// N is visible as state.Context when iterator() runs for N+1.
func TestForIteratorContextPropagates(t *testing.T) {
	b := newForTestBuilder("ctx-prop", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForDataItems,
	}, "")

	// The body records the context it received then exports a new one.
	var receivedContexts []any
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		receivedContexts = append(receivedContexts, st.Context)
		st.Context = map[string]any{testConstLast: st.Data[constDefaultItemVar]}
		return st.Data[constDefaultItemVar], nil
	}

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		if _, err := b.iterator(ctx, 0, "alpha", state, bodyFn); err != nil {
			return err
		}
		if _, err := b.iterator(ctx, 1, "beta", state, bodyFn); err != nil {
			return err
		}
		return nil
	}, workflow.RegisterOptions{Name: "ctx-prop-outer"})

	env.ExecuteWorkflow("ctx-prop-outer")
	require.NoError(t, env.GetWorkflowError())

	require.Len(t, receivedContexts, 2)
	// First iteration starts with no exported context.
	assert.Nil(t, receivedContexts[0])
	// Second iteration sees the context exported by the first iteration.
	assert.Equal(t, map[string]any{testConstLast: "alpha"}, receivedContexts[1])
	// Working state ends with the context from the final iteration.
	assert.Equal(t, map[string]any{testConstLast: "beta"}, state.Context)
}

// TestForIteratorWhileSeesOutput verifies that the while condition for iteration
// N+1 sees the output produced by iteration N, and that the body is not invoked
// for a rejected iteration.
func TestForIteratorWhileSeesOutput(t *testing.T) {
	b := newForTestBuilder("while-out", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForDataItems,
	}, "${ $output.continue }")

	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		// The first iteration returns continue=false so the second iteration's
		// while check stops the loop.
		return map[string]any{testConstFlowContinue: false}, nil
	}

	state := utils.NewState()
	// Pre-seed output so the first while check (before any iteration) passes.
	state.Output = map[string]any{testConstFlowContinue: true}

	// stoppedAt: 0 never stopped, 1 stopped on first, 2 stopped on second.
	stoppedAt := 0
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		if _, err := b.iterator(ctx, 0, "first", state, bodyFn); err != nil {
			if errors.Is(err, errForkIterationStop) {
				stoppedAt = 1
				return nil
			}
			return err
		}
		if _, err := b.iterator(ctx, 1, "second", state, bodyFn); err != nil {
			if errors.Is(err, errForkIterationStop) {
				stoppedAt = 2
				return nil
			}
			return err
		}
		return nil
	}, workflow.RegisterOptions{Name: "while-out-outer"})

	env.ExecuteWorkflow("while-out-outer")
	require.NoError(t, env.GetWorkflowError())
	// The while condition stopped the loop at the second iteration call.
	assert.Equal(t, 2, stoppedAt)
	// The body ran only for the first iteration; the second was rejected by while.
	assert.Equal(t, 1, callCount)
}

// TestForExecArrayAccumulatesResults verifies exec returns an ordered array of
// per-iteration results for an array for.in value, with the body invoked once
// per item and no JSON coercion of the values.
func TestForExecArrayAccumulatesResults(t *testing.T) {
	b := newForTestBuilder("accum", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return map[string]any{testConstProcessed: st.Data[constDefaultItemVar]}, nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"x", "y", "z"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	assert.Equal(t, []any{
		map[string]any{testConstProcessed: "x"},
		map[string]any{testConstProcessed: "y"},
		map[string]any{testConstProcessed: "z"},
	}, output)
	assert.Equal(t, 3, callCount)
	// exec sets the parent output to the aggregated result on clean completion.
	assert.Equal(t, output, state.Output)
}

// TestForExecObjectAccumulatesResults verifies the object for.in variant returns
// a map keyed by the original object keys. Inline execution preserves concrete
// Go types (no float64 coercion). Map iteration order is not asserted.
func TestForExecObjectAccumulatesResults(t *testing.T) {
	b := newForTestBuilder("obj-accum", model.ForTaskConfiguration{
		Each: testConstVal,
		At:   "key",
		In:   testConstForRefDataItems,
	}, "")

	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		return st.Data[testConstVal], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: map[string]any{"a": 1, "b": 2}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	// Inline execution keeps ints as ints; there is no child-boundary JSON trip.
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, output)
}

// TestForExecNumericAccumulatesResults verifies the numeric for.in variant
// returns a slice with one entry per iteration index, preserving int types.
func TestForExecNumericAccumulatesResults(t *testing.T) {
	b := newForTestBuilder("num-accum", model.ForTaskConfiguration{
		Each: testConstVal,
		At:   testConstIdx,
		In:   testConstForRefDataCount,
	}, "")

	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		return st.Data[testConstIdx], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: 3})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	assert.Equal(t, []any{0, 1, 2}, output)
}

// TestForExecNumericFloat64Whole verifies a float64 whole number (e.g. 5.0),
// as jq may yield after a JSON round trip, is accepted as an iteration count.
func TestForExecNumericFloat64Whole(t *testing.T) {
	b := newForTestBuilder("num-float64-whole", model.ForTaskConfiguration{
		Each: testConstVal,
		At:   testConstIdx,
		In:   testConstForRefDataCount,
	}, "")

	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return st.Data[testConstIdx], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: float64(5)})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	assert.Equal(t, []any{0, 1, 2, 3, 4}, output)
	assert.Equal(t, 5, callCount, "body must be invoked once per iteration")
}

// TestForExecNumericFloat64Zero verifies a float64 zero yields no iterations
// rather than an error.
func TestForExecNumericFloat64Zero(t *testing.T) {
	b := newForTestBuilder("num-float64-zero", model.ForTaskConfiguration{
		Each: testConstVal,
		At:   testConstIdx,
		In:   testConstForRefDataCount,
	}, "")

	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return st.Data[testConstIdx], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: float64(0)})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	assert.Equal(t, []any{}, output, "zero count must produce no iterations")
	assert.Equal(t, 0, callCount, "body must not be invoked when count is zero")
}

// TestForExecNumericFloat64Fractional verifies a non-whole-number float64 for.in
// value fails with a clear error naming the offending value, and no iterations
// run.
func TestForExecNumericFloat64Fractional(t *testing.T) {
	b := newForTestBuilder("num-float64-frac", model.ForTaskConfiguration{
		Each: testConstVal,
		At:   testConstIdx,
		In:   testConstForRefDataCount,
	}, "")

	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return st.Data[testConstIdx], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstCount: 5.5})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	_, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "for task numeric iteration value must be a whole number")
	assert.Contains(t, execErr.Error(), "5.5")
	assert.Equal(t, 0, callCount, "no iterations must run when the for.in value is invalid")
}

// TestForExecLoopVarsDoNotLeakToParent verifies loop-local variables are not
// present in the parent state's Data after the loop completes.
func TestForExecLoopVarsDoNotLeakToParent(t *testing.T) {
	b := newForTestBuilder("leak-check", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	// The body also mutates its own Data; those mutations must not leak either.
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		st.AddData(map[string]any{"body-only": true})
		return st.Data[constDefaultItemVar], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"a", "b"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	_, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	assert.Nil(t, state.Data[constDefaultItemVar], "item must not leak to parent state")
	assert.Nil(t, state.Data[testConstIdx], "idx must not leak to parent state")
	assert.Nil(t, state.Data["body-only"], "arbitrary body Data must not leak to parent state")
	// The pre-loop data must be unmodified.
	assert.Equal(t, []any{"a", "b"}, state.Data[testConstItems])
}

// TestForIterInterIterationDataIsolation verifies arbitrary Data written by one
// iteration body does not carry into the next iteration; only Context and Output
// cross the boundary.
func TestForIterInterIterationDataIsolation(t *testing.T) {
	b := newForTestBuilder("data-iso", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	var seenBodyOnly []any
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		seenBodyOnly = append(seenBodyOnly, st.Data["body-only"])
		st.AddData(map[string]any{"body-only": st.Data[constDefaultItemVar]})
		return st.Data[constDefaultItemVar], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"a", "b"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	_, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	require.Len(t, seenBodyOnly, 2)
	// Neither iteration sees the "body-only" data written by the other.
	assert.Nil(t, seenBodyOnly[0])
	assert.Nil(t, seenBodyOnly[1], "arbitrary body Data must not cross to the next iteration")
}

// TestForExecContextDoesNotLeakToParent verifies $context updated by an export
// inside the loop does not propagate to the parent state after exec(), and that
// loop-internal Data (e.g. the accumulated iteration result) does not appear in
// the parent Data.
func TestForExecContextDoesNotLeakToParent(t *testing.T) {
	b := newForTestBuilder("ctx-leak", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		st.Context = map[string]any{testConstLast: st.Data[constDefaultItemVar]}
		return st.Data[constDefaultItemVar], nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"x", "y"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	_, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	// workingState.Context is loop-private and must NOT be copied to state.Context.
	assert.Nil(t, state.Context, "loop-private context must not leak to parent state")
	// Loop-internal Data (taskName -> lastResult set by addIterationResult on
	// workingState) must not appear in the parent state's Data.
	assert.Nil(t, state.Data["ctx-leak"], "accumulated iteration result must not leak to parent Data")
}

// TestForExecOutputIsAggregatedResult verifies exec() sets state.Output to the
// aggregated per-iteration result, not the last iteration's internal $output.
func TestForExecOutputIsAggregatedResult(t *testing.T) {
	b := newForTestBuilder("no-out-leak", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	const iterationOutput = "iteration-output"
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		return iterationOutput, nil
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"a"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.NoError(t, execErr)

	assert.Equal(t, []any{iterationOutput}, output)
	assert.Equal(t, []any{iterationOutput}, state.Output,
		"exec() must set state.Output to the aggregated loop result")
}

// TestForExecErrorLeavesParentStateUnchanged verifies that when an iteration
// body fails, the loop aborts, a contextual error is returned, and the parent
// state is not modified.
func TestForExecErrorLeavesParentStateUnchanged(t *testing.T) {
	b := newForTestBuilder("err-unchanged", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return nil, fmt.Errorf("simulated iteration failure")
	}

	state := utils.NewState()
	state.Context = map[string]any{"original": true}
	state.Output = "original-output"
	state.AddData(map[string]any{testConstItems: []any{"a", "b"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)
	require.Error(t, execErr)
	assert.Nil(t, output)
	assert.Contains(t, execErr.Error(), "error running for iteration tasks")
	// The loop aborts on the first failure rather than running every item.
	assert.Equal(t, 1, callCount)

	// Context, Output and Data must all be unchanged.
	assert.Equal(t, map[string]any{"original": true}, state.Context,
		"exec() must not modify state.Context when an iteration fails")
	assert.Equal(t, "original-output", state.Output,
		"exec() must not modify state.Output when an iteration fails")
	assert.Equal(t, []any{"a", "b"}, state.Data[testConstItems],
		"pre-loop Data must be unchanged when an iteration fails")
	assert.Nil(t, state.Data["err-unchanged"],
		"loop task result must not appear in parent Data when an iteration fails")
	assert.Nil(t, state.Data[constDefaultItemVar],
		"loop-local variable must not leak into parent Data when an iteration fails")
}

// TestForExecPropagatesDirectErrEnd proves that an iteration body returning
// (output, flow.ErrEnd) directly — the normal inline shape — terminates the
// loop, preserves the carried output, surfaces flow.ErrEnd to the caller, and
// does not run later iterations. This is the primary inline end path.
func TestForExecPropagatesDirectErrEnd(t *testing.T) {
	b := newForTestBuilder("iter-end", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	endOutput := map[string]any{testConstValue: "end-time-output"}
	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return endOutput, flow.ErrEnd
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"x", "y"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd), "direct end must surface as flow.ErrEnd")
	assert.Equal(t, endOutput, output, "carried end output must be preserved")
	assert.Equal(t, 1, callCount, "later iterations must not run after end")
	// Parent output reflects the effective end output.
	assert.Equal(t, endOutput, state.Output)
}

// TestForExecPropagatesEncodedErrEnd is the retained backwards-compatibility
// path: an encoded Temporal end error (as produced by
// flow.NewEndApplicationError) is still recognised, its carried payload output
// preserved, and later iterations skipped. Direct flow.ErrEnd remains the
// primary path (see TestForExecPropagatesDirectErrEnd).
func TestForExecPropagatesEncodedErrEnd(t *testing.T) {
	b := newForTestBuilder("iter-end-encoded", model.ForTaskConfiguration{
		Each: constDefaultItemVar,
		At:   testConstIdx,
		In:   testConstForRefDataItems,
	}, "")

	encodedOutput := map[string]any{testConstValue: "encoded-end-output"}
	callCount := 0
	var bodyFn TemporalWorkflowFunc = func(_ workflow.Context, _ any, st *utils.State) (any, error) {
		callCount++
		return nil, flow.NewEndApplicationError(encodedOutput)
	}

	state := utils.NewState()
	state.AddData(map[string]any{testConstItems: []any{"x", "y"}})

	execFn, err := b.exec(bodyFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "for-exec", execFn, nil, state)

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd), "encoded end must surface as flow.ErrEnd")
	assert.Equal(t, encodedOutput, output, "encoded end payload output must be preserved")
	assert.Equal(t, 1, callCount, "later iterations must not run after end")
	assert.Equal(t, encodedOutput, state.Output)
}

// TestForBuildDoesNotRegisterChildWorkflow proves the for body is built inline:
// Build produces a non-nil executable function and never registers a child
// workflow, and PostLoad and Validate use the same inline (non-registering)
// configuration.
func TestForBuildDoesNotRegisterChildWorkflow(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-for-noreg"}}

	w := new(WorkflowRegistryMock)

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:            doc,
			eventEmitter:   testEvents,
			name:           "loop",
			taskPath:       []string{"loop"},
			temporalWorker: w,
			task: &model.ForTask{
				For: model.ForTaskConfiguration{In: "[]"},
				Do: &model.TaskList{
					&model.TaskItem{Key: testConstStep, Task: &model.SetTask{}},
				},
			},
		},
	}

	fn, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, fn, "Build must return a non-nil executable function")

	require.NoError(t, b.PostLoad())
	require.NoError(t, b.Validate())

	// No child workflow may be registered for the inline body.
	w.AssertNotCalled(t, "RegisterWorkflowWithOptions", mock.Anything, mock.Anything)
}

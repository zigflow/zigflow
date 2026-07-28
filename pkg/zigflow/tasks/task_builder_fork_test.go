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
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

const (
	// winnerKey is the output key used by the competing-fork tests.
	winnerKey = "winner"
	// branchAKey is a fork branch key reused across the fork builder tests.
	branchAKey = "branchA"
	// fastWinner is the winning branch's marker in the competing-fork test.
	fastWinner = "fast"
)

// newForkExecBuilder builds a ForkTaskBuilder for the exec-level tests. Branches
// are supplied directly to exec as inline closures, so the task's Branches list
// is not consulted.
func newForkExecBuilder(compete bool) *ForkTaskBuilder {
	return &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "fork-task",
			task: &model.ForkTask{
				Fork: model.ForkTaskConfiguration{Compete: compete},
			},
		},
	}
}

// runForkExec builds the fork exec function from the supplied inline branches
// and runs it inside the workflow test environment, returning the raw output
// and error captured before the test-environment boundary (so native Go types
// and the error chain survive for assertions).
func runForkExec(t *testing.T, compete bool, state *utils.State, branches ...forkBranch) (any, error) {
	t.Helper()

	b := newForkExecBuilder(compete)
	fn, err := b.exec(branches)
	require.NoError(t, err)

	if state == nil {
		state = utils.NewState()
	}

	return runInlineWorkflowFunc(t, "fork-exec-host", fn, nil, state)
}

// sleepBranch returns a branch that waits (in workflow time) before producing
// its output, letting tests force completion order to differ from declaration
// order without wall-clock sleeps.
func sleepBranch(name string, d time.Duration, output any) forkBranch {
	return forkBranch{
		name: name,
		fn: func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			if err := workflow.Sleep(ctx, d); err != nil {
				return nil, err
			}
			return output, nil
		},
	}
}

// TestForkExecAggregatesDeclarationOrderDespiteCompletion proves branches run
// concurrently and the aggregate output is by branch name regardless of the
// order branches complete in. The fast branch finishing first must not drop or
// reorder the slow branch's contribution.
func TestForkExecAggregatesDeclarationOrderDespiteCompletion(t *testing.T) {
	output, err := runForkExec(
		t, false, nil,
		sleepBranch("a", 3*time.Hour, "a-out"),
		sleepBranch("b", 1*time.Hour, "b-out"),
		sleepBranch("c", 0, "c-out"),
	)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"a": "a-out",
		"b": "b-out",
		"c": "c-out",
	}, output)
}

// TestForkExecBranchesReceiveClonedParentState proves each branch starts from
// the same parent input state (a clone), and that the fork does not mutate the
// parent state.
func TestForkExecBranchesReceiveClonedParentState(t *testing.T) {
	parent := utils.NewState()
	parent.Input = map[string]any{"in": "put"}
	parent.AddData(map[string]any{testConstSeed: 1})

	var aSeed, bSeed any
	var aInput, bInput any

	output, err := runForkExec(
		t, false, parent,
		forkBranch{name: "a", fn: func(_ workflow.Context, input any, st *utils.State) (any, error) {
			aSeed = st.Data[testConstSeed]
			aInput = input
			return "a", nil
		}},
		forkBranch{name: "b", fn: func(_ workflow.Context, input any, st *utils.State) (any, error) {
			bSeed = st.Data[testConstSeed]
			bInput = input
			return "b", nil
		}},
	)
	require.NoError(t, err)

	// Both branches saw the same seeded parent data (via their clones).
	assert.Equal(t, 1, aSeed)
	assert.Equal(t, 1, bSeed)
	// Input flows through unchanged.
	assert.Nil(t, aInput)
	assert.Nil(t, bInput)
	assert.Equal(t, map[string]any{"a": "a", "b": "b"}, output)

	// Parent state is untouched by the fork.
	assert.Equal(t, map[string]any{testConstSeed: 1}, parent.Data)
	assert.Nil(t, parent.Output)
}

// TestForkExecBranchDataAndContextIsolation proves no branch observes another
// branch's Data or Context mutations, and neither leaks to the parent.
func TestForkExecBranchDataAndContextIsolation(t *testing.T) {
	parent := utils.NewState()

	var aSawX, bSawX any
	var aSawCtx, bSawCtx any

	_, err := runForkExec(
		t, false, parent,
		// "a" waits so "b" mutates its own state first; isolation must hold
		// regardless of ordering.
		forkBranch{name: "a", fn: func(ctx workflow.Context, _ any, st *utils.State) (any, error) {
			if err := workflow.Sleep(ctx, time.Hour); err != nil {
				return nil, err
			}
			aSawX = st.Data["x"]
			aSawCtx = st.Context
			st.AddData(map[string]any{"x": "a"})
			st.Context = map[string]any{"who": "a"}
			return "a", nil
		}},
		forkBranch{name: "b", fn: func(_ workflow.Context, _ any, st *utils.State) (any, error) {
			bSawX = st.Data["x"]
			bSawCtx = st.Context
			st.AddData(map[string]any{"x": "b"})
			st.Context = map[string]any{"who": "b"}
			return "b", nil
		}},
	)
	require.NoError(t, err)

	// Neither branch saw the other's Data or Context mutation.
	assert.Nil(t, aSawX, "branch a must not see branch b's Data mutation")
	assert.Nil(t, bSawX, "branch b must not see branch a's Data mutation")
	assert.Nil(t, aSawCtx, "branch a must not see branch b's Context mutation")
	assert.Nil(t, bSawCtx, "branch b must not see branch a's Context mutation")

	// Neither branch's mutation leaked to the parent.
	assert.Nil(t, parent.Data["x"], "branch Data must not leak to parent")
	assert.Nil(t, parent.Context, "branch Context must not leak to parent")
}

// TestForkExecSingleBranchFailure proves a genuine branch failure fails the
// fork, is wrapped with inline terminology and the branch name, and leaves the
// parent state unchanged.
func TestForkExecSingleBranchFailure(t *testing.T) {
	parent := utils.NewState()
	parent.Output = testConstOriginal

	output, err := runForkExec(
		t, false, parent,
		forkBranch{name: "boom", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return nil, errors.New("genuine branch failure")
		}},
	)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "error running fork branch tasks")
	assert.Contains(t, err.Error(), "boom", "wrapped error must identify the failing branch")
	assert.Contains(t, err.Error(), "genuine branch failure")
	assert.NotContains(t, err.Error(), flow.ErrEnd.Error())

	// Parent output is untouched on failure.
	assert.Equal(t, testConstOriginal, parent.Output)
}

// TestForkExecMultipleFailuresLowestIndexWins proves failure selection is
// deterministic by declaration index, not completion order: the branch that
// fails first in wall-clock terms must not win if a lower-index branch also
// fails.
func TestForkExecMultipleFailuresLowestIndexWins(t *testing.T) {
	_, err := runForkExec(
		t, false, nil,
		// index 0 fails, but only after a delay.
		forkBranch{name: "a", fn: func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			if err := workflow.Sleep(ctx, time.Hour); err != nil {
				return nil, err
			}
			return nil, errors.New("failure-a")
		}},
		// index 1 fails immediately.
		forkBranch{name: "b", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return nil, errors.New("failure-b")
		}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failure-a", "lowest-index failure must win deterministically")
	assert.Contains(t, err.Error(), "(a)")
	assert.NotContains(t, err.Error(), "failure-b")
}

// TestForkExecNonCompetingRunsAllBranchesDespiteFailure proves non-competing
// forks wait for every branch: a sibling still runs to completion even though
// another branch fails.
func TestForkExecNonCompetingRunsAllBranchesDespiteFailure(t *testing.T) {
	siblingRan := false

	_, err := runForkExec(
		t, false, nil,
		forkBranch{name: "boom", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return nil, errors.New("boom")
		}},
		forkBranch{name: "sibling", fn: func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			if err := workflow.Sleep(ctx, time.Hour); err != nil {
				return nil, err
			}
			siblingRan = true
			return testConstOK, nil
		}},
	)

	require.Error(t, err)
	assert.True(t, siblingRan, "non-competing fork must wait for all branches even when one fails")
}

// TestForkExecCompeteFirstCompletedWinsAndCancelsLosers proves competing forks
// return the first completed branch's output and cancel the losers.
func TestForkExecCompeteFirstCompletedWinsAndCancelsLosers(t *testing.T) {
	slowCompleted := false

	output, err := runForkExec(
		t, true, nil,
		forkBranch{name: fastWinner, fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return map[string]any{winnerKey: fastWinner}, nil
		}},
		forkBranch{name: "slow", fn: func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			if err := workflow.Sleep(ctx, time.Hour); err != nil {
				return nil, err
			}
			slowCompleted = true
			return map[string]any{winnerKey: "slow"}, nil
		}},
	)
	require.NoError(t, err)

	// The competing fork returns the winner's output directly.
	assert.Equal(t, map[string]any{winnerKey: fastWinner}, output)
	// The loser was cancelled before it could finish its delayed work.
	assert.False(t, slowCompleted, "losing branch must be cancelled once the winner is decided")
}

// TestForkExecNonCompetingWaitsForAll proves that without compete every branch
// completes and contributes to the aggregate.
func TestForkExecNonCompetingWaitsForAll(t *testing.T) {
	aRan, bRan := false, false

	output, err := runForkExec(
		t, false, nil,
		forkBranch{name: "a", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			aRan = true
			return "a", nil
		}},
		forkBranch{name: "b", fn: func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			if err := workflow.Sleep(ctx, time.Hour); err != nil {
				return nil, err
			}
			bRan = true
			return "b", nil
		}},
	)
	require.NoError(t, err)

	assert.True(t, aRan)
	assert.True(t, bRan)
	assert.Equal(t, map[string]any{"a": "a", "b": "b"}, output)
}

// TestForkExecDirectErrEnd proves a branch returning (output, flow.ErrEnd)
// directly — the primary inline path — terminates the fork with flow.ErrEnd and
// the branch's carried output, not wrapped as a fork failure.
func TestForkExecDirectErrEnd(t *testing.T) {
	endOutput := map[string]any{testConstValue: "branch-end-output"}

	output, err := runForkExec(
		t, false, nil,
		forkBranch{name: "ending", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return endOutput, flow.ErrEnd
		}},
	)

	require.Error(t, err)
	assert.True(t, errors.Is(err, flow.ErrEnd), "direct end must surface as flow.ErrEnd")
	assert.Equal(t, endOutput, output, "branch end output must be preserved")
	assert.NotContains(t, err.Error(), "error running fork branch tasks")
}

// TestForkExecEncodedErrEnd proves the retained backwards-compatibility path: an
// encoded Temporal end error is decoded and its payload output preserved.
func TestForkExecEncodedErrEnd(t *testing.T) {
	encodedOutput := map[string]any{testConstValue: testConstEncodedEndOutput}

	output, err := runForkExec(
		t, false, nil,
		forkBranch{name: "ending", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return nil, flow.NewEndApplicationError(encodedOutput)
		}},
	)

	require.Error(t, err)
	assert.True(t, errors.Is(err, flow.ErrEnd), "encoded end must surface as flow.ErrEnd")
	assert.Equal(t, encodedOutput, output, "encoded end payload output must be preserved")
}

// TestForkExecErrorTakesPrecedenceOverEnd proves a genuine failure outranks an
// end directive regardless of index: an end at index 0 does not mask an error
// at index 1.
func TestForkExecErrorTakesPrecedenceOverEnd(t *testing.T) {
	_, err := runForkExec(
		t, false, nil,
		forkBranch{name: "ends", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return map[string]any{"k": "v"}, flow.ErrEnd
		}},
		forkBranch{name: "fails", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return nil, errors.New("genuine failure")
		}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error running fork branch tasks")
	assert.False(t, errors.Is(err, flow.ErrEnd), "a genuine failure must outrank an end directive")
}

// TestForkExecEndLowestIndexWins proves that among multiple ending branches the
// lowest-index branch's output is the one carried, deterministically.
func TestForkExecEndLowestIndexWins(t *testing.T) {
	output, err := runForkExec(
		t, false, nil,
		// index 0 ends after a delay.
		forkBranch{name: "a", fn: func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
			if serr := workflow.Sleep(ctx, time.Hour); serr != nil {
				return nil, serr
			}
			return "end-a", flow.ErrEnd
		}},
		// index 1 ends immediately.
		forkBranch{name: "b", fn: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			return "end-b", flow.ErrEnd
		}},
	)

	require.Error(t, err)
	assert.True(t, errors.Is(err, flow.ErrEnd))
	assert.Equal(t, "end-a", output, "lowest-index end output must win deterministically")
}

// TestForkExecEmptyReturnsEmptyMap proves an empty branch set is a safe no-op
// rather than a workflow that blocks forever.
func TestForkExecEmptyReturnsEmptyMap(t *testing.T) {
	output, err := runForkExec(t, false, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, output)
}

// TestForkBranchBuilderUsesInlineOptions proves fork branch bodies are built
// with both inline options enabled: not registered as standalone workflows and
// exposing inline control-flow directives.
func TestForkBranchBuilderUsesInlineOptions(t *testing.T) {
	b := &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         testForkTaskName,
			taskPath:     []string{testForkTaskName},
			task:         &model.ForkTask{},
		},
	}

	branch := &model.TaskItem{Key: branchAKey, Task: &model.SetTask{}}
	builder, err := b.branchBuilder(branch)
	require.NoError(t, err)

	assert.True(t, builder.opts.DisableRegisterWorkflow, "fork branch bodies must not register a workflow")
	assert.True(t, builder.opts.InlineExecution, "fork branch bodies must run with inline control-flow semantics")
}

// TestForkBuildDoesNotRegisterBranchWorkflows proves Build/PostLoad/Validate
// build branch bodies without registering any branch child workflow.
func TestForkBuildDoesNotRegisterBranchWorkflows(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork-noreg"}}
	w := new(WorkflowRegistryMock)

	forkTask := &model.ForkTask{
		Fork: model.ForkTaskConfiguration{
			Branches: &model.TaskList{
				&model.TaskItem{Key: branchAKey, Task: &model.SetTask{}},
				&model.TaskItem{Key: "branchB", Task: &model.SetTask{}},
			},
		},
	}

	b := &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			eventEmitter:   testEvents,
			name:           testForkTaskName,
			taskPath:       []string{testForkTaskName},
			task:           forkTask,
			temporalWorker: w,
		},
	}

	fn, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, fn)

	require.NoError(t, b.PostLoad())
	require.NoError(t, b.Validate())

	w.AssertNotCalled(t, "RegisterWorkflowWithOptions", mock.Anything, mock.Anything)
}

// TestForkDuplicateLeafNamesRegisterDistinctActivities proves branches that
// reuse the same leaf task name register distinct per-task activity aliases,
// because each branch threads a unique task path.
func TestForkDuplicateLeafNamesRegisterDistinctActivities(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork-dup"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork-dup.dispatch.left.step",
		}).
		Once()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork-dup.dispatch.right.step",
		}).
		Once()

	forkTask := &model.ForkTask{
		Fork: model.ForkTaskConfiguration{
			Branches: &model.TaskList{
				&model.TaskItem{Key: "left", Task: &model.DoTask{Do: &model.TaskList{
					&model.TaskItem{Key: testConstStep, Task: newTestHTTPTask()},
				}}},
				&model.TaskItem{Key: "right", Task: &model.DoTask{Do: &model.TaskList{
					&model.TaskItem{Key: testConstStep, Task: newTestHTTPTask()},
				}}},
			},
		},
	}

	b := &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			eventEmitter:   testEvents,
			name:           testForkTaskName,
			taskPath:       []string{testForkTaskName},
			task:           forkTask,
			temporalWorker: w,
		},
	}

	_, err := b.Build()
	require.NoError(t, err)

	w.AssertExpectations(t)
}

// testForkTaskName is the fork task's own name (and sole path segment) used by
// the alias-derivation tests below.
const testForkTaskName = "dispatch"

// A single-task fork branch is wrapped as an inline do-task before being built.
// The generated per-task activity alias must be derived from the original,
// user-visible branch key ("branchA"), threaded via the fork's task path.
//
// No branch workflow is registered (inline execution); only the activity alias
// is asserted.
func TestForkSingleTaskBranchAliasUsesOriginalBranchKey(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork-single"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork-single.dispatch.branchA",
		}).
		Once()

	forkTask := &model.ForkTask{
		Fork: model.ForkTaskConfiguration{
			Branches: &model.TaskList{
				&model.TaskItem{Key: branchAKey, Task: newTestHTTPTask()},
			},
		},
	}

	b := &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			eventEmitter:   testEvents,
			name:           testForkTaskName,
			taskPath:       []string{testForkTaskName},
			task:           forkTask,
			temporalWorker: w,
		},
	}

	_, err := b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

// A multi-task fork branch is a do-task scope: the branch key is an intermediate
// path segment and the body's leaf tasks nest beneath it.
func TestForkMultiTaskBranchAliasNestsUnderBranchKey(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork-multi"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork-multi.dispatch.branchB.leaf",
		}).
		Once()

	forkTask := &model.ForkTask{
		Fork: model.ForkTaskConfiguration{
			Branches: &model.TaskList{
				&model.TaskItem{
					Key: "branchB",
					Task: &model.DoTask{
						Do: &model.TaskList{
							&model.TaskItem{Key: "leaf", Task: newTestHTTPTask()},
						},
					},
				},
			},
		},
	}

	b := &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			eventEmitter:   testEvents,
			name:           testForkTaskName,
			taskPath:       []string{testForkTaskName},
			task:           forkTask,
			temporalWorker: w,
		},
	}

	_, err := b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

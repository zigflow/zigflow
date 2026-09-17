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
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type fakeWorkflowContext struct{}

func (fakeWorkflowContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (fakeWorkflowContext) Done() workflow.Channel      { return nil }
func (fakeWorkflowContext) Err() error                  { return nil }
func (fakeWorkflowContext) Value(key any) any {
	return nil
}

func TestForkTaskBuilderAwaitCondition(t *testing.T) {
	builder := &ForkTaskBuilder{}

	tests := []struct {
		name        string
		replyErr    error
		endSeen     bool
		isCompeting bool
		winningCtx  workflow.Context
		hasReplied  []bool
		expect      bool
	}{
		{
			name:     "reply error short circuits",
			replyErr: errors.New("boom"),
			expect:   true,
		},
		{
			name:    "end signal short circuits",
			endSeen: true,
			expect:  true,
		},
		{
			name:        "competing fork waits for winner",
			isCompeting: true,
			expect:      false,
		},
		{
			name:        "competing fork with winner returns true",
			isCompeting: true,
			winningCtx:  fakeWorkflowContext{},
			expect:      true,
		},
		{
			name:       "non competing waits for all replies",
			hasReplied: []bool{true, false},
			expect:     false,
		},
		{
			name:       "non competing completes when all replied",
			hasReplied: []bool{true, true},
			expect:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cond := builder.awaitCondition(tc.replyErr, tc.endSeen, tc.isCompeting, tc.winningCtx, tc.hasReplied)
			assert.Equal(t, tc.expect, cond())
		})
	}
}

// forkBranch is the shape of a child workflow backing a fork branch in
// these tests.
type forkBranch func(ctx workflow.Context, input any, state *utils.State) (map[string]any, error)

// testForkParentWorkflowID is the workflow ID the Temporal test
// environment gives the workflow under test. The fork builder derives
// its child workflow IDs from the parent's, so tests that need to
// address an individual branch build the ID from this.
const testForkParentWorkflowID = "default-test-workflow-id"

// forkSettleDelay is how long the host workflow lingers after a
// successful fork so that cancellations the fork issued as it returned
// are processed while the workflow is still running.
const forkSettleDelay = time.Second

// forkChildWorkflowID returns the child workflow ID the fork builder
// assigns to the named branch, so a test can cancel one branch without
// touching the parent or its siblings.
func forkChildWorkflowID(branch string) string {
	return testForkParentWorkflowID + "_fork_" + branch
}

// runForkExec executes the supplied fork branches inside a Temporal
// test environment. registerBranch maps each branch name to the child
// workflow function that should back it.
func runForkExec(
	t *testing.T,
	compete bool,
	branches map[string]forkBranch,
) (workflowErr error) {
	t.Helper()

	return runForkExecEnv(t, compete, branches, nil).GetWorkflowError()
}

// runForkExecEnv is runForkExec with access to the environment. setup
// runs after the workflows are registered and before execution, so a
// test can schedule delayed callbacks such as cancelling one branch;
// the completed environment is returned so a test can assert on the
// fork's result as well as its error.
func runForkExecEnv(
	t *testing.T,
	compete bool,
	branches map[string]forkBranch,
	setup func(env *testsuite.TestWorkflowEnvironment),
) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	forkedTasks := make([]*forkedTask, 0, len(branches))
	for name := range branches {
		forkedTasks = append(forkedTasks, &forkedTask{
			task:              &model.TaskItem{Key: name},
			childWorkflowName: "fork-" + name,
			taskName:          name,
		})
	}

	builder := &ForkTaskBuilder{
		name: "fork-task-end",
		task: &model.ForkTask{
			Fork: model.ForkTaskConfiguration{
				Compete: compete,
			},
		},
	}

	fn, err := builder.exec(forkedTasks)
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	for _, ft := range forkedTasks {
		impl := branches[ft.taskName]
		env.RegisterWorkflowWithOptions(impl, workflow.RegisterOptions{Name: ft.childWorkflowName})
	}

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		res, err := fn(ctx, nil, utils.NewState())
		if err != nil {
			return nil, err
		}

		// Stay alive briefly after a successful fork. A competing fork
		// cancels its losing branches on the way out, and the test
		// environment panics ("illegal access from outside of workflow
		// context") if a child cancellation callback is processed after
		// the workflow has completed. A real workflow carries on to its
		// next task, so settling here matches production rather than
		// papering over anything.
		if err := workflow.Sleep(ctx, forkSettleDelay); err != nil {
			return nil, err
		}

		return res, nil
	}, workflow.RegisterOptions{Name: "fork-exec-host"})

	if setup != nil {
		setup(env)
	}

	env.ExecuteWorkflow("fork-exec-host")
	return env
}

// TestForkTaskBuilderExecPropagatesEndFromBranch proves that a branch
// signalling `then: end` short-circuits the fork without being wrapped
// as "error forking task", and surfaces flow.ErrEnd carrying the
// branch's effective output. The other branches' eventual results
// must not be reported back.
func TestForkTaskBuilderExecPropagatesEndFromBranch(t *testing.T) {
	endingOutput := map[string]any{testConstValue: "branch-end-output"}

	// A single end-emitting branch is sufficient: the assertion is that
	// the fork as a whole surfaces flow.ErrEnd rather than wrapping the
	// signal as a fork failure. Returning endingOutput alongside the
	// end signal also satisfies the unparam lint by avoiding a function
	// whose result is always nil.
	workflowErr := runForkExec(t, false, map[string]forkBranch{
		"ending": func(_ workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return endingOutput, flow.NewEndApplicationError(endingOutput)
		},
	})

	require.Error(t, workflowErr)
	assert.Contains(t, workflowErr.Error(), flow.ErrEnd.Error())
	assert.NotContains(t, workflowErr.Error(), "error forking task",
		"a branch-emitted end must not be wrapped as a fork failure")
}

// TestForkTaskBuilderExecStillWrapsRealBranchFailure regresses normal
// fork error handling: a real branch failure must still surface as
// "error forking task" rather than being mistaken for end propagation.
func TestForkTaskBuilderExecStillWrapsRealBranchFailure(t *testing.T) {
	workflowErr := runForkExec(t, false, map[string]forkBranch{
		"boom": func(_ workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return nil, errors.New("genuine branch failure")
		},
	})

	require.Error(t, workflowErr)
	assert.Contains(t, workflowErr.Error(), "error forking task")
	assert.NotContains(t, workflowErr.Error(), flow.ErrEnd.Error())
}

// testForkTaskName is the fork task's own name (and sole path segment) used
// by the alias-derivation tests below.
const testForkTaskName = "dispatch"

// A single-task fork branch is wrapped in a synthetic child workflow before
// being built. The generated per-task activity alias must still be derived
// from the original, user-visible branch key ("branchA"), not from the
// synthetic child workflow name ("workflow_fork_dispatch_branchA").
//
// Only the original-key alias is registered as an expectation; the mock
// fails any RegisterActivityWithOptions call with a different name, so an
// alias built from the synthetic name would fail the test rather than pass
// silently.
func TestForkSingleTaskBranchAliasUsesOriginalBranchKey(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork-single"}}

	w := new(WorkflowRegistryMock)
	// The wrapper child workflow registration is incidental here.
	w.On("RegisterWorkflowWithOptions", mock.Anything, mock.Anything).Maybe()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork-single.dispatch.branchA",
		}).
		Once()

	forkTask := &model.ForkTask{
		Fork: model.ForkTaskConfiguration{
			Branches: &model.TaskList{
				&model.TaskItem{Key: "branchA", Task: newTestHTTPTask()},
			},
		},
	}

	b := &ForkTaskBuilder{
		doc:            doc,
		eventEmitter:   testEvents,
		name:           testForkTaskName,
		taskPath:       []string{testForkTaskName},
		task:           forkTask,
		temporalWorker: w,
	}

	_, err := b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

// A multi-task fork branch is a do-task scope: the branch key is an
// intermediate path segment and the body's leaf tasks nest beneath it. This
// pins the sibling behaviour the single-task case is kept consistent with.
func TestForkMultiTaskBranchAliasNestsUnderBranchKey(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork-multi"}}

	w := new(WorkflowRegistryMock)
	w.On("RegisterWorkflowWithOptions", mock.Anything, mock.Anything).Maybe()
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
		doc:            doc,
		eventEmitter:   testEvents,
		name:           testForkTaskName,
		taskPath:       []string{testForkTaskName},
		task:           forkTask,
		temporalWorker: w,
	}

	_, err := b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

// TestForkTaskBuilderExecPropagatesCancelledBranchInNonCompetingFork is
// the regression test for a fork that waits forever. A branch that is
// cancelled independently of the fork has terminated and will never
// reply, so if the fork keeps it in the "not yet replied" set the await
// condition can never be satisfied and the parent workflow blocks
// indefinitely.
//
// The cancellation must instead end the await and propagate with
// Temporal cancellation semantics intact. The test asserts the fork
// surfaces a cancellation and, crucially, that it did not simply run
// out of time waiting.
func TestForkTaskBuilderExecPropagatesCancelledBranchInNonCompetingFork(t *testing.T) {
	env := runForkExecEnv(t, false, map[string]forkBranch{
		// Blocks on a durable timer and reports the cancellation of its
		// own context, so the child closes as CANCELED.
		"slow": func(ctx workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return nil, workflow.Sleep(ctx, time.Hour)
		},
		// Replies straight away, so the fork is left waiting only on the
		// branch that gets cancelled.
		"fast": func(_ workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return map[string]any{testConstValue: "fast-done"}, nil
		},
	}, func(env *testsuite.TestWorkflowEnvironment) {
		env.RegisterDelayedCallback(func() {
			env.CancelWorkflowByID(forkChildWorkflowID("slow"), "")
		}, time.Second)
	})

	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr, "a cancelled branch must not be discarded")
	assert.True(t, temporal.IsCanceledError(workflowErr),
		"the branch cancellation must propagate as a cancellation, got: %v", workflowErr)
	assert.NotContains(t, workflowErr.Error(), "deadline exceeded",
		"the fork must stop waiting rather than block until the workflow times out")
}

// TestForkTaskBuilderExecCompetingLoserCancellationKeepsWinner proves
// the intentional side of fork cancellation is unaffected. In a
// competing fork the winner satisfies the await, exec then cancels the
// losing branches, and those cancellations must not turn a successful
// fork into a cancellation or a failure.
//
// This works without any special-casing because CancelOthers only runs
// after the await has been satisfied and exec has read the fork state,
// so a loser's cancellation is observed after the result is already
// settled.
func TestForkTaskBuilderExecCompetingLoserCancellationKeepsWinner(t *testing.T) {
	winningOutput := map[string]any{testConstValue: "winner"}

	env := runForkExecEnv(t, true, map[string]forkBranch{
		"winner": func(_ workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return winningOutput, nil
		},
		// Still running when the winner replies, so exec cancels it.
		"loser": func(ctx workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return nil, workflow.Sleep(ctx, time.Hour)
		},
	}, nil)

	require.NoError(t, env.GetWorkflowError(),
		"cancelling the losing branch must not fail the fork")

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, winningOutput, result,
		"the fork must return the winning branch's result")
}

// TestForkTaskBuilderExecPropagatesUnexpectedCancellationInCompetingFork
// covers the case that rules out treating every cancellation in a
// competing fork as intentional: a branch cancelled before any winner
// exists is an unexpected cancellation, not a loser being cleaned up.
//
// Discarding it would leave the fork waiting on the remaining branch
// with no way to ever satisfy the await.
func TestForkTaskBuilderExecPropagatesUnexpectedCancellationInCompetingFork(t *testing.T) {
	// Neither branch can ever reply on its own: one is cancelled, the
	// other blocks on a condition that is never met. If the cancellation
	// were discarded there would be no winner and nothing left to wake
	// the fork.
	blockForever := func(ctx workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
		return nil, workflow.Await(ctx, func() bool { return false })
	}

	env := runForkExecEnv(t, true, map[string]forkBranch{
		testConstTaskCancelled: blockForever,
		"stuck":                blockForever,
	}, func(env *testsuite.TestWorkflowEnvironment) {
		env.RegisterDelayedCallback(func() {
			env.CancelWorkflowByID(forkChildWorkflowID(testConstTaskCancelled), "")
		}, time.Second)
	})

	workflowErr := env.GetWorkflowError()
	require.Error(t, workflowErr, "an unexpected branch cancellation must not be discarded")
	assert.True(t, temporal.IsCanceledError(workflowErr),
		"the cancellation must propagate as a cancellation, got: %v", workflowErr)
	assert.NotContains(t, workflowErr.Error(), "deadline exceeded",
		"the fork must stop waiting rather than block until the workflow times out")
}

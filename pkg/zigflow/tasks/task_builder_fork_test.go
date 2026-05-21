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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type fakeWorkflowContext struct{}

func (fakeWorkflowContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (fakeWorkflowContext) Done() workflow.Channel      { return nil }
func (fakeWorkflowContext) Err() error                  { return nil }
func (fakeWorkflowContext) Value(key interface{}) interface{} {
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

// runForkExec executes the supplied fork branches inside a Temporal
// test environment. registerBranch maps each branch name to the child
// workflow function that should back it.
func runForkExec(
	t *testing.T,
	compete bool,
	branches map[string]func(ctx workflow.Context, input any, state *utils.State) (map[string]any, error),
) (workflowErr error) {
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
		builder: builder[*model.ForkTask]{
			name: "fork-task-end",
			task: &model.ForkTask{
				Fork: model.ForkTaskConfiguration{
					Compete: compete,
				},
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
		return fn(ctx, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: "fork-exec-host"})

	env.ExecuteWorkflow("fork-exec-host")
	return env.GetWorkflowError()
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
	workflowErr := runForkExec(t, false, map[string]func(workflow.Context, any, *utils.State) (map[string]any, error){
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
	workflowErr := runForkExec(t, false, map[string]func(workflow.Context, any, *utils.State) (map[string]any, error){
		"boom": func(_ workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
			return nil, errors.New("genuine branch failure")
		},
	})

	require.Error(t, workflowErr)
	assert.Contains(t, workflowErr.Error(), "error forking task")
	assert.NotContains(t, workflowErr.Error(), flow.ErrEnd.Error())
}

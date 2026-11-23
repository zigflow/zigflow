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

// Package cancelfork exercises cancelling a workflow while a non-competing
// fork is running. The workflow is `set` -> fork of two long `wait`
// branches -> `set`. Cancelling it while both branches wait must cancel
// each branch, must not run the trailing set task, and must close the
// execution and both branches as CANCELED.
//
// Regression test for the bug where the fork closed in the same workflow
// task that requested the branch cancellations, so the parent close
// policy terminated the branches before they received the cancellation.
package cancelfork

import (
	"context"
	"testing"
	"time"

	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/tests/e2e/utils"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
)

// workflowID is fixed so the test can address the execution and derive
// the branch workflow IDs the fork assigns.
const workflowID = "cancel-fork-cancels-branches"

// startupTimeout bounds the wait for the worker to come up and both
// branches to reach their timers.
const startupTimeout = 2 * time.Minute

// cancelTimeout bounds the wait for the execution to close after
// cancellation is requested. The branch waits are ten minutes, so closing
// inside this window proves the cancellation interrupted them.
const cancelTimeout = 30 * time.Second

// trailingStage is the value the trailing set task would record. It must
// never appear anywhere in the execution history.
const trailingStage = "after_fork_ran"

// branches are the fork's branch keys, from which the fork derives each
// branch's child workflow ID.
var branches = []string{"left", "right"}

var testCase = utils.TestCase{
	Name:         "cancel-fork",
	WorkflowPath: "workflow.yaml",
	Test: func(t *testing.T, test *utils.TestCase) {
		c, err := temporal.NewConnectionWithEnvvars(
			temporal.WithZerolog(&zlog.Logger),
		)
		require.NoError(t, err)
		defer c.Close()

		wCtx := context.Background()

		we, err := c.ExecuteWorkflow(wCtx, client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: test.Workflow.Document.Namespace,
		}, test.Workflow.Document.Name)
		require.NoError(t, err)

		runID := we.GetRunID()

		// Terminate on the way out so a failing assertion cannot leave
		// ten minute timers running against the shared Temporal server.
		// Terminating the parent terminates any branch still open.
		t.Cleanup(func() {
			_ = c.TerminateWorkflow(context.Background(), workflowID, runID, "cancel-fork test cleanup")
		})

		// Cancel only once both branches are genuinely blocked on their
		// timers, so the test does not depend on an arbitrary sleep.
		for _, branch := range branches {
			waitForBranchTimer(t, c, wCtx, branchWorkflowID(branch))
		}

		require.NoError(t, c.CancelWorkflow(wCtx, workflowID, runID))

		closeCtx, closeCancel := context.WithTimeout(wCtx, cancelTimeout)
		defer closeCancel()

		var result any
		err = we.Get(closeCtx, &result)
		require.Error(t, err, "a cancelled workflow must not return a result")
		assert.True(t, sdktemporal.IsCanceledError(err),
			"the client must observe the execution as cancelled, got: %v", err)

		assertCanceled(t, c, wCtx, workflowID, runID)

		// The reported bug: the branches closed as TERMINATED by the
		// parent close policy instead of receiving the cancellation.
		for _, branch := range branches {
			assertCanceled(t, c, wCtx, branchWorkflowID(branch), "")
		}

		assertTrailingTaskDidNotRun(t, c, wCtx, runID)
	},
}

// branchWorkflowID returns the child workflow ID the fork assigns to a
// branch.
func branchWorkflowID(branch string) string {
	return workflowID + "_fork_" + branch
}

// waitForBranchTimer polls a branch's history until its wait timer has
// started. The branch may not exist yet when polling begins.
func waitForBranchTimer(t *testing.T, c client.Client, ctx context.Context, id string) {
	t.Helper()

	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		iter := c.GetWorkflowHistory(ctx, id, "", false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		for iter.HasNext() {
			event, err := iter.Next()
			if err != nil {
				break
			}
			if event.GetEventType() == enums.EVENT_TYPE_TIMER_STARTED {
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("branch %q did not reach its wait within %s", id, startupTimeout)
}

// assertCanceled checks that Temporal recorded the execution as CANCELED.
func assertCanceled(t *testing.T, c client.Client, ctx context.Context, id, runID string) {
	t.Helper()

	desc, err := c.DescribeWorkflow(ctx, id, runID)
	require.NoError(t, err)
	assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_CANCELED, desc.Status,
		"Temporal must record %q as CANCELED", id)
}

// assertTrailingTaskDidNotRun checks the closed parent's history for any
// sign that the task after the fork ran.
func assertTrailingTaskDidNotRun(t *testing.T, c client.Client, ctx context.Context, runID string) {
	t.Helper()

	iter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		require.NoError(t, err, "read workflow history")

		assert.NotEqual(t, enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED, event.GetEventType(),
			"a cancelled workflow must not complete")
		assert.NotContains(t, event.String(), trailingStage,
			"the task after the cancelled fork must not have run")
	}
}

func init() {
	utils.AddTestCase(&testCase)
}

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

// Package cancel exercises workflow cancellation. The workflow is
// `set` -> long `wait` -> `set`. Cancelling it while the wait is running
// must interrupt the wait, must not run the trailing set task, and must
// close the execution as CANCELED.
//
// Regression test for the bug where the cancelled wait was reported as a
// successful task, the do loop carried on, and the workflow closed as
// COMPLETED with the trailing task's output.
package cancel

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

// workflowID is fixed so the test can address the execution directly
// when requesting cancellation and reading its history.
const workflowID = "cancel-stops-remaining-tasks"

// startupTimeout bounds the wait for the worker to come up and the
// workflow to reach its timer.
const startupTimeout = 2 * time.Minute

// cancelTimeout bounds the wait for the execution to close after
// cancellation is requested. The workflow's own wait is ten minutes, so
// closing inside this window is itself proof that the cancellation
// interrupted the timer rather than the timer expiring.
const cancelTimeout = 30 * time.Second

// trailingStage is the value the trailing set task would record. It must
// never appear anywhere in the execution history.
const trailingStage = "after_wait_ran"

var testCase = utils.TestCase{
	Name:         "cancel",
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

		// Terminate on the way out so a failing assertion cannot leave a
		// ten minute timer running against the shared Temporal server.
		t.Cleanup(func() {
			_ = c.TerminateWorkflow(context.Background(), workflowID, runID, "cancel test cleanup")
		})

		// Establish deterministically that the workflow has reached the
		// wait: the timer only appears in history once the wait task has
		// started it. No arbitrary sleep is involved.
		waitForTimerStarted(t, c, wCtx, runID)

		require.NoError(t, c.CancelWorkflow(wCtx, workflowID, runID))

		// The wait is ten minutes; a cancellation that only took effect
		// when the timer expired would blow this deadline.
		closeCtx, closeCancel := context.WithTimeout(wCtx, cancelTimeout)
		defer closeCancel()

		var result any
		err = we.Get(closeCtx, &result)
		require.Error(t, err, "a cancelled workflow must not return a result")
		assert.True(t, sdktemporal.IsCanceledError(err),
			"the client must observe the execution as cancelled, got: %v", err)

		desc, err := c.DescribeWorkflowExecution(wCtx, workflowID, runID)
		require.NoError(t, err)
		assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_CANCELED, desc.GetWorkflowExecutionInfo().GetStatus(),
			"Temporal must record the execution as CANCELED")

		// Prove the trailing set task never ran. If it had, the do loop
		// would have finished and the execution would have closed as
		// COMPLETED carrying the trailing stage as its result, exactly
		// the reported bug.
		assertTrailingTaskDidNotRun(t, c, wCtx, runID)
	},
}

// waitForTimerStarted polls the execution history until the wait task's
// timer has been started, so the test only requests cancellation once
// the workflow is genuinely blocked on the wait.
func waitForTimerStarted(t *testing.T, c client.Client, ctx context.Context, runID string) {
	t.Helper()

	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		iter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		for iter.HasNext() {
			event, err := iter.Next()
			require.NoError(t, err, "read workflow history")

			if event.GetEventType() == enums.EVENT_TYPE_TIMER_STARTED {
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("workflow %q did not reach the wait within %s", workflowID, startupTimeout)
}

// assertTrailingTaskDidNotRun checks the closed execution's history for
// any sign that the task after the cancelled wait ran: the execution
// must not have completed, and the trailing task's value must appear
// nowhere in the recorded events.
func assertTrailingTaskDidNotRun(t *testing.T, c client.Client, ctx context.Context, runID string) {
	t.Helper()

	iter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		require.NoError(t, err, "read workflow history")

		assert.NotEqual(t, enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED, event.GetEventType(),
			"a cancelled workflow must not complete")
		assert.NotContains(t, event.String(), trailingStage,
			"the task after the cancelled wait must not have run")
	}
}

func init() {
	utils.AddTestCase(&testCase)
}

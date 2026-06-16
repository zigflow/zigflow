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

// Package continueasnew is a regression test for message routing across
// Continue-As-New.
//
// The workflow forces Continue-As-New early (via canMaxHistoryLength), so the
// run that the test interacts with is the continued run, not the run that was
// originally started. The test then sends a signal and issues a query against
// the same Workflow ID with an empty Run ID, which Temporal routes to the
// current (continued) execution. This proves the signal and query handlers are
// preserved and re-registered across Continue-As-New.
//
// The test intentionally targets the continued execution via Workflow ID. It
// never references a Run ID when signalling or querying, so the routing under
// test is the same routing a real client would rely on.
//
// Observation is asserted through the query handler rather than the final
// workflow output. The signal value is carried into query state the moment the
// continued run receives it, so the query is the direct evidence that the
// message reached the continued execution. Asserting on completion instead
// would be unreliable: listen tasks are neverSkipCAN, so a Continue-As-New that
// fires immediately after the signal causes the listener to re-run and block
// again on the next run, and the workflow does not necessarily complete.
//
// TODO: assert end-to-end completion once Zigflow handles a Continue-As-New
// that happens after a signal has already been consumed (today the re-run
// listener blocks again waiting for a fresh signal).
//
// TODO: extend this to cover update handlers once the signal + query path is
// proven stable. Updates are blocking like signals but are validated and
// replied to synchronously, so they need their own assertions.
package continueasnew

import (
	"context"
	"testing"
	"time"

	"github.com/mrsimonemms/golang-helpers/temporal"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/tests/e2e/utils"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// workflowID is deterministic so the test can address the execution by Workflow
// ID alone, exactly as a real client would when it does not know the Run ID.
const workflowID = "continue-as-new-message-routing"

// canTimeout bounds how long the test waits for Continue-As-New to roll the
// execution over to a new Run ID. The workflow's burnHistory timer plus the
// Continue-As-New round trip should complete well within this.
const canTimeout = 60 * time.Second

// queryTimeout bounds how long the test retries the query against the continued
// run while the listener is being re-registered after Continue-As-New.
const queryTimeout = 30 * time.Second

var testCase = utils.TestCase{
	Name:         "continue-as-new",
	WorkflowPath: "workflow.yaml",
	Test: func(t *testing.T, test *utils.TestCase) {
		c, err := temporal.NewConnectionWithEnvvars(
			temporal.WithZerolog(&zlog.Logger),
		)
		require.NoError(t, err)
		defer c.Close()

		wCtx := context.Background()

		// Start the workflow with a deterministic Workflow ID. The returned
		// run ID is the *original* run; Continue-As-New will replace it.
		//
		// This workflow does not complete on its own (the signal listener
		// re-blocks after Continue-As-New, see the package doc), so it is left
		// running between test invocations. Terminating any existing execution
		// guarantees each run starts from a clean execution rather than
		// attaching to a leftover one stuck on the same deterministic Workflow
		// ID.
		we, err := c.ExecuteWorkflow(wCtx, client.StartWorkflowOptions{
			ID:                       workflowID,
			TaskQueue:                test.Workflow.Document.Namespace,
			WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING,
		}, test.Workflow.Document.Name)
		require.NoError(t, err)

		// Terminate at the end so the non-completing workflow does not linger.
		t.Cleanup(func() {
			_ = c.TerminateWorkflow(context.Background(), workflowID, "", "continue-as-new test cleanup")
		})

		originalRunID := we.GetRunID()
		require.NotEmpty(t, originalRunID, "original run ID")
		t.Logf("started workflow %q, original run ID %q", workflowID, originalRunID)

		// Prove Continue-As-New happened: the current run ID for this Workflow
		// ID must differ from the original run ID.
		continuedRunID := waitForNewRun(t, c, wCtx, originalRunID)
		t.Logf("workflow continued as new, continued run ID %q", continuedRunID)

		// Query the same Workflow ID with an empty Run ID before sending the
		// signal. This must route to the continued run and succeed, proving the
		// query handler was re-registered across Continue-As-New. The signal has
		// not arrived yet, so the observed value is null.
		before := queryContinuedRun(t, c, wCtx)
		t.Logf("query before signal: %v", before)
		_, hasSignalKey := before["signal"]
		assert.True(t, hasSignalKey, "query state should expose the signal field")

		// Send a signal to the same Workflow ID with an empty Run ID. Temporal
		// routes this to the continued run; the empty Run ID is the whole point
		// of the test.
		require.NoError(
			t,
			c.SignalWorkflow(wCtx, workflowID, "", "approve", true),
			"signal continued run by workflow ID",
		)

		// Assert the continued run observed the signal. The query is polled by
		// Workflow ID with an empty Run ID, so it always reflects the current
		// run's state; the signal value appears as soon as the continued run
		// receives it. This is the core assertion: a message addressed to the
		// Workflow ID after Continue-As-New reaches the continued execution.
		assertSignalObserved(t, c, wCtx)
	},
}

// waitForNewRun polls the current run ID for the workflow until it differs from
// originalRunID (proving Continue-As-New occurred) and then stays unchanged for
// stableChecks consecutive polls (proving the continued run has settled and is
// not about to continue-as-new again). Waiting for a stable run ID means the
// signal the test sends next is addressed to the run that will actually consume
// it, rather than to a run that is replaced moments later. It fails the test if
// no stable continued run is observed within canTimeout.
func waitForNewRun(t *testing.T, c client.Client, ctx context.Context, originalRunID string) string {
	t.Helper()

	const stableChecks = 4

	deadline := time.Now().Add(canTimeout)
	var candidate string
	stable := 0
	for time.Now().Before(deadline) {
		desc, err := c.DescribeWorkflowExecution(ctx, workflowID, "")
		require.NoError(t, err, "describe workflow execution")

		currentRunID := desc.GetWorkflowExecutionInfo().GetExecution().GetRunId()
		if currentRunID != "" && currentRunID != originalRunID {
			if currentRunID == candidate {
				stable++
				if stable >= stableChecks {
					return currentRunID
				}
			} else {
				// A fresh run ID: Continue-As-New happened (again). Restart the
				// stability count against this new run.
				candidate = currentRunID
				stable = 1
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("workflow %q did not settle on a stable continued run within %s (original run ID %q, last seen %q)",
		workflowID, canTimeout, originalRunID, candidate)
	return ""
}

// queryContinuedRun queries the workflow by Workflow ID with an empty Run ID,
// retrying until the query handler is registered on the continued run or
// queryTimeout elapses. The query is addressed by Workflow ID only, so a
// success proves the continued run answered it.
func queryContinuedRun(t *testing.T, c client.Client, ctx context.Context) map[string]any {
	t.Helper()

	deadline := time.Now().Add(queryTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := c.QueryWorkflow(ctx, workflowID, "", "get_state")
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var state map[string]any
		require.NoError(t, resp.Get(&state), "decode query result")
		return state
	}

	t.Fatalf("query against continued run %q did not succeed within %s: %v",
		workflowID, queryTimeout, lastErr)
	return nil
}

// assertSignalObserved polls the query handler by Workflow ID (empty Run ID)
// until the observed signal value is true, proving the continued run received
// the signal. It fails the test if the signal is never observed within
// queryTimeout, which is the meaningful failure if Zigflow does not route a
// signal addressed to the Workflow ID to the continued run.
func assertSignalObserved(t *testing.T, c client.Client, ctx context.Context) {
	t.Helper()

	deadline := time.Now().Add(queryTimeout)
	var lastState map[string]any
	for time.Now().Before(deadline) {
		resp, err := c.QueryWorkflow(ctx, workflowID, "", "get_state")
		if err != nil {
			// The continued run may briefly be unqueryable while it rolls over;
			// keep polling until the deadline.
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var state map[string]any
		require.NoError(t, resp.Get(&state), "decode query result")
		lastState = state

		if state["signal"] == true {
			t.Logf("signal observed by continued run: %v", state)
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("continued run never observed the signal within %s (last query state: %v)",
		queryTimeout, lastState)
}

func init() {
	utils.AddTestCase(&testCase)
}

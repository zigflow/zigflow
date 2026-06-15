//go:build e2e

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

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// Happy-path transfer input, matching the shape the money-transfer UI sends
// when starting AccountTransferWorkflow: an amount and the two accounts.
const (
	transferAmount = 100
	fromAccount    = "account-1"
	toAccount      = "account-2"

	// workflowType and taskQueue mirror the named workflow and document task
	// queue in workflow.yaml. The UI starts this exact workflow type.
	workflowType = "AccountTransferWorkflow"
	taskQueue    = "MoneyTransfer"

	// transferStatusQuery is the query the UI polls for progress; the workflow
	// registers it via the queryState listener (id: transferStatus).
	transferStatusQuery = "transferStatus"
)

// TestMoneyTransferHappyPathE2E runs the money-transfer AccountTransferWorkflow
// happy path end to end. The example is normally driven by a UI rather than a
// main.go starter, so the test acts like the UI: it starts the workflow through
// the Temporal Go client, polls the transferStatus query for progress, and
// waits for completion. No browser or UI automation is involved.
//
// The workflow's HTTP backend (the compose "server" service) is replaced by a
// local recording mock so the run is hermetic and the test can assert which
// calls were made, including that the activity attempt counter reaches the
// backend rather than being silently null.
func TestMoneyTransferHappyPathE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	// The backend endpoints all reply 200 "SUCCESS", mirroring the real server.
	backend := e2etest.StartHTTPRecorder(t, "SUCCESS")

	workflowFile := writeHostWorkflow(t, backend.URL)

	e2etest.StartWorker(ctx, t, temporal.Address, workflowFile)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	// The workflow runs roughly six seconds of sleeps plus the backend calls.
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Start the workflow exactly as the UI would: the AccountTransferWorkflow
	// type on the MoneyTransfer task queue, with an {amount, fromAccount,
	// toAccount} input.
	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		ID:        "money-transfer-e2e",
		TaskQueue: taskQueue,
	}, workflowType, map[string]any{
		"amount":      transferAmount,
		"fromAccount": fromAccount,
		"toAccount":   toAccount,
	})
	require.NoError(t, err, "execute workflow")

	// Act like the UI polling for progress: the transferStatus query handler is
	// registered early but not at the very first instant, so poll until it
	// answers. Queries also work after completion, so this never deadlocks.
	state := pollTransferStatus(runCtx, t, c, we.GetID())
	e2etest.AssertNumeric(t, state["progressPercentage"])
	e2etest.AssertNonEmptyString(t, state["transferState"])

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")

	pretty, _ := json.MarshalIndent(got, "", "  ")
	t.Logf("workflow output:\n%s", pretty)

	assertTransferResult(t, got)
	assertBackendCalls(t, backend)
}

func assertTransferResult(t *testing.T, got map[string]any) {
	t.Helper()

	// Stable terminal state: the transfer finished at 100% with a 30s approval
	// window seeded at setup.
	assert.Equal(t, "finished", got["stateTransferState"], "transfer reaches the finished state")
	assert.Equal(t, float64(100), got["stateProgressPercentage"], "transfer completes at 100%")
	assert.Equal(t, float64(30), got["stateApprovalTime"], "approval time carried through")

	// Run-specific values: shape only.
	e2etest.AssertValidUUID(t, got["idempotencyKey"])
	e2etest.AssertValidUUID(t, got["stateChargeId"])

	// No state field should carry an unresolved ${ ... } expression string.
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "${", "output must not contain unresolved expressions")
}

func assertBackendCalls(t *testing.T, backend *e2etest.HTTPRecorder) {
	t.Helper()

	// The happy path calls each backend endpoint exactly once, in order.
	for _, path := range []string{"/validate", "/withdraw", "/deposit", "/notify"} {
		assert.Len(t, backend.RequestsForPath(path), 1, "one call to %s", path)
	}

	// withdraw and deposit carry the activity attempt counter. Assert it is
	// present and a positive number (shape, since attempt is runtime metadata)
	// rather than silently null - this guards the activity attempt expression.
	for _, path := range []string{"/withdraw", "/deposit"} {
		reqs := backend.RequestsForPath(path)
		require.Len(t, reqs, 1, "exactly one %s call to inspect", path)

		body := decodeJSONBody(t, reqs[0].Body)

		require.Contains(t, body, "attempt", "%s body should include attempt", path)
		attempt := e2etest.AssertNumeric(t, body["attempt"])
		assert.GreaterOrEqual(t, attempt, float64(1), "%s attempt should be a positive number", path)

		// The other activity-input fields the backend requires must also be
		// present and well-formed.
		e2etest.AssertValidUUID(t, body["idempotencyKey"])
		assert.Equal(t, workflowType, body["name"], "%s name is the workflow type", path)
		assert.Equal(t, float64(transferAmount), body["amount"], "%s amount echoes the input", path)
	}

	// The notification carries the transfer parties.
	notify := backend.RequestsForPath("/notify")
	require.Len(t, notify, 1, "one notify call")
	notifyBody := decodeJSONBody(t, notify[0].Body)
	assert.Equal(t, float64(transferAmount), notifyBody["amount"], "notify amount")
	assert.Equal(t, fromAccount, notifyBody["fromAccount"], "notify fromAccount")
	assert.Equal(t, toAccount, notifyBody["toAccount"], "notify toAccount")
}

// pollTransferStatus queries the transferStatus query until it answers, mirroring
// the UI's polling. It returns the decoded query state.
func pollTransferStatus(ctx context.Context, t *testing.T, c client.Client, workflowID string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := c.QueryWorkflow(ctx, workflowID, "", transferStatusQuery)
		if err == nil {
			var state map[string]any
			require.NoError(t, resp.Get(&state), "decode transferStatus query")
			t.Logf("transferStatus query: %v", state)
			return state
		}

		require.True(t, time.Now().Before(deadline), "transferStatus query never answered: %v", err)
		time.Sleep(250 * time.Millisecond)
	}
}

// writeHostWorkflow reads the canonical workflow.yaml and rewrites only the
// backend endpoint host, leaving the workflow semantics untouched. The committed
// workflow targets the Docker Compose topology (the backend at
// http://server:3000); this adapts that runtime wiring for a host-side worker
// reaching the local recording mock instead. The patched copy is written to a
// temp file and its path returned.
//
// The rewrite is guarded with an exact-count assertion so the test fails loudly
// if workflow.yaml drifts from the host:port this depends on, rather than
// silently testing the unpatched (container-only) value.
func writeHostWorkflow(t *testing.T, backendURL string) string {
	t.Helper()

	source, err := os.ReadFile("workflow.yaml")
	require.NoError(t, err, "read workflow.yaml")

	const composeBackend = "http://server:3000"
	const wantCount = 4 // validate, withdraw, deposit, notify

	patched := string(source)
	require.Equalf(t, wantCount, strings.Count(patched, composeBackend),
		"expected %d occurrences of %q in workflow.yaml; the example may have drifted", wantCount, composeBackend)
	patched = strings.ReplaceAll(patched, composeBackend, backendURL)

	// dest is rooted at the test's own temp dir with a fixed name, so the path
	// is fully test-controlled despite gosec's taint analysis flagging it.
	dest := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(dest, []byte(patched), 0o600), "write patched workflow") //nolint:gosec // test-controlled path

	return dest
}

// decodeJSONBody parses a recorded request body into a map for field assertions.
func decodeJSONBody(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var out map[string]any
	require.NoErrorf(t, json.Unmarshal(body, &out), "decode request body: %s", body)
	return out
}

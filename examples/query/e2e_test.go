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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// TestQueryE2E runs the query example. The query listener is non-blocking, so
// the workflow progresses on its own; the test issues a query mid-run to
// exercise the handler, then asserts the final state once it completes.
func TestQueryE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	e2etest.StartWorker(ctx, t, temporal.Address, workflowFile)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "query")
	require.NoError(t, err, "execute workflow")

	// The query listener is non-blocking, so the workflow sets its state and
	// progresses on its own. Give it a moment, then query the state to exercise
	// the query handler. The workflow's final output is intentionally empty, so
	// the query is the meaningful assertion here.
	time.Sleep(3 * time.Second)
	resp, err := c.QueryWorkflow(runCtx, we.GetID(), "", "get_state")
	require.NoError(t, err, "query get_state")

	var state map[string]any
	require.NoError(t, resp.Get(&state), "decode query result")
	t.Logf("query state: %v", state)

	e2etest.AssertValidUUID(t, state["id"])
	e2etest.AssertNonEmptyString(t, state["status"])
	e2etest.AssertNumeric(t, state["progressPercentage"])

	// The workflow should still run to completion.
	assert.NoError(t, we.Get(runCtx, nil), "query workflow should complete")
}

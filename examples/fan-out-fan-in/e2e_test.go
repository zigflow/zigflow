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
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// TestFanOutFanInE2E runs the fan-out-fan-in example. Three branches each fetch
// a dataset concurrently and reduce it to a count; the workflow then aggregates
// them into a single total.
func TestFanOutFanInE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	mock := e2etest.StartHTTPSMock(t, []string{"jsonplaceholder.typicode.com"}, map[string]any{
		"/posts":    []any{1, 2},       // length 2
		"/comments": []any{1, 2, 3},    // length 3
		"/albums":   []any{1, 2, 3, 4}, // length 4
	})

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	temporalHost, _, err := net.SplitHostPort(temporal.Address)
	require.NoError(t, err)

	e2etest.StartWorkerWithEnv(ctx, t, mock.WorkerEnv(temporalHost), temporal.Address, workflowFile)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "fan-out-fan-in")
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	// 2 + 3 + 4 across the three concurrent branches.
	assert.EqualValues(t, 9, e2etest.AssertNumeric(t, got["total"]), "aggregated total of all branches")
	assert.Len(t, got["datasets"], 3, "all three branches contributed")
}

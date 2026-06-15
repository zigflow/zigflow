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

// TestSignalE2E runs the signal example, which blocks until an "approve" signal
// is received and accepted. The test sends an accepted signal so the workflow
// can complete.
func TestSignalE2E(t *testing.T) {
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
	}, "signal")
	require.NoError(t, err, "execute workflow")

	// acceptIf requires a truthy payload; sending true is accepted and unblocks
	// the listener. Signals are buffered, so sending promptly is safe.
	require.NoError(t, c.SignalWorkflow(runCtx, we.GetID(), "", "approve", true), "signal workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	response, ok := got["response"].(map[string]any)
	require.True(t, ok, "response should be an object")
	assert.Equal(t, true, response["signal"], "signal value carried into output")
}

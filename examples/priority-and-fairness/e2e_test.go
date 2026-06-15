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

// TestPriorityAndFairnessE2E runs the priority-and-fairness example with a
// single workflow. The priority/fairness keys influence Temporal scheduling but
// the workflow output simply echoes the input alongside a fixed message.
func TestPriorityAndFairnessE2E(t *testing.T) {
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
	}, "priority-fairness", map[string]any{
		"priorityKey":    1,
		"fairnessKey":    "tenant-a",
		"fairnessWeight": 1,
	})
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	data, ok := got["data"].(map[string]any)
	require.True(t, ok, "data should be an object")
	assert.Equal(t, "Priority and Fairness demo", data["message"], "data.message")
	assert.Equal(t, float64(1), data["priorityKey"], "data.priorityKey")
	assert.Equal(t, "tenant-a", data["fairnessKey"], "data.fairnessKey")
}

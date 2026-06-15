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

// TestUpdateE2E runs the update example, which waits for two accepted updates
// (temperature and bpm) before completing. The test sends values that satisfy
// each update's acceptIf validator.
func TestUpdateE2E(t *testing.T) {
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
	}, "updates", map[string]any{"userId": 3})
	require.NoError(t, err, "execute workflow")

	sendUpdate(runCtx, t, c, we.GetID(), "temperature", 39)
	sendUpdate(runCtx, t, c, we.GetID(), "bpm", 130)

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	assert.Equal(t, float64(39), got["temperature"], "temperature carried into output")
	assert.Equal(t, float64(130), got["bpm"], "bpm carried into output")
}

func sendUpdate(ctx context.Context, t *testing.T, c client.Client, workflowID, name string, value any) {
	t.Helper()

	handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   name,
		WaitForStage: client.WorkflowUpdateStageCompleted,
		Args:         []any{value},
	})
	require.NoError(t, err, "send %s update", name)

	var res any
	require.NoError(t, handle.Get(ctx, &res), "get %s update result", name)
}

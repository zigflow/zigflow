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

// TestActivityCallE2E runs the activity-call example, whose activities are
// served by the example's own Go worker on a separate task queue. The test
// starts both the Zigflow workflow worker and that activity worker against the
// same Temporal instance.
func TestActivityCallE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	e2etest.StartWorker(ctx, t, temporal.Address, workflowFile)
	// The activities run on the activity-call-worker queue served by the
	// example's own Go worker.
	e2etest.StartGoWorker(ctx, t, temporal.Address, "./worker")

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "activity-call", map[string]any{"userId": "u-123"})
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	// workflowId is the generated request UUID; message comes from the activity.
	e2etest.AssertValidUUID(t, got["workflowId"])
	e2etest.AssertNonEmptyString(t, got["message"])

	profile, ok := got["profile"].(map[string]any)
	require.True(t, ok, "profile should be an object")
	assert.Equal(t, "u-123", profile["userId"], "profile.userId echoes the input")
	e2etest.AssertNonEmptyString(t, profile["tier"])
}

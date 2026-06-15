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

// TestCompetingConcurrentTasksE2E runs the competing-tasks example. Two forked
// branches race; the one calling /users/2 waits 2s and the other waits 50s, so
// the first reliably wins. Both call jsonplaceholder, served here by a local
// mock to keep the test hermetic.
func TestCompetingConcurrentTasksE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	mock := e2etest.StartHTTPSMock(t, []string{"jsonplaceholder.typicode.com"}, map[string]any{
		"/users/1": map[string]any{"id": 1, "name": "Leanne Graham"},
		"/users/2": map[string]any{"id": 2, "name": "Ervin Howell"},
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
	}, "competing-tasks")
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	assert.Equal(t, "task1", got["winner"], "the 2s branch should win the race")
	e2etest.AssertValidUUID(t, got["id"])
}

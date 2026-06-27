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

// TestErrorFallbackE2E runs the error-fallback example. The primary endpoint
// always returns HTTP 500, so after the retry policy is exhausted the catch
// block calls the backup endpoint. The workflow completes successfully using
// the fallback result.
func TestErrorFallbackE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	mock := e2etest.StartHTTPSMock(t, []string{"httpbin.org", "jsonplaceholder.typicode.com"}, map[string]any{
		"/status/500": e2etest.MockResponse{Status: 500, Body: map[string]any{"error": "unavailable"}},
		"/users/1":    map[string]any{"name": "Ada"},
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
	}, "error-fallback")
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	// The primary failed every attempt, so the workflow recovered via the
	// backup service and still completed.
	assert.Equal(t, "fallback", got["source"], "should have recovered via the fallback")
	assert.Equal(t, "Ada", got["name"], "fallback result carried through")
	assert.Equal(t, true, got["complete"], "workflow continued past the recovered error")
}

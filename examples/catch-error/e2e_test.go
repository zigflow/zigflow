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

// TestCatchErrorE2E runs the catch-error example, which calls an endpoint that
// returns HTTP 418 and catches the resulting error. The mock returns 418 for
// /status/418 so the test never touches the public internet, and the workflow
// completes via its catch block.
func TestCatchErrorE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	mock := e2etest.StartHTTPSMock(t, []string{"httpbin.org"}, map[string]any{
		"/status/418": e2etest.MockResponse{Status: 418, Body: map[string]any{"teapot": true}},
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
	}, "catch-error")
	require.NoError(t, err, "execute workflow")

	// The error is caught, so the workflow completes and exposes the captured
	// data under the "data" key.
	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")
	t.Logf("workflow output: %v", got)

	assert.Contains(t, got, "data", "caught error data should be exposed")
}

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

// scheduleID matches document.metadata.scheduleId in workflow.yaml.
const scheduleID = "some-schedule"

// TestScheduleE2E covers the schedule example, which creates a Temporal schedule
// rather than running a workflow to completion. The Zigflow worker registers the
// schedule on startup, so the test asserts the schedule exists rather than
// awaiting a workflow result.
func TestScheduleE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	// scheduleInput references $env.EXAMPLE_ENVVAR, populated from ZIGGY_*.
	e2etest.StartWorkerWithEnv(
		ctx, t,
		[]string{"ZIGGY_EXAMPLE_ENVVAR=some-example-envvar"},
		temporal.Address, workflowFile,
	)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// The worker creates the schedule during startup, before reporting ready.
	// A successful Describe proves the schedule was registered.
	desc, err := c.ScheduleClient().GetHandle(runCtx, scheduleID).Describe(runCtx)
	require.NoError(t, err, "describe schedule")

	assert.NotNil(t, desc.Schedule.Action, "schedule should have an action")
}

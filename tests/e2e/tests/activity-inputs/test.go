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

// Package activityinputs checks that activities are scheduled with their inputs
// resolved by the workflow, that a resolved value which looks like an
// expression reaches the activity as text, and that $data.activity is left for
// the activity to evaluate.
package activityinputs

import (
	"context"
	"os"
	"testing"

	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/tests/e2e/utils"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

const (
	// Workflow input that reads like an expression: data, never evaluated.
	note        = "${ $env.HTTP_MOCK }"
	attemptExpr = "${ $data.activity.attempt }"
)

var testCase = utils.TestCase{
	Name:         "activity-inputs",
	WorkflowPath: "workflow.yaml",
	Test: func(t *testing.T, test *utils.TestCase) {
		c, err := temporal.NewConnectionWithEnvvars(temporal.WithZerolog(&zlog.Logger))
		require.NoError(t, err)
		defer c.Close()

		ctx := context.Background()
		run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
			TaskQueue: test.Workflow.Document.Namespace,
		}, test.Workflow.Document.Name, map[string]any{"id": "2", "note": note})
		require.NoError(t, err)

		var output map[string]any
		require.NoError(t, run.Get(ctx, &output))
		assert.Equal(t, map[string]any{
			"getPost": map[string]any{"id": "2", "title": "another title", "views": float64(200)},
			"echo":    note + " 1",
		}, output)

		scheduled := scheduledActivities(t, c, run.GetID(), run.GetRunID())
		require.Len(t, scheduled, 2)

		with := scheduled[0].task["with"].(map[string]any)
		assert.Equal(t, "http://"+os.Getenv("ZIGGY_HTTP_MOCK")+"/posts/2", with["endpoint"])
		assert.Equal(t, map[string]any{"x-note": note, "x-attempt": attemptExpr}, with["headers"])
		assert.Equal(t, map[string]any{"deferred": []any{"/headers/x-attempt"}}, scheduled[0].state["activityInputs"])

		shell := scheduled[1].task["run"].(map[string]any)["shell"].(map[string]any)
		assert.Equal(t, []any{note, attemptExpr}, shell["arguments"])
		assert.Equal(t, map[string]any{"deferred": []any{"/exec/args/1"}}, scheduled[1].state["activityInputs"])
	},
}

type scheduledActivity struct {
	task  map[string]any
	state map[string]any
}

// scheduledActivities returns the task and state of every activity scheduled,
// as recorded in the workflow history.
func scheduledActivities(t *testing.T, c client.Client, workflowID, runID string) []scheduledActivity {
	t.Helper()

	var out []scheduledActivity
	iter := c.GetWorkflowHistory(context.Background(), workflowID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		require.NoError(t, err)
		attrs := event.GetActivityTaskScheduledEventAttributes()
		if attrs == nil {
			continue
		}
		payloads := attrs.GetInput().GetPayloads()
		var a scheduledActivity
		require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payloads[0], &a.task))
		require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payloads[2], &a.state))
		out = append(out, a)
	}
	return out
}

func init() {
	utils.AddTestCase(&testCase)
}

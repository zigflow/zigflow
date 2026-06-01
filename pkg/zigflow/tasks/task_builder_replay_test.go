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

package tasks

import (
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// RELEASE ORDERING NOTE
//
// The Temporal versioning fix in dispatchActivityName must ship in the SAME
// release as the per-task activity alias change. The alias-without-marker
// state must never be deployed on its own:
//
//   - Without the versioning fix, executions started under the alias change
//     record per-task aliases (e.g. "wf.fetch") with NO version marker.
//   - If the versioning fix is then deployed, those in-flight histories have
//     no marker, so GetVersion returns DefaultVersion and the worker would
//     schedule the LEGACY name on replay, mismatching the recorded alias and
//     failing replay with a non-determinism error.
//
// Shipping both together means no execution ever runs in the
// alias-without-marker state, so this hazard cannot occur. These tests cover
// the reverse, supported direction: pre-alias histories (legacy name, no
// marker) replaying cleanly after the change.

const replayWorkflowName = "wf-replay"

// These are the literal activity type names Temporal recorded in histories
// before per-task aliases existed, derived from the activity method names via
// the default ActivitiesList registration. They are written as literals here,
// independently of the legacy* constants the production code schedules, so
// these tests verify the constants agree with the real historical names
// rather than merely agreeing with themselves.
const (
	historicalCallHTTPActivityType   = "CallHTTPActivity"
	historicalCallGRPCActivityType   = "CallGRPCActivity"
	historicalCallScriptActivityType = "CallScriptActivity"
)

// TestReplayUsesLegacyActivityTypeForOldHistories is the replay regression
// guard for the per-task activity alias change. For each activity dispatch
// path changed by this PR it replays a history that was recorded before the
// change, when the activity was scheduled under its historical fixed type
// name and no version marker existed.
//
// The replayed workflow is the current dispatch closure, which now calls
// workflow.GetVersion. Because the crafted history has no version marker,
// GetVersion must return workflow.DefaultVersion and the closure must
// schedule the historical name again, matching the recorded
// ActivityTaskScheduled event. If the closure scheduled the per-task alias
// instead, replay would fail with a non-determinism error.
//
// The histories use literal historical names, so a case fails if: the wrong
// legacy constant is used, legacyActivityName is not assigned (run paths), or
// the path schedules the new alias during replay of an old history.
func TestReplayUsesLegacyActivityTypeForOldHistories(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: replayWorkflowName}}

	cases := []struct {
		name string
		// historicalActivityType is the literal type the old history
		// recorded; the dispatch closure must reschedule exactly this.
		historicalActivityType string
		// newBuilder constructs the task builder under test with a nil
		// worker (registration is skipped; only the dispatch closure, which
		// is what we replay, is built as in production).
		newBuilder func(t *testing.T) TaskBuilder
	}{
		{
			name:                   "call-http",
			historicalActivityType: historicalCallHTTPActivityType,
			newBuilder: func(t *testing.T) TaskBuilder {
				t.Helper()
				b, err := NewCallHTTPTaskBuilder(nil, newTestHTTPTask(), "fetch", doc, testEvents, nil)
				require.NoError(t, err)
				return b
			},
		},
		{
			name:                   "call-grpc",
			historicalActivityType: historicalCallGRPCActivityType,
			newBuilder: func(t *testing.T) TaskBuilder {
				t.Helper()
				b, err := NewCallGRPCTaskBuilder(nil, newTestGRPCTask(), "fetch", doc, testEvents, nil)
				require.NoError(t, err)
				return b
			},
		},
		{
			name:                   "run-script",
			historicalActivityType: historicalCallScriptActivityType,
			newBuilder: func(t *testing.T) TaskBuilder {
				t.Helper()
				task := &model.RunTask{
					Run: model.RunTaskConfiguration{
						Await: utils.Ptr(true),
						Script: &model.Script{
							Language:   constScriptLanguagePython,
							InlineCode: utils.Ptr("print(1)"),
						},
					},
				}
				b, err := NewRunTaskBuilder(nil, task, "fetch", doc, testEvents, nil)
				require.NoError(t, err)
				require.NoError(t, b.PostLoad())
				return b
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, err := tc.newBuilder(t).Build()
			require.NoError(t, err)

			workflowName := replayWorkflowName + "-" + tc.name

			replayer := worker.NewWorkflowReplayer()
			replayer.RegisterWorkflowWithOptions(
				func(ctx workflow.Context, input any, state *utils.State) (any, error) {
					ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
						StartToCloseTimeout: time.Minute,
					})
					return fn(ctx, input, state)
				},
				workflow.RegisterOptions{Name: workflowName},
			)

			history := legacyActivityHistory(t, workflowName, tc.historicalActivityType)

			require.NoError(t, replayer.ReplayWorkflowHistory(nil, history),
				"an old history that scheduled %q must still replay after the per-task alias change",
				tc.historicalActivityType)
		})
	}
}

// legacyActivityHistory builds a minimal, marker-free history for a workflow
// that scheduled a single activity under the given historical type name and
// completed. The absence of a version marker is the crucial property: it is
// what an execution started before the alias change looks like.
func legacyActivityHistory(t *testing.T, workflowName, activityType string) *historypb.History {
	t.Helper()

	dc := converter.GetDefaultDataConverter()

	// The workflow takes (input, state); state must be non-nil because the
	// dispatch closures write the activity result into it on success.
	startedInput, err := dc.ToPayloads(map[string]any{}, utils.NewState())
	require.NoError(t, err)

	activityResult, err := dc.ToPayloads(map[string]any{"ok": true})
	require.NoError(t, err)

	taskQueue := &taskqueuepb.TaskQueue{Name: "replay-task-queue"}

	events := []*historypb.HistoryEvent{
		{
			EventId:   1,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					WorkflowType: &commonpb.WorkflowType{Name: workflowName},
					TaskQueue:    taskQueue,
					Input:        startedInput,
				},
			},
		},
		{
			EventId:   2,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{
				WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
					TaskQueue: taskQueue,
				},
			},
		},
		{
			EventId:   3,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{
				WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{},
			},
		},
		{
			EventId:   4,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{
				WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{
					ScheduledEventId: 2,
					StartedEventId:   3,
				},
			},
		},
		{
			// The activity scheduled under the LEGACY name. The replayer
			// matches the command's activity type against this; the activity
			// id matches the scheduled event id by SDK convention.
			EventId:   5,
			EventType: enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
					ActivityId:                   "5",
					ActivityType:                 &commonpb.ActivityType{Name: activityType},
					TaskQueue:                    taskQueue,
					WorkflowTaskCompletedEventId: 4,
				},
			},
		},
		{
			EventId:   6,
			EventType: enumspb.EVENT_TYPE_ACTIVITY_TASK_STARTED,
			Attributes: &historypb.HistoryEvent_ActivityTaskStartedEventAttributes{
				ActivityTaskStartedEventAttributes: &historypb.ActivityTaskStartedEventAttributes{
					ScheduledEventId: 5,
				},
			},
		},
		{
			EventId:   7,
			EventType: enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
				ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
					ScheduledEventId: 5,
					StartedEventId:   6,
					Result:           activityResult,
				},
			},
		},
		{
			EventId:   8,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{
				WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
					TaskQueue: taskQueue,
				},
			},
		},
		{
			EventId:   9,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{
				WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{},
			},
		},
		{
			EventId:   10,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
			Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{
				WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{
					ScheduledEventId: 8,
					StartedEventId:   9,
				},
			},
		},
		{
			EventId:   11,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{
				WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{
					Result:                       activityResult,
					WorkflowTaskCompletedEventId: 10,
				},
			},
		},
	}

	return &historypb.History{Events: events}
}

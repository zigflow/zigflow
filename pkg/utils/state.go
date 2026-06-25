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

package utils

import (
	"context"
	"encoding/json"
	"maps"

	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type State struct {
	CANStartFrom *string        `json:"canStartFrom,omitempty"` // Continue-as-new from here
	Context      any            `json:"context"`                // Output data exported to later tasks output
	Data         map[string]any `json:"data"`                   // Data stored along the way
	Env          map[string]any `json:"env"`                    // Available environment variables
	Input        any            `json:"input,omitempty"`        // The input given by the caller
	Output       any            `json:"output"`                 // What will be output to the caller
}

func (s *State) init() *State {
	if s.Env == nil {
		s.Env = map[string]any{}
	}
	if s.Data == nil {
		s.Data = map[string]any{}
	}

	return s
}

func (s *State) AddData(data map[string]any) *State {
	maps.Copy(s.Data, data)

	return s
}

// jsonNormalize converts a metadata map to JSON-native types (string, number,
// bool, []any, map[string]any) so it is safe to evaluate with gojq, which
// panics on any other Go type. The map is round-tripped through JSON, so each
// value is rendered the way gojq will see it. The transform is pure and
// deterministic, so it is replay-safe; on marshal/unmarshal error the original
// map is returned unchanged.
func jsonNormalize(m map[string]any) map[string]any {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return m
	}
	return out
}

func (s *State) AddActivityInfo(ctx context.Context) *State {
	info := activity.GetInfo(ctx)

	activityData := map[string]any{
		"activity_id":               info.ActivityID,
		"activity_type_name":        info.ActivityType.Name,
		"attempt":                   info.Attempt,
		"deadline":                  info.Deadline,
		"heartbeat_token":           info.HeartbeatTimeout,
		"is_local_activity":         info.IsLocalActivity,
		"priority_key":              info.Priority.PriorityKey,
		"schedule_to_close_timeout": info.ScheduleToCloseTimeout,
		"scheduled_time":            info.ScheduledTime,
		"start_to_close_timeout":    info.StartToCloseTimeout,
		"started_time":              info.StartedTime,
		"task_queue":                info.TaskQueue,
		"task_token":                string(info.TaskToken),
		"workflow_namespace":        info.Namespace,
		"workflow_execution_id":     info.WorkflowExecution.ID,
		"workflow_execution_run_id": info.WorkflowExecution.RunID,
	}

	if w := info.WorkflowType; w != nil {
		activityData["workflow_type_name"] = w.Name
	}

	s.AddData(map[string]any{
		"activity": jsonNormalize(activityData),
	})

	return s
}

func (s *State) AddWorkflowInfo(ctx workflow.Context) *State {
	info := workflow.GetInfo(ctx)

	workflowData := map[string]any{
		"attempt":                    info.Attempt,
		"binary_checksum":            info.BinaryChecksum,
		"continued_execution_run_id": info.ContinuedExecutionRunID,
		"cron_schedule":              info.CronSchedule,
		"first_run_id":               info.FirstRunID,
		"namespace":                  info.Namespace,
		"original_run_id":            info.OriginalRunID,
		"parent_workflow_namespace":  info.ParentWorkflowNamespace,
		"priority_key":               info.Priority.PriorityKey,
		"task_queue_name":            info.TaskQueueName,
		"workflow_execution_id":      info.WorkflowExecution.ID,
		"workflow_execution_run_id":  info.WorkflowExecution.RunID,
		"workflow_execution_timeout": info.WorkflowExecutionTimeout,
		"workflow_start_time":        info.WorkflowStartTime,
		"workflow_type_name":         info.WorkflowType.Name,
	}

	if r := info.RootWorkflowExecution; r != nil {
		workflowData["root_workflow_execution_id"] = r.ID
		workflowData["root_workflow_execution_run_id"] = r.RunID
	}

	if p := info.ParentWorkflowExecution; p != nil {
		workflowData["parent_workflow_execution_id"] = p.ID
		workflowData["parent_workflow_execution_run_id"] = p.RunID
	}

	s.AddData(map[string]any{
		"workflow": jsonNormalize(workflowData),
	})

	return s
}

func (s *State) ClearOutput() *State {
	s.Output = nil
	return s
}

func (s *State) Clone() *State {
	s1 := NewState()

	s1.Context = swUtils.DeepCloneValue(s.Context)
	s1.Data = swUtils.DeepClone(s.Data)
	s1.Env = swUtils.DeepClone(s.Env)
	s1.Input = swUtils.DeepCloneValue(s.Input)
	s1.Output = swUtils.DeepCloneValue(s.Output)

	return s1
}

// Returns the state as a map.
func (s *State) GetAsMap() map[string]any {
	s1 := s.Clone()

	return map[string]any{
		"$context": s1.Context,
		"$data":    s1.Data,
		"$env":     s1.Env,
		"$input":   s1.Input,
		"$output":  s1.Output,
	}
}

func NewState() *State {
	s := &State{}
	return s.init()
}

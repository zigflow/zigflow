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

package metadata

import (
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
)

type ScheduleInfo struct {
	ID           string
	WorkflowName string
	Input        []any
}

func GetScheduleInfo(workflow *model.Workflow, envvars map[string]any) (*ScheduleInfo, error) {
	// This is the workflow name we trigger - this is required
	var workflowName string
	if t, ok := workflow.Document.Metadata[MetadataScheduleWorkflowName]; ok {
		if t, ok := t.(string); ok {
			workflowName = t
		} else {
			return nil, fmt.Errorf("schedule workflow name must be a string")
		}
	}

	// Optionally, get the schedule ID - default to "zigflow_<workflow.document.name>"
	scheduleID := fmt.Sprintf("zigflow_%s", workflow.Document.Name)
	if s, ok := workflow.Document.Metadata[MetadataScheduleID]; ok {
		if sID, ok := s.(string); ok {
			// Schedule ID is set in the metadata
			scheduleID = sID
		} else {
			return nil, fmt.Errorf("schedule id must be a string")
		}
	}

	// Optionally, get any input
	var input []any
	if in, ok := workflow.Document.Metadata[MetadataScheduleInput]; ok {
		if i, ok := in.([]any); ok {
			input = i
		} else {
			return nil, fmt.Errorf("schedule input must be in array format")
		}
	}

	// Parse any envvars in the input
	state := utils.NewState()
	state.Env = envvars

	parsedInput, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		"input": input,
	}), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error interpolating input for schedules: %w", err)
	}

	return &ScheduleInfo{
		ID:           scheduleID,
		WorkflowName: workflowName,
		Input:        parsedInput.(map[string]any)["input"].([]any),
	}, nil
}

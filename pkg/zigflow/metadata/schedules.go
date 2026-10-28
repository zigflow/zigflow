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
	"slices"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog/log"
	"github.com/zigflow/zigflow/pkg/utils"
)

type ScheduleInfo struct {
	ID           string
	WorkflowName string
	Input        []any
}

func GetScheduleInfo(workflow *model.Workflow, envvars map[string]any) (*ScheduleInfo, error) {
	// The workflow type the schedule starts; only needed when there is a schedule
	var workflowType string
	if workflow.Schedule != nil {
		target, err := ScheduleTarget(workflow)
		if err != nil {
			return nil, err
		}
		if _, ok := workflow.Document.Metadata[MetadataScheduleWorkflowName]; ok {
			log.Warn().
				Str("docs", "https://zigflow.dev/docs/dsl/intro#metadata").
				Msgf("document.metadata.%s is deprecated; use %s", MetadataScheduleWorkflowName, MetadataScheduleWorkflowType)
		}
		workflowType = target
	}

	// Optionally, get the schedule ID - default to "zigflow_<document.workflowType>"
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
		WorkflowName: workflowType,
		Input:        parsedInput.(map[string]any)["input"].([]any),
	}, nil
}

// ScheduleTarget returns the workflow type a schedule starts: scheduleWorkflowType,
// else the deprecated scheduleWorkflowName, else document.workflowType when the
// document runs as that one workflow. A document whose top-level tasks are all
// do tasks registers each of them as its own workflow type instead, so it must
// name one.
func ScheduleTarget(workflow *model.Workflow) (string, error) {
	target, err := stringMetadata(workflow, MetadataScheduleWorkflowType)
	if err != nil {
		return "", err
	}
	legacy, err := stringMetadata(workflow, MetadataScheduleWorkflowName)
	if err != nil {
		return "", err
	}

	switch {
	case target != "" && legacy != "" && target != legacy:
		return "", fmt.Errorf(
			"document.metadata.%s %q and the deprecated %s %q differ: keep %s only",
			MetadataScheduleWorkflowType, target, MetadataScheduleWorkflowName, legacy, MetadataScheduleWorkflowType,
		)
	case target != "":
		return target, nil
	case legacy != "":
		return legacy, nil
	case runsAsOneWorkflow(workflow):
		return workflow.Document.Name, nil
	}
	return "", fmt.Errorf(
		"document.metadata.%s is required for a schedule when every top-level task is a do task",
		MetadataScheduleWorkflowType,
	)
}

func stringMetadata(workflow *model.Workflow, key string) (string, error) {
	value, ok := workflow.Document.Metadata[key]
	if !ok {
		return "", nil
	}
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("document.metadata.%s must be a string", key)
	}
	return s, nil
}

// runsAsOneWorkflow mirrors DoTaskBuilder.Build: a top-level task list that is
// not only do tasks runs as one workflow of type document.workflowType.
func runsAsOneWorkflow(workflow *model.Workflow) bool {
	if workflow.Do == nil {
		return false
	}
	return slices.ContainsFunc(*workflow.Do, func(task *model.TaskItem) bool {
		return task.AsDoTask() == nil
	})
}

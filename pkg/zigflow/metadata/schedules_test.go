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

package metadata_test

import (
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
)

const (
	docType   = "nightly"
	otherType = "cleanup"
)

func scheduled(meta map[string]any, tasks ...*model.TaskItem) *model.Workflow {
	list := model.TaskList(tasks)
	return &model.Workflow{
		Document: model.Document{Name: docType, Namespace: "zigflow", Metadata: meta},
		Do:       &list,
		Schedule: &model.Schedule{Cron: "0 0 * * *"},
	}
}

func setTask(key string) *model.TaskItem {
	return &model.TaskItem{Key: key, Task: &model.SetTask{
		Set: model.NewObjectOrRuntimeExpr(map[string]any{"a": 1}),
	}}
}

func doTask(key string) *model.TaskItem {
	inner := model.TaskList{setTask("step")}
	return &model.TaskItem{Key: key, Task: &model.DoTask{Do: &inner}}
}

func TestScheduleTarget(t *testing.T) {
	tests := []struct {
		name     string
		workflow *model.Workflow
		want     string
		wantErr  string
	}{
		{
			name:     "scheduleWorkflowType wins",
			workflow: scheduled(map[string]any{metadata.MetadataScheduleWorkflowType: otherType}, setTask("a")),
			want:     otherType,
		},
		{
			name:     "deprecated scheduleWorkflowName still works",
			workflow: scheduled(map[string]any{metadata.MetadataScheduleWorkflowName: otherType}, setTask("a")),
			want:     otherType,
		},
		{
			name: "both keys with the same value",
			workflow: scheduled(map[string]any{
				metadata.MetadataScheduleWorkflowType: otherType,
				metadata.MetadataScheduleWorkflowName: otherType,
			}, setTask("a")),
			want: otherType,
		},
		{
			name: "both keys with different values",
			workflow: scheduled(map[string]any{
				metadata.MetadataScheduleWorkflowType: otherType,
				metadata.MetadataScheduleWorkflowName: docType,
			}, setTask("a")),
			wantErr: "differ",
		},
		{
			name:     "no key defaults to document.workflowType for one workflow",
			workflow: scheduled(nil, setTask("a"), doTask("b")),
			want:     docType,
		},
		{
			name:     "no key is an error when every top-level task is a do task",
			workflow: scheduled(nil, doTask(docType), doTask(otherType)),
			wantErr:  "required for a schedule",
		},
		{
			name:     "a do-only file names one of its workflows",
			workflow: scheduled(map[string]any{metadata.MetadataScheduleWorkflowType: otherType}, doTask(docType), doTask(otherType)),
			want:     otherType,
		},
		{
			name:     "a non-string value is an error",
			workflow: scheduled(map[string]any{metadata.MetadataScheduleWorkflowType: 42}, setTask("a")),
			wantErr:  "must be a string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := metadata.ScheduleTarget(test.workflow)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGetScheduleInfo_TargetOnlyMattersWithASchedule(t *testing.T) {
	workflow := scheduled(map[string]any{
		metadata.MetadataScheduleWorkflowType: otherType,
		metadata.MetadataScheduleWorkflowName: docType,
	}, doTask(docType))
	workflow.Schedule = nil

	info, err := metadata.GetScheduleInfo(workflow, nil)
	assert.NoError(t, err, "a document without a schedule is not checked for a schedule target")
	assert.Empty(t, info.WorkflowName)
}

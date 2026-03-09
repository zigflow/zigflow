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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestListenTaskBuilderListEvents(t *testing.T) {
	newEvent := func(id string, t ListenTaskType) *model.EventFilter {
		return &model.EventFilter{
			With: &model.EventProperties{
				ID:   id,
				Type: string(t),
			},
		}
	}

	tests := []struct {
		name      string
		task      *model.ListenTask
		expectAll bool
		expectErr string
	}{
		{
			name: "all events respected",
			task: &model.ListenTask{
				Listen: model.ListenTaskConfiguration{
					To: &model.EventConsumptionStrategy{
						All: []*model.EventFilter{
							newEvent("sig-1", ListenTaskTypeSignal),
						},
					},
				},
			},
			expectAll: true,
		},
		{
			name: "any events respected",
			task: &model.ListenTask{
				Listen: model.ListenTaskConfiguration{
					To: &model.EventConsumptionStrategy{
						Any: []*model.EventFilter{
							newEvent("sig-1", ListenTaskTypeSignal),
						},
					},
				},
			},
		},
		{
			name: "single event treated as all",
			task: &model.ListenTask{
				Listen: model.ListenTaskConfiguration{
					To: &model.EventConsumptionStrategy{
						One: newEvent("sig-one", ListenTaskTypeSignal),
					},
				},
			},
			expectAll: true,
		},
		{
			name: "missing events returns error",
			task: &model.ListenTask{
				Listen: model.ListenTaskConfiguration{
					To: &model.EventConsumptionStrategy{},
				},
			},
			expectErr: "no listen task configured",
		},
		{
			name: "invalid event returns error",
			task: &model.ListenTask{
				Listen: model.ListenTaskConfiguration{
					To: &model.EventConsumptionStrategy{
						All: []*model.EventFilter{
							{
								With: &model.EventProperties{
									Type: string(ListenTaskTypeSignal),
								},
							},
						},
					},
				},
			},
			expectErr: "listen task id is not set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := &ListenTaskBuilder{
				builder: builder[*model.ListenTask]{
					name: "listen",
					task: tc.task,
				},
			}

			events, isAll, err := builder.listEvents()
			if tc.expectErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, events)
			assert.Equal(t, tc.expectAll, isAll)
		})
	}
}

func TestListenTaskBuilderProcessReply(t *testing.T) {
	builder := &ListenTaskBuilder{
		builder: builder[*model.ListenTask]{
			name: "listen",
			task: &model.ListenTask{},
		},
	}

	event := &model.EventFilter{
		With: &model.EventProperties{
			ID:   "evt",
			Type: string(ListenTaskTypeSignal),
			Additional: map[string]any{
				"data": map[string]any{
					"result": "${ $data.message }",
				},
			},
		},
	}

	state := utils.NewState()
	state.AddData(map[string]any{
		"message": "hello",
	})

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return builder.processReply(ctx, event, state)
	}, workflow.RegisterOptions{Name: "process-reply"})

	env.ExecuteWorkflow("process-reply")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, map[string]any{"result": "hello"}, result)
}

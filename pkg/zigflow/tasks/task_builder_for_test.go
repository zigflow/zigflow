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

func TestForTaskBuilderAddIterationResult(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
		response any
	}{
		{
			name:     "adds string response to state data",
			taskName: "my-task",
			response: "some-result",
		},
		{
			name:     "adds map response to state data",
			taskName: "map-task",
			response: map[string]any{"key": "value"},
		},
		{
			name:     "adds nil response to state data",
			taskName: "nil-task",
			response: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := utils.NewState()

			b := &ForTaskBuilder{
				builder: builder[*model.ForTask]{
					doc:          testWorkflow,
					eventEmitter: testEvents,
					name:         tc.taskName,
					task: &model.ForTask{
						For: model.ForTaskConfiguration{In: "${ .data.items }"},
						Do:  &model.TaskList{},
					},
				},
			}

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			workflowName := "add-iteration-" + tc.name
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
				b.addIterationResult(ctx, state, tc.response)
				return nil
			}, workflow.RegisterOptions{Name: workflowName})

			env.ExecuteWorkflow(workflowName)
			assert.NoError(t, env.GetWorkflowError())

			assert.Equal(t, tc.response, state.Data[tc.taskName])
		})
	}
}

func TestForTaskBuilderCheckWhile(t *testing.T) {
	tests := []struct {
		name        string
		while       string
		stateData   map[string]any
		expect      bool
		expectError bool
	}{
		{
			name:   "empty while defaults to true",
			expect: true,
		},
		{
			name:  "boolean true expression",
			while: "${ $data.flag }",
			stateData: map[string]any{
				"flag": true,
			},
			expect: true,
		},
		{
			name:  "boolean false expression",
			while: "${ $data.flag }",
			stateData: map[string]any{
				"flag": false,
			},
			expect: false,
		},
		{
			name:  "non boolean resolves to false",
			while: "${ $data.text }",
			stateData: map[string]any{
				"text": "not-bool",
			},
			expect: false,
		},
		{
			name:        "invalid expression returns error",
			while:       "${ $data. }",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := utils.NewState()
			state.AddData(tc.stateData)

			builder := &ForTaskBuilder{
				builder: builder[*model.ForTask]{
					eventEmitter: testEvents,
					name:         "for-task",
					task: &model.ForTask{
						For:   model.ForTaskConfiguration{In: "${ .data.items }"},
						While: tc.while,
						Do:    &model.TaskList{},
					},
				},
			}

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			workflowName := "check-" + tc.name
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (bool, error) {
				return builder.checkWhile(ctx, state)
			}, workflow.RegisterOptions{Name: workflowName})

			env.ExecuteWorkflow(workflowName)

			err := env.GetWorkflowError()
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			var res bool
			assert.NoError(t, env.GetWorkflowResult(&res))
			assert.Equal(t, tc.expect, res)
		})
	}
}

func TestForTaskBuilderIterator(t *testing.T) {
	state := utils.NewState()
	state.Input = map[string]any{
		"request_id": "abc",
	}

	builder := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         "iterate",
			task: &model.ForTask{
				For: model.ForTaskConfiguration{
					Each: "value",
					At:   "idx",
					In:   "${ .data.items }",
				},
				Do: &model.TaskList{
					&model.TaskItem{Key: "first", Task: &model.DoTask{}},
				},
			},
		},
		childWorkflowName: utils.GenerateChildWorkflowName("for", "iterate"),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		return map[string]any{
			"child_value": st.Data["value"],
		}, nil
	}, workflow.RegisterOptions{Name: builder.childWorkflowName})

	state.AddData(map[string]any{
		"items": []any{"item-value"},
	})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return builder.iterator(ctx, 0, "item-value", state)
	}, workflow.RegisterOptions{Name: "iterator-test"})

	env.ExecuteWorkflow("iterator-test")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, map[string]any{"child_value": "item-value"}, result)
	assert.Equal(t, "item-value", state.Data["value"])
	assert.Equal(t, 0, state.Data["idx"])
}

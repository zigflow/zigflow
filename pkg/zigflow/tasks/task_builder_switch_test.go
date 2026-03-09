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

func TestSwitchTaskBuilderBuildRejectsMultipleDefaults(t *testing.T) {
	task := &model.SwitchTask{
		Switch: []model.SwitchItem{
			{
				"defaultOne": {
					Then: &model.FlowDirective{Value: "child-a"},
				},
			},
			{
				"defaultTwo": {
					Then: &model.FlowDirective{Value: "child-b"},
				},
			},
		},
	}

	builder, err := NewSwitchTaskBuilder(nil, task, "switch-task", nil, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.Nil(t, fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple switch statements without when")
}

func TestSwitchTaskBuilderExecutesMatchingCase(t *testing.T) {
	childWorkflow := "child-switch"
	childRan := false
	task := &model.SwitchTask{
		Switch: []model.SwitchItem{
			{
				"match": {
					When: model.NewRuntimeExpression("${ $data.run }"),
					Then: &model.FlowDirective{Value: childWorkflow},
				},
			},
		},
	}

	builder, err := NewSwitchTaskBuilder(nil, task, "switch-task", nil, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	state.AddData(map[string]any{
		"run": true,
	})

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		childRan = true
		return nil, nil
	}, workflow.RegisterOptions{Name: childWorkflow})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "switch-test"})

	env.ExecuteWorkflow("switch-test")
	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, childRan)
}

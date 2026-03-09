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

func TestSetTaskBuilderBuild(t *testing.T) {
	task := &model.SetTask{
		Set: map[string]any{
			"result": map[string]any{
				"value": "${ $env.VALUE }",
			},
		},
	}

	builder, err := NewSetTaskBuilder(nil, task, "set-task", nil, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	state.Env["VALUE"] = "ok"

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "set-task"})

	env.ExecuteWorkflow("set-task")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))

	expected := map[string]any{
		"result": map[string]any{
			"value": "ok",
		},
	}

	assert.Equal(t, expected, result)
	assert.Equal(t, expected["result"], state.Data["result"])
}

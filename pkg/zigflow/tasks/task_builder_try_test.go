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
	"errors"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestTryTaskBuilderGetTasks(t *testing.T) {
	task := &model.TryTask{
		Try: &model.TaskList{
			&model.TaskItem{Key: "task", Task: &model.SetTask{}},
		},
		Catch: &model.TryTaskCatch{
			Do: &model.TaskList{
				&model.TaskItem{Key: "catch", Task: &model.SetTask{}},
			},
		},
	}

	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			task: task,
		},
	}

	got := builder.getTasks()
	assert.Equal(t, task.Try, got["try"])
	assert.Equal(t, task.Catch.Do, got["catch"])
}

func TestTryTaskBuilderExecRunsCatchOnError(t *testing.T) {
	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					Do: &model.TaskList{},
				},
			},
		},
		tryChildWorkflowName:   "try-child",
		catchChildWorkflowName: "catch-child",
	}

	fn, err := builder.exec()
	assert.NoError(t, err)

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		return nil, errors.New("boom")
	}, workflow.RegisterOptions{Name: builder.tryChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		return map[string]any{
			"handled": true,
		}, nil
	}, workflow.RegisterOptions{Name: builder.catchChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "try-exec"})

	env.ExecuteWorkflow("try-exec")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, map[string]any{"handled": true}, result)
}

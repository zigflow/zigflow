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
	"context"
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestCallActivityTaskBuilderExecute(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	const activityName = "dslTestActivity"
	env.RegisterActivityWithOptions(func(ctx context.Context, value string) (string, error) {
		return value + "-processed", nil
	}, activity.RegisterOptions{Name: activityName})

	task := &model.CallFunction{
		Call: customCallFunctionActivity,
		With: map[string]any{
			"name":      activityName,
			"arguments": []any{"${ $input.message }"},
			"taskQueue": "some-task-queue",
		},
	}

	b, err := NewCallActivityTaskBuilder(nil, task, "callActivity", nil, testEvents)
	assert.NoError(t, err)

	fn, err := b.Build()
	assert.NoError(t, err)

	workflowFunc := func(ctx workflow.Context) (string, error) {
		state := utils.NewState().AddWorkflowInfo(ctx)
		input := map[string]any{"message": "ping"}
		state.Input = input
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		result, err := fn(ctx, input, state)
		if err != nil {
			return "", err
		}
		if result == nil {
			return "", nil
		}
		return result.(string), nil
	}

	env.ExecuteWorkflow(workflowFunc)

	var got string
	assert.NoError(t, env.GetWorkflowError())
	assert.NoError(t, env.GetWorkflowResult(&got))
	assert.Equal(t, "ping-processed", got)
}

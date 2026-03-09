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
	"go.temporal.io/sdk/temporal"
)

// mockTask implements model.Task for testing purposes
type mockTask struct {
	base *model.TaskBase
}

func (m *mockTask) GetBase() *model.TaskBase {
	return m.base
}

func TestShouldRun(t *testing.T) {
	type testCase struct {
		name          string
		ifExpr        string
		expectedRun   bool
		expectError   bool
		errorContains string
	}

	tests := []testCase{
		{
			name:        "No If statement returns true",
			ifExpr:      "",
			expectedRun: true,
		},
		{
			name:        "Boolean true returns true",
			ifExpr:      "true",
			expectedRun: true,
		},
		{
			name:        "Boolean false returns false",
			ifExpr:      "false",
			expectedRun: false,
		},
		{
			name:        "String TRUE returns true",
			ifExpr:      "TRUE",
			expectedRun: true,
		},
		{
			name:        "String '1' returns true",
			ifExpr:      "1",
			expectedRun: true,
		},
		{
			name:        "String FALSE returns false",
			ifExpr:      "FALSE",
			expectedRun: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ifField *model.RuntimeExpression
			if tc.ifExpr != "" {
				ifField = &model.RuntimeExpression{Value: tc.ifExpr}
			}

			task := &mockTask{base: &model.TaskBase{If: ifField}}
			b := builder[*mockTask]{
				task: task,
			}

			result, err := b.ShouldRun(&utils.State{})

			if tc.expectError {
				assert.Error(t, err)
				var appErr *temporal.ApplicationError
				assert.ErrorAs(t, err, &appErr)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedRun, result)
		})
	}
}

func TestNewTaskBuilderFactory(t *testing.T) {
	doc := &model.Workflow{}

	tests := []struct {
		name         string
		task         model.Task
		expectedType any
		expectErr    bool
	}{
		{
			name:         "call http",
			task:         &model.CallHTTP{},
			expectedType: &CallHTTPTaskBuilder{},
		},
		{
			name:         "do task",
			task:         &model.DoTask{},
			expectedType: &DoTaskBuilder{},
		},
		{
			name:         "for task",
			task:         &model.ForTask{},
			expectedType: &ForTaskBuilder{},
		},
		{
			name:         "fork task",
			task:         &model.ForkTask{},
			expectedType: &ForkTaskBuilder{},
		},
		{
			name:         "listen task",
			task:         &model.ListenTask{},
			expectedType: &ListenTaskBuilder{},
		},
		{
			name:         "raise task",
			task:         &model.RaiseTask{},
			expectedType: &RaiseTaskBuilder{},
		},
		{
			name:         "run task",
			task:         &model.RunTask{},
			expectedType: &RunTaskBuilder{},
		},
		{
			name:         "set task",
			task:         &model.SetTask{},
			expectedType: &SetTaskBuilder{},
		},
		{
			name:         "switch task",
			task:         &model.SwitchTask{},
			expectedType: &SwitchTaskBuilder{},
		},
		{
			name:         "try task",
			task:         &model.TryTask{},
			expectedType: &TryTaskBuilder{},
		},
		{
			name:         "wait task",
			task:         &model.WaitTask{},
			expectedType: &WaitTaskBuilder{},
		},
		{
			name:      "unsupported task type",
			task:      &mockTask{base: &model.TaskBase{}},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder, err := NewTaskBuilder(tc.name, tc.task, nil, doc, testEvents)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, builder)
				return
			}

			assert.NoError(t, err)
			assert.IsType(t, tc.expectedType, builder)
			assert.Equal(t, tc.name, builder.GetTaskName())
		})
	}
}

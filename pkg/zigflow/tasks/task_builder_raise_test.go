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
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestRaiseTaskBuilderBuild(t *testing.T) {
	tests := []struct {
		name      string
		errorDef  *model.Error
		expectErr func(error)
	}{
		{
			name: "standard workflow error mapping",
			errorDef: &model.Error{
				Type:   model.NewUriTemplate(model.ErrorTypeValidation),
				Status: 400,
				Title:  model.NewStringOrRuntimeExpr("Validation error"),
				Detail: model.NewStringOrRuntimeExpr("Invalid payload"),
			},
			expectErr: func(err error) {
				var appErr *temporal.ApplicationError
				assert.ErrorAs(t, err, &appErr)
				if assert.NotNil(t, appErr) {
					assert.Contains(t, appErr.Error(), "Validation error")
				}
			},
		},
		{
			name: "temporal non retryable error mapping",
			errorDef: &model.Error{
				Type:   model.NewUriTemplate("custom"),
				Status: 500,
				Title:  model.NewStringOrRuntimeExpr(temporaErrlNonRetryable),
				Detail: model.NewStringOrRuntimeExpr("non retryable"),
			},
			expectErr: func(err error) {
				var appErr *temporal.ApplicationError
				assert.ErrorAs(t, err, &appErr)
				if assert.NotNil(t, appErr) {
					assert.True(t, appErr.NonRetryable())
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := &RaiseTaskBuilder{
				builder: builder[*model.RaiseTask]{
					name: "raise-task",
					task: &model.RaiseTask{
						Raise: model.RaiseTaskConfiguration{
							Error: model.RaiseTaskError{
								Definition: tc.errorDef,
							},
						},
					},
				},
			}

			fn, err := builder.Build()
			assert.NoError(t, err)

			state := utils.NewState()

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				return fn(ctx, nil, state)
			}, workflow.RegisterOptions{Name: "raise-" + tc.name})

			env.ExecuteWorkflow("raise-" + tc.name)
			err = env.GetWorkflowError()
			assert.Error(t, err)
			tc.expectErr(err)
		})
	}
}

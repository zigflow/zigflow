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
	"sync"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				name: "raise-task",
				task: &model.RaiseTask{
					Raise: model.RaiseTaskConfiguration{
						Error: model.RaiseTaskError{
							Definition: tc.errorDef,
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

func raiseInDocument(errorDef *model.Error) *RaiseTaskBuilder {
	task := &model.RaiseTask{Raise: model.RaiseTaskConfiguration{Error: model.RaiseTaskError{Definition: errorDef}}}
	list := model.TaskList{
		&model.TaskItem{Key: testConstTaskBefore, Task: &model.SetTask{}},
		&model.TaskItem{Key: "failing", Task: task},
	}
	return &RaiseTaskBuilder{builder[*model.RaiseTask]{doc: &model.Workflow{Do: &list}, name: "failing", task: task}}
}

func runRaise(t *testing.T, builder *RaiseTaskBuilder) error {
	t.Helper()
	fn, err := builder.Build()
	require.NoError(t, err)
	return runRaiseFn(fn)
}

func runRaiseFn(fn TemporalWorkflowFunc) error {
	return runRaiseFnWithState(fn, utils.NewState())
}

func runRaiseFnWithState(fn TemporalWorkflowFunc, state *utils.State) error {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "raise"})
	env.ExecuteWorkflow("raise")
	return env.GetWorkflowError()
}

// The specification asks a runtime to set instance to a JSON Pointer to the
// component the error comes from: the raise task itself.
func TestRaiseErrorInstanceIsTheTaskPointer(t *testing.T) {
	for name, errorType := range map[string]string{
		"standard type": model.ErrorTypeValidation,
		"custom type":   "https://example.com/errors/custom",
	} {
		t.Run(name, func(t *testing.T) {
			builder := raiseInDocument(&model.Error{
				Type:  model.NewUriTemplate(errorType),
				Title: model.NewStringOrRuntimeExpr("Boom"),
			})
			assert.ErrorContains(t, runRaise(t, builder), "Origin: '/do/1/failing'")
		})
	}
}

func TestRaiseKeepsTheInstanceTheTaskSets(t *testing.T) {
	for name, instance := range map[string]any{
		"pointer":            "/my/own/pointer",
		"runtime expression": model.RuntimeExpression{Value: `${ "/my/own/pointer" }`},
		// SDK v4 decodes any value that is not a JSON Pointer as a runtime
		// expression, and the schema does not enforce the pointer format.
		"plain string decoded as an expression": model.RuntimeExpression{Value: "/my/own/pointer"},
	} {
		t.Run(name, func(t *testing.T) {
			builder := raiseInDocument(&model.Error{
				Type:     model.NewUriTemplate("https://example.com/errors/custom"),
				Title:    model.NewStringOrRuntimeExpr("Boom"),
				Instance: &model.JsonPointerOrRuntimeExpression{Value: instance},
			})
			assert.ErrorContains(t, runRaise(t, builder), "Origin: '/my/own/pointer'")
		})
	}
}

func TestRaiseInstanceExpressionReadsTheWorkflowState(t *testing.T) {
	builder := raiseInDocument(&model.Error{
		Type:     model.NewUriTemplate("https://example.com/errors/custom"),
		Title:    model.NewStringOrRuntimeExpr("Boom"),
		Instance: &model.JsonPointerOrRuntimeExpression{Value: model.RuntimeExpression{Value: `${ $input.where }`}},
	})
	fn, err := builder.Build()
	require.NoError(t, err)

	state := utils.NewState()
	state.Input = map[string]any{"where": "/from/input"}
	assert.ErrorContains(t, runRaiseFnWithState(fn, state), "Origin: '/from/input'")
}

func TestRaiseInstanceExpressionMustBeAString(t *testing.T) {
	for name, expression := range map[string]string{
		"number": `${ 1 }`,
		"object": `${ {"at": "/somewhere"} }`,
	} {
		t.Run(name, func(t *testing.T) {
			builder := raiseInDocument(&model.Error{
				Type:     model.NewUriTemplate("https://example.com/errors/custom"),
				Title:    model.NewStringOrRuntimeExpr("Boom"),
				Instance: &model.JsonPointerOrRuntimeExpression{Value: model.RuntimeExpression{Value: expression}},
			})
			assert.ErrorContains(t, runRaise(t, builder), "error instance must evaluate to a string")
		})
	}
}

// Every run of a workflow shares its definition, so raising an error must not
// write the evaluated detail, title or instance back into it.
func TestRaiseDoesNotChangeItsDefinition(t *testing.T) {
	detail := model.NewStringOrRuntimeExpr(`${ "evaluated" }`)
	title := model.NewStringOrRuntimeExpr(`${ "Boom" }`)
	builder := raiseInDocument(&model.Error{
		Type:   model.NewUriTemplate("https://example.com/errors/custom"),
		Title:  title,
		Detail: detail,
	})

	require.Error(t, runRaise(t, builder))

	definition := builder.task.Raise.Error.Definition
	assert.Same(t, detail, definition.Detail)
	assert.Same(t, title, definition.Title)
	assert.Nil(t, definition.Instance)
}

// Runs of a workflow share its definition, and a worker runs them
// concurrently: raising must not write to it (`go test -race` catches it).
func TestRaiseConcurrentRunsDoNotShareState(t *testing.T) {
	builder := raiseInDocument(&model.Error{
		Type:   model.NewUriTemplate("https://example.com/errors/custom"),
		Title:  model.NewStringOrRuntimeExpr(`${ "Boom" }`),
		Detail: model.NewStringOrRuntimeExpr(`${ "detail" }`),
	})

	fn, err := builder.Build()
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			assert.ErrorContains(t, runRaiseFn(fn), "Origin: '/do/1/failing'")
		})
	}
	wg.Wait()
}

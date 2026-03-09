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
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestRunTaskBuilderBuildSetsAwaitDefault(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Workflow: &model.RunWorkflow{
				Namespace: "default",
				Name:      "child-runner",
				Version:   "1.0.0",
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, fn)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		return nil, nil
	}, workflow.RegisterOptions{Name: task.Run.Workflow.Name})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, map[string]any{}, utils.NewState())
	}, workflow.RegisterOptions{Name: "run-default-await"})

	env.ExecuteWorkflow("run-default-await")
	assert.NoError(t, env.GetWorkflowError())
	assert.NotNil(t, task.Run.Await)
	assert.True(t, *task.Run.Await)
}

func TestRunTaskBuilderRunWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		await         *bool
		expectNilResp bool
	}{
		{
			name:          "await child workflow result",
			await:         utils.Ptr(true),
			expectNilResp: false,
		},
		{
			name:          "skip await returns nil response",
			await:         utils.Ptr(false),
			expectNilResp: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := &model.RunTask{
				Run: model.RunTaskConfiguration{
					Await: tc.await,
					Workflow: &model.RunWorkflow{
						Namespace: "default",
						Name:      "child-runner",
						Version:   "1.0.0",
					},
				},
			}

			builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents)
			assert.NoError(t, err)

			fn, err := builder.Build()
			assert.NoError(t, err)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				return map[string]any{
					"child": "done",
				}, nil
			}, workflow.RegisterOptions{Name: task.Run.Workflow.Name})

			state := utils.NewState()

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				return fn(ctx, map[string]any{"request": "data"}, state)
			}, workflow.RegisterOptions{Name: "run-" + tc.name})

			env.ExecuteWorkflow("run-" + tc.name)
			assert.NoError(t, env.GetWorkflowError())

			var result any
			err = env.GetWorkflowResult(&result)

			if tc.expectNilResp {
				assert.EqualError(t, err, "no data available")
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, map[string]any{"child": "done"}, result)
			}

			val, ok := state.Data["run-task"]
			assert.True(t, ok)
			if tc.expectNilResp {
				assert.Nil(t, val)
			} else {
				assert.Equal(t, map[string]any{"child": "done"}, val)
			}
		})
	}
}

func TestRunTaskBuilderRunScriptValidation(t *testing.T) {
	t.Parallel()

	inline := "print('noop')"
	tests := []struct {
		name      string
		task      *model.RunTask
		assertErr string
	}{
		{
			name: "invalid language",
			task: &model.RunTask{
				Run: model.RunTaskConfiguration{
					Script: &model.Script{
						Language:   "golang",
						InlineCode: utils.Ptr(inline),
					},
				},
			},
			assertErr: "unknown script language 'golang' for task: script-task",
		},
		{
			name: "missing inline code",
			task: &model.RunTask{
				Run: model.RunTaskConfiguration{
					Script: &model.Script{
						Language: "python",
					},
				},
			},
			assertErr: "run script has no code defined: script-task",
		},
		{
			name: "await disabled",
			task: &model.RunTask{
				Run: model.RunTaskConfiguration{
					Await: utils.Ptr(false),
					Script: &model.Script{
						Language:   "python",
						InlineCode: utils.Ptr(inline),
					},
				},
			},
			assertErr: "run scripts must be run with await: script-task",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder, err := NewRunTaskBuilder(nil, tc.task, "script-task", nil, testEvents)
			assert.NoError(t, err)

			_, err = builder.Build()
			assert.EqualError(t, err, tc.assertErr)
		})
	}
}

func TestRunTaskBuilderRunScriptExecutesActivity(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	runActivities := &activities.Run{}

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Script: &model.Script{
				Language:   "python",
				InlineCode: utils.Ptr("print('hello')"),
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "script-task", nil, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	env.OnActivity(
		runActivities.CallScriptActivity,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return("script-success", nil).Once()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, map[string]any{"request": "data"}, state)
	}, workflow.RegisterOptions{Name: "script-run"})

	env.ExecuteWorkflow("script-run")
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "script-success", result)

	assert.Equal(t, "script-success", state.Data["script-task"])
}

func TestRunTaskBuilderRunShellExecutesActivity(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	runActivities := &activities.Run{}

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Shell: &model.Shell{
				Command: "echo",
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "shell-task", nil, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	env.OnActivity(
		runActivities.CallShellActivity,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return("shell-success", nil).Once()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, map[string]any{"request": "data"}, state)
	}, workflow.RegisterOptions{Name: "shell-run"})

	env.ExecuteWorkflow("shell-run")
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "shell-success", result)

	assert.Equal(t, "shell-success", state.Data["shell-task"])
}

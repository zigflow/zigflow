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
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
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
			testConstHandledKey: true,
		}, nil
	}, workflow.RegisterOptions{Name: builder.catchChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "try-exec"})

	env.ExecuteWorkflow("try-exec")
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, map[string]any{testConstHandledKey: true}, result)
}

// TestTryTaskBuilderExecPropagatesEndFromTryChild proves that a
// `then: end` directive inside the try child workflow is NOT treated
// as a catchable failure. The carried output must survive the boundary
// and exec must surface flow.ErrEnd to the do-task pipeline so the
// overall workflow ends cleanly, not run the catch handler.
func TestTryTaskBuilderExecPropagatesEndFromTryChild(t *testing.T) {
	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task-end",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					Do: &model.TaskList{},
				},
			},
		},
		tryChildWorkflowName:   "try-child-end",
		catchChildWorkflowName: "catch-child-end",
	}

	fn, err := builder.exec()
	require.NoError(t, err)

	state := utils.NewState()
	childOutput := map[string]any{testConstValue: "end-time-output"}
	catchRan := false

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		return nil, flow.NewEndApplicationError(childOutput)
	}, workflow.RegisterOptions{Name: builder.tryChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		catchRan = true
		return map[string]any{testConstHandledKey: true}, nil
	}, workflow.RegisterOptions{Name: builder.catchChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "try-exec-end"})

	env.ExecuteWorkflow("try-exec-end")

	// The try task surfaces flow.ErrEnd through the Temporal envelope.
	wErr := env.GetWorkflowError()
	require.Error(t, wErr)
	assert.Contains(t, wErr.Error(), flow.ErrEnd.Error())
	assert.False(t, catchRan, "catch handler must not run when the try child workflow signalled end")
}

// TestTryTaskBuilderExecPropagatesEndFromCatchChild is the symmetric
// case: when the try child fails for a real reason and the catch
// handler itself emits `then: end`, that end must propagate as
// flow.ErrEnd rather than being wrapped as a generic catch-workflow
// failure.
func TestTryTaskBuilderExecPropagatesEndFromCatchChild(t *testing.T) {
	builder := &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			name: "try-task-catch-end",
			task: &model.TryTask{
				Try: &model.TaskList{},
				Catch: &model.TryTaskCatch{
					Do: &model.TaskList{},
				},
			},
		},
		tryChildWorkflowName:   "try-child-real-fail",
		catchChildWorkflowName: "catch-child-end",
	}

	fn, err := builder.exec()
	require.NoError(t, err)

	state := utils.NewState()
	catchOutput := map[string]any{testConstValue: "catch-end-output"}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		return nil, errors.New("boom from try")
	}, workflow.RegisterOptions{Name: builder.tryChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (map[string]any, error) {
		return nil, flow.NewEndApplicationError(catchOutput)
	}, workflow.RegisterOptions{Name: builder.catchChildWorkflowName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "try-exec-catch-end"})

	env.ExecuteWorkflow("try-exec-catch-end")

	wErr := env.GetWorkflowError()
	require.Error(t, wErr)
	assert.Contains(t, wErr.Error(), flow.ErrEnd.Error())
	assert.NotContains(t, wErr.Error(), "error calling catcg workflow",
		"catch-emitted end must not be wrapped as a catch-workflow failure")
}

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

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// These tests prove DoTaskOpts.DisableRegisterWorkflow and
// DoTaskOpts.InlineExecution are independent:
//   - InlineExecution alone selects direct flow.ErrEnd propagation.
//   - DisableRegisterWorkflow alone selects whether the function is registered.
// Neither implies the other.

// TestWorkflowExecutorInlineExecutionReturnsDirectErrEnd covers the try/for
// body configuration: with InlineExecution set (and registration disabled), a
// task emitting `then: end` makes the executor return flow.ErrEnd directly with
// the effective state.Output preserved, rather than encoding it or completing
// cleanly at a boundary.
func TestWorkflowExecutorInlineExecutionReturnsDirectErrEnd(t *testing.T) {
	builder := newTestDoTaskBuilder("inline-end", DoTaskOpts{
		DisableRegisterWorkflow: true,
		InlineExecution:         true,
	})

	expectedOutput := map[string]any{testConstValue: "captured"}
	runOrder := make([]string, 0, 2)
	taskOne := workflowFunc{
		TaskBuilder: newFakeTaskBuilder(testConstTaskOne, &model.TaskBase{}),
		Name:        testConstTaskOne,
		Func: func(_ workflow.Context, _ any, _ *utils.State) (any, error) {
			runOrder = append(runOrder, testConstTaskOne)
			return expectedOutput, nil
		},
	}
	taskEnd := newSwitchWorkflowFunc(flow.ErrEnd, &runOrder)
	taskAfter := newSimpleWorkflowFunc("task-after", &model.TaskBase{}, &runOrder)

	wf := builder.workflowExecutor([]workflowFunc{taskOne, taskEnd, taskAfter})

	output, execErr := runInlineWorkflowFunc(t, builder.GetTaskName(), wf, nil, utils.NewState())

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd), "inline execution must return flow.ErrEnd directly")
	assert.Equal(t, expectedOutput, output, "effective state.Output must be preserved")
	// The task after the end directive must not run.
	assert.Equal(t, []string{testConstTaskOne, testConstTaskSwitch}, runOrder)
}

// TestWorkflowExecutorInlineExecutionAtRootStillReturnsDirectErrEnd proves the
// inline branch is chosen by InlineExecution regardless of whether the executor
// happens to be running at the root (ParentWorkflowExecution == nil). Without
// InlineExecution a root executor would swallow ErrEnd into a clean completion;
// with it, the directive is surfaced for the enclosing builder.
func TestWorkflowExecutorInlineExecutionAtRootStillReturnsDirectErrEnd(t *testing.T) {
	builder := newTestDoTaskBuilder("inline-end-root", DoTaskOpts{
		DisableRegisterWorkflow: false,
		InlineExecution:         true,
	})

	runOrder := make([]string, 0, 1)
	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrEnd, &runOrder),
		newSimpleWorkflowFunc("task-after", &model.TaskBase{}, &runOrder),
	}

	wf := builder.workflowExecutor(tasks)

	_, execErr := runInlineWorkflowFunc(t, builder.GetTaskName(), wf, nil, utils.NewState())

	require.Error(t, execErr)
	assert.True(t, errors.Is(execErr, flow.ErrEnd),
		"InlineExecution must select direct flow.ErrEnd even at the root boundary")
	assert.Equal(t, []string{testConstTaskSwitch}, runOrder)
}

// TestWorkflowExecutorDisableRegisterWithoutInlineKeepsBoundaryHandling is the
// key decoupling test: with registration disabled but InlineExecution unset, a
// non-root executor must retain its Temporal-boundary handling — re-emitting
// the encoded end ApplicationError — rather than exposing a raw inline
// flow.ErrEnd solely because registration was disabled.
func TestWorkflowExecutorDisableRegisterWithoutInlineKeepsBoundaryHandling(t *testing.T) {
	const childName = "disable-register-no-inline-child"
	const parentName = "disable-register-no-inline-parent"

	builder := newTestDoTaskBuilder(childName, DoTaskOpts{
		DisableRegisterWorkflow: true,
		InlineExecution:         false,
	})

	runOrder := make([]string, 0, 2)
	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrEnd, &runOrder),
		newSimpleWorkflowFunc("task-after-end", &model.TaskBase{}, &runOrder),
	}
	childWf := builder.workflowExecutor(tasks)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return childWf(ctx, input, st)
	}, workflow.RegisterOptions{Name: childName})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		var res any
		err := workflow.ExecuteChildWorkflow(ctx, childName, nil, utils.NewState()).Get(ctx, &res)
		return res, err
	}, workflow.RegisterOptions{Name: parentName})

	env.ExecuteWorkflow(parentName)

	err := env.GetWorkflowError()
	require.Error(t, err)
	// Boundary handling (not inline) must have run: the end directive is
	// re-emitted as the typed end ApplicationError, not a raw flow.ErrEnd.
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected a Temporal ApplicationError, got %T: %v", err, err)
	assert.Equal(t, flow.EndApplicationErrorType, appErr.Type(),
		"disabling registration alone must not enable inline flow.ErrEnd propagation")
	assert.Equal(t, []string{testConstTaskSwitch}, runOrder)
}

// TestBuildInlineExecutionDoesNotDisableRegistration proves registration is
// controlled exclusively by DisableRegisterWorkflow: with it false, the
// generated function is still registered even when InlineExecution is true.
func TestBuildInlineExecutionDoesNotDisableRegistration(t *testing.T) {
	w := new(WorkflowRegistryMock)
	w.On("RegisterWorkflowWithOptions", mock.Anything, workflow.RegisterOptions{Name: "reg-inline"}).Once()

	b, err := NewDoTaskBuilder(
		w,
		&model.DoTask{Do: &model.TaskList{
			&model.TaskItem{Key: testConstStep, Task: &model.SetTask{}},
		}},
		"reg-inline",
		testWorkflow,
		testEvents,
		nil,
		DoTaskOpts{DisableRegisterWorkflow: false, InlineExecution: true},
	)
	require.NoError(t, err)

	_, err = b.Build()
	require.NoError(t, err)

	w.AssertExpectations(t)
}

// TestBuildDisableRegisterWorkflowSuppressesRegistrationRegardlessOfInline
// proves the converse: DisableRegisterWorkflow true suppresses registration
// whatever InlineExecution is set to.
func TestBuildDisableRegisterWorkflowSuppressesRegistrationRegardlessOfInline(t *testing.T) {
	for _, inline := range []bool{false, true} {
		w := new(WorkflowRegistryMock)

		b, err := NewDoTaskBuilder(
			w,
			&model.DoTask{Do: &model.TaskList{
				&model.TaskItem{Key: testConstStep, Task: &model.SetTask{}},
			}},
			"noreg-inline",
			testWorkflow,
			testEvents,
			nil,
			DoTaskOpts{DisableRegisterWorkflow: true, InlineExecution: inline},
		)
		require.NoError(t, err)

		_, err = b.Build()
		require.NoError(t, err)

		w.AssertNotCalled(t, "RegisterWorkflowWithOptions", mock.Anything, mock.Anything)
	}
}

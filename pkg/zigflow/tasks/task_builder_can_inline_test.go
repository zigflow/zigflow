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
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// These tests cover the ownership model for Continue-As-New after structural
// task bodies (for/fork/try) were made to run inline inside the owning
// registered workflow.
//
// The invariants under test:
//   - Only the registered (non-inline) executor mints a Temporal
//     Continue-As-New error, and it always uses its own registered workflow
//     type — never an empty string and never a nested/structural task name.
//   - An inline structural body never independently issues Continue-As-New.
//   - Continue-As-New is evaluated at safe root-owned checkpoints: before a
//     structural task begins (preflight) and after it completes.
//   - A Continue-As-New error is never routed into a try catch body.

// runRootIterateWithCAN registers a root (registered, non-inline) workflow that
// drives root.iterateTasks with Continue-As-New already suggested, and returns
// the resulting workflow error. This mirrors how the registered executor
// evaluates continuation before each task in its list.
func runRootIterateWithCAN(
	t *testing.T, rootName string, tasks []workflowFunc, state *utils.State,
) error {
	t.Helper()

	root := newTestDoTaskBuilder(rootName)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetContinueAsNewSuggested(true)
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, root.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: rootName})

	env.ExecuteWorkflow(rootName)

	return env.GetWorkflowError()
}

// assertContinuesWithType asserts the error is a Continue-As-New error whose
// target workflow type is exactly wantType (i.e. the registered workflow type),
// never empty and never a structural task path.
func assertContinuesWithType(t *testing.T, err error, wantType string) {
	t.Helper()

	require.Error(t, err)
	require.True(t, workflow.IsContinueAsNewError(err), "expected a continue-as-new error, got %v", err)

	var canErr *workflow.ContinueAsNewError
	require.True(t, errors.As(err, &canErr), "expected *workflow.ContinueAsNewError, got %T", err)
	require.NotNil(t, canErr.WorkflowType)
	assert.NotEmpty(t, canErr.WorkflowType.Name, "continue-as-new must never use an empty workflow type")
	assert.Equal(t, wantType, canErr.WorkflowType.Name,
		"continue-as-new must target the registered workflow type")
}

// TestInlineExecutorDoesNotContinueAsNew is the core regression: an inline
// structural body (InlineExecution: true, empty registered name) must never
// mint a Continue-As-New error even when Temporal suggests one. Doing so would
// use an empty workflow type and a nested resume marker the root cannot
// interpret. Instead the inline body runs its tasks normally and lets the
// owning registered executor own continuation.
func TestInlineExecutorDoesNotContinueAsNew(t *testing.T) {
	inline := newTestDoTaskBuilder("", DoTaskOpts{
		DisableRegisterWorkflow: true,
		InlineExecution:         true,
	})

	runOrder := make([]string, 0, 2)
	state := utils.NewState()
	tasks := []workflowFunc{
		newSimpleWorkflowFunc(testConstTaskOne, &model.TaskBase{}, &runOrder),
		newSimpleWorkflowFunc(testConstTaskTwo, &model.TaskBase{}, &runOrder),
	}

	// Register the wrapper under a non-empty owning workflow type; the inline
	// builder itself has an empty name and must not use it for continuation.
	const owningWorkflow = "owning-registered-wf"
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetContinueAsNewSuggested(true)
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, inline.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: owningWorkflow})

	env.ExecuteWorkflow(owningWorkflow)

	err := env.GetWorkflowError()
	assert.NoError(t, err)
	assert.False(t, workflow.IsContinueAsNewError(err),
		"inline execution must never issue continue-as-new")
	assert.Equal(t, []string{testConstTaskOne, testConstTaskTwo}, runOrder,
		"inline body must run all tasks rather than continue-as-new")
	assert.Nil(t, state.CANStartFrom, "inline execution must not persist a resume marker")
}

// TestRegisteredExecutorContinuesWithRegisteredType proves the registered
// executor still Continue-As-News when suggested, uses its own registered
// workflow type, records the resume task, and does not run the task it stops
// before.
func TestRegisteredExecutorContinuesWithRegisteredType(t *testing.T) {
	const rootName = "registered-root-wf"

	runOrder := make([]string, 0, 1)
	state := utils.NewState()
	tasks := []workflowFunc{
		newSimpleWorkflowFunc(testConstTaskOne, &model.TaskBase{}, &runOrder),
	}

	err := runRootIterateWithCAN(t, rootName, tasks, state)

	assertContinuesWithType(t, err, rootName)
	assert.Empty(t, runOrder, "the task must not run when continuation happens before it")
	if assert.NotNil(t, state.CANStartFrom) {
		assert.Equal(t, testConstTaskOne+"-0", *state.CANStartFrom,
			"the resume marker must name the task the new run resumes at")
	}
}

// structuralPreflightCase describes a structural task whose inline body must not
// run when the registered executor performs a preflight Continue-As-New.
type structuralPreflightCase struct {
	name       string
	structName string
	// build returns the structural task's exec function plus a pointer to a flag
	// that is set true only if the inline body actually executed.
	build func(t *testing.T) (TemporalWorkflowFunc, TaskBuilder, *bool)
}

// TestStructuralPreflightContinueAsNew proves that for each of for/fork/try the
// registered executor can Continue-As-New before the structural task begins when
// continuation is already suggested. The structural body must not partially
// execute, the resumed run must start at the structural task, and the registered
// workflow type must be used.
func TestStructuralPreflightContinueAsNew(t *testing.T) {
	cases := []structuralPreflightCase{
		{
			name:       "for",
			structName: "loop",
			build: func(t *testing.T) (TemporalWorkflowFunc, TaskBuilder, *bool) {
				bodyRan := false
				b := newForTestBuilder("loop", model.ForTaskConfiguration{In: testConstForDataItems}, "")
				fn, err := b.exec(func(workflow.Context, any, *utils.State) (any, error) {
					bodyRan = true
					return nil, nil
				})
				require.NoError(t, err)
				return fn, b, &bodyRan
			},
		},
		{
			name:       "fork",
			structName: "fork-task",
			build: func(t *testing.T) (TemporalWorkflowFunc, TaskBuilder, *bool) {
				branchRan := false
				b := newForkExecBuilder(false)
				fn, err := b.exec([]forkBranch{{
					name: "branch",
					fn: func(workflow.Context, any, *utils.State) (any, error) {
						branchRan = true
						return nil, nil
					},
				}})
				require.NoError(t, err)
				return fn, b, &branchRan
			},
		},
		{
			name:       "try",
			structName: "try-task",
			build: func(t *testing.T) (TemporalWorkflowFunc, TaskBuilder, *bool) {
				bodyRan := false
				b := newInlineTryBuilder("")
				fn, err := b.exec(
					func(workflow.Context, any, *utils.State) (any, error) {
						bodyRan = true
						return nil, nil
					},
					func(workflow.Context, any, *utils.State) (any, error) {
						bodyRan = true
						return nil, nil
					},
				)
				require.NoError(t, err)
				return fn, b, &bodyRan
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, builder, bodyRan := tc.build(t)

			rootName := "registered-preflight-" + tc.name
			state := utils.NewState()
			tasks := []workflowFunc{{
				TaskBuilder: builder,
				Name:        tc.structName,
				Func:        fn,
			}}

			err := runRootIterateWithCAN(t, rootName, tasks, state)

			assertContinuesWithType(t, err, rootName)
			assert.False(t, *bodyRan, "the structural body must not partially execute during preflight")
			if assert.NotNil(t, state.CANStartFrom) {
				assert.Equal(t, tc.structName+"-0", *state.CANStartFrom,
					"the resumed run must start at the structural task")
			}
		})
	}
}

// TestTryContinueAsNewBypassesCatch proves a Continue-As-New error surfacing
// from the try body is never routed into the catch body: it propagates
// unchanged to the owning executor.
func TestTryContinueAsNewBypassesCatch(t *testing.T) {
	const rootName = "try-can-bypass"

	builder := newInlineTryBuilder("")

	catchRan := false
	tryFn := func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
		// A Continue-As-New error minted against the owning registered type,
		// exactly as the registered executor would produce it.
		return nil, workflow.NewContinueAsNewError(ctx, rootName)
	}
	catchFn := func(workflow.Context, any, *utils.State) (any, error) {
		catchRan = true
		return nil, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	_, execErr := runInlineWorkflowFunc(t, rootName, fn, nil, utils.NewState())

	require.True(t, workflow.IsContinueAsNewError(execErr),
		"a continue-as-new error must propagate from the try body unchanged")
	assert.False(t, catchRan, "continue-as-new must never run the catch body")
}

// TestTryCatchContinueAsNewBypassesFurtherHandling proves a Continue-As-New
// request from the catch body (after the try body genuinely failed) is
// propagated as control flow rather than wrapped as a catch failure.
func TestTryCatchContinueAsNewBypassesFurtherHandling(t *testing.T) {
	const rootName = "try-catch-can-bypass"

	builder := newInlineTryBuilder("")

	tryFn := func(workflow.Context, any, *utils.State) (any, error) {
		return nil, errors.New("boom")
	}
	catchFn := func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
		return nil, workflow.NewContinueAsNewError(ctx, rootName)
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	_, execErr := runInlineWorkflowFunc(t, rootName, fn, nil, utils.NewState())

	assert.True(t, workflow.IsContinueAsNewError(execErr),
		"a continue-as-new error from the catch body must propagate unchanged")
}

// TestTryOrdinaryErrorStillCaught guards that adding the Continue-As-New bypass
// did not stop ordinary failures from being caught.
func TestTryOrdinaryErrorStillCaught(t *testing.T) {
	builder := newInlineTryBuilder("")

	catchRan := false
	tryFn := func(workflow.Context, any, *utils.State) (any, error) {
		return nil, errors.New("boom")
	}
	catchFn := func(workflow.Context, any, *utils.State) (any, error) {
		catchRan = true
		return map[string]any{testConstHandledKey: true}, nil
	}

	fn, err := builder.exec(tryFn, catchFn)
	require.NoError(t, err)

	output, execErr := runInlineWorkflowFunc(t, "try-ordinary-error", fn, nil, utils.NewState())
	require.NoError(t, execErr)
	assert.True(t, catchRan, "ordinary task failures must still be caught")
	assert.Equal(t, map[string]any{testConstHandledKey: true}, output)
}

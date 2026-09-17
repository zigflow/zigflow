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

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// testCancelAfter is the point at which the regression tests request
// workflow cancellation. It is well inside testCancelWait so the
// cancellation always lands while the wait is still running.
const testCancelAfter = time.Second

// testActivityCancelAfter is the cancellation delay used by the
// activity-based tests. Those activities block on a real context rather
// than a skippable workflow timer, so the delay is spent in wall-clock
// time and is kept short.
const testActivityCancelAfter = 100 * time.Millisecond

// testCancelWait is the wait duration used by the cancellation
// regression tests. It is long enough that the workflow could never
// finish it on its own, so a test that completes proves the
// cancellation interrupted the wait rather than the wait expiring.
const testCancelWait = time.Hour

// TestDoTaskBuilderTaskContextObservesWorkflowCancellation proves the
// workflow.Context that iterateTasks hands to a task still carries the
// root workflow's cancellation.
//
// iterateTasks replaces ctx with the context returned by
// metadata.SetActivityOptions before running each task, so a task never
// sees the root context directly. If that derivation lost the
// cancellation relationship, a cancellable operation inside a task
// would keep blocking after the workflow had been cancelled.
func TestDoTaskBuilderTaskContextObservesWorkflowCancellation(t *testing.T) {
	builder := newTestDoTaskBuilder("cancel-context")

	var sleepErr error
	var ctxErr error

	tasks := []workflowFunc{
		{
			TaskBuilder: newFakeTaskBuilder("sleep", &model.TaskBase{}),
			Name:        "sleep",
			Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				sleepErr = workflow.Sleep(ctx, testCancelWait)
				ctxErr = ctx.Err()
				return nil, sleepErr
			},
		},
	}

	env := newCancellingTestEnv(t, builder.workflowExecutor(tasks), "cancel-context")
	env.ExecuteWorkflow("cancel-context")

	assert.True(t, temporal.IsCanceledError(sleepErr),
		"the task context must observe the workflow cancellation")
	assert.Error(t, ctxErr, "the task context must report itself as cancelled")
}

// TestDoTaskBuilderCancellationStopsIteration proves a cancelled task
// cannot cause execution to proceed to the next task, and that the
// cancellation is returned to the caller of iterateTasks rather than
// being absorbed as a non-error outcome.
func TestDoTaskBuilderCancellationStopsIteration(t *testing.T) {
	builder := newTestDoTaskBuilder("cancel-stops-iteration")

	runOrder := make([]string, 0, 2)

	tasks := []workflowFunc{
		{
			TaskBuilder: newFakeTaskBuilder(testConstTaskCancelled, &model.TaskBase{}),
			Name:        testConstTaskCancelled,
			Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				runOrder = append(runOrder, testConstTaskCancelled)
				return nil, temporal.NewCanceledError()
			},
		},
		{
			TaskBuilder: newFakeTaskBuilder(testConstTaskAfter, &model.TaskBase{}),
			Name:        testConstTaskAfter,
			Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				runOrder = append(runOrder, testConstTaskAfter)
				return nil, nil
			},
		},
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: "cancel-stops-iteration"})

	env.ExecuteWorkflow("cancel-stops-iteration")

	err := env.GetWorkflowError()
	require.Error(t, err, "a cancelled task must not be reported as a successful iteration")
	assert.True(t, temporal.IsCanceledError(err), "iterateTasks must surface the cancellation")
	assert.Equal(t, []string{testConstTaskCancelled}, runOrder,
		"no task after the cancelled one may run")
}

// TestDoTaskBuilderCancellationReachesWorkflowBoundary is the
// regression test for the reported bug: a `set` task, a long wait, then
// another `set` task. Cancelling during the wait must interrupt it,
// must not run the trailing task, and must close the workflow as a
// cancellation rather than a completion.
//
// Before the fix the wait reported cancellation as a successful result
// and the do loop carried on, so the workflow completed with the
// trailing task's output.
func TestDoTaskBuilderCancellationReachesWorkflowBoundary(t *testing.T) {
	builder := newTestDoTaskBuilder("cancel-boundary")

	waitBuilder, err := NewWaitTaskBuilder(nil, &model.WaitTask{
		Wait: &model.Duration{Value: model.DurationInline{Hours: 1}},
	}, "sleep", testWorkflow, testEvents, nil)
	require.NoError(t, err)

	waitFn, err := waitBuilder.Build()
	require.NoError(t, err)

	runOrder := make([]string, 0, 3)
	var capturedState *utils.State

	tasks := []workflowFunc{
		{
			TaskBuilder: newFakeTaskBuilder(testConstTaskBefore, &model.TaskBase{}),
			Name:        testConstTaskBefore,
			Func: func(ctx workflow.Context, input any, st *utils.State) (any, error) {
				runOrder = append(runOrder, testConstTaskBefore)
				capturedState = st
				return map[string]any{testConstStage: testConstTaskBefore}, nil
			},
		},
		{
			TaskBuilder: waitBuilder,
			Name:        "sleep",
			Func:        waitFn,
		},
		{
			TaskBuilder: newFakeTaskBuilder(testConstTaskAfter, &model.TaskBase{}),
			Name:        testConstTaskAfter,
			Func: func(ctx workflow.Context, input any, st *utils.State) (any, error) {
				runOrder = append(runOrder, testConstTaskAfter)
				return map[string]any{testConstStage: "after_wait_ran"}, nil
			},
		},
	}

	env := newCancellingTestEnv(t, builder.workflowExecutor(tasks), "cancel-boundary")
	env.ExecuteWorkflow("cancel-boundary")

	werr := env.GetWorkflowError()
	require.Error(t, werr, "a cancelled workflow must not complete successfully")
	assert.True(t, temporal.IsCanceledError(werr),
		"the cancellation must reach the workflow boundary as a cancellation error")
	assert.Equal(t, []string{testConstTaskBefore}, runOrder,
		"the task after the cancelled wait must not run")
	require.NotNil(t, capturedState)
	assert.Equal(t, map[string]any{testConstStage: testConstTaskBefore}, capturedState.Output,
		"the trailing task must not have contributed an output")
}

// TestWaitTaskBuilderPropagatesCancellation is the focused builder test:
// a cancelled timer must be reported as a cancellation, not as a
// completed wait.
func TestWaitTaskBuilderPropagatesCancellation(t *testing.T) {
	builder, err := NewWaitTaskBuilder(nil, &model.WaitTask{
		Wait: &model.Duration{Value: model.DurationInline{Hours: 1}},
	}, "wait", testWorkflow, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	env := newCancellingTestEnv(t, fn, "wait-cancel")
	env.ExecuteWorkflow("wait-cancel")

	assert.True(t, temporal.IsCanceledError(env.GetWorkflowError()),
		"a cancelled wait must return the cancellation")
}

// TestWaitExtBuilderPropagatesCancellation covers both wait extension
// forms: an inline duration and an `until` timestamp. Both sleep on the
// workflow timer and both must report cancellation.
func TestWaitExtBuilderPropagatesCancellation(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		body *models.WaitExtBody
	}{
		{
			name: "inline duration",
			body: &models.WaitExtBody{Hours: 1},
		},
		{
			name: "until timestamp",
			body: &models.WaitExtBody{Until: start.Add(testCancelWait).Format(time.RFC3339)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder, err := NewWaitExtTaskBuilder(nil, &models.WaitExtTask{Wait: tc.body}, "wait-ext", nil, testEvents, nil)
			require.NoError(t, err)

			fn, err := builder.Build()
			require.NoError(t, err)

			env := newCancellingTestEnv(t, fn, "wait-ext-cancel")
			env.SetStartTime(start)
			env.ExecuteWorkflow("wait-ext-cancel")

			assert.True(t, temporal.IsCanceledError(env.GetWorkflowError()),
				"a cancelled wait extension must return the cancellation")
		})
	}
}

// TestListenTaskBuilderPropagatesWorkflowCancellation proves a listener
// blocked on a signal reports a workflow cancellation.
//
// The listener awaits on its own cancellable child context, which its
// signal handler uses to abort. Both that internal abort and a genuine
// workflow cancellation surface from AwaitWithTimeout as a
// CanceledError, so the builder must look at the originating workflow
// context to tell them apart.
func TestListenTaskBuilderPropagatesWorkflowCancellation(t *testing.T) {
	builder, err := NewListenTaskBuilder(nil, &model.ListenTask{
		Listen: model.ListenTaskConfiguration{
			To: &model.EventConsumptionStrategy{
				One: &model.EventFilter{
					With: &model.EventProperties{
						ID:   "sig-1",
						Type: string(ListenTaskTypeSignal),
					},
				},
			},
		},
	}, "listen", testWorkflow, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	env := newCancellingTestEnv(t, fn, "listen-cancel")
	env.ExecuteWorkflow("listen-cancel")

	assert.True(t, temporal.IsCanceledError(env.GetWorkflowError()),
		"a listener cancelled by the workflow must return the cancellation")
}

// TestListenTaskBuilderInternalAbortIsNotCancellation guards the other
// side of the listen fix: when the listener's own signal handler aborts
// the await (here by failing to evaluate acceptIf) the workflow itself
// has not been cancelled. That abort must keep its existing failure
// behaviour and must not be reported as a Temporal cancellation, which
// would cancel an execution nobody asked to cancel.
func TestListenTaskBuilderInternalAbortIsNotCancellation(t *testing.T) {
	builder, err := NewListenTaskBuilder(nil, &model.ListenTask{
		Listen: model.ListenTaskConfiguration{
			To: &model.EventConsumptionStrategy{
				One: &model.EventFilter{
					With: &model.EventProperties{
						ID:   "sig-1",
						Type: string(ListenTaskTypeSignal),
						Additional: map[string]any{
							// Not a valid expression against the signal
							// payload, so getAcceptIf fails and the
							// handler aborts the await internally.
							"acceptIf": "${ .nope | fromjson }",
						},
					},
				},
			},
		},
	}, "listen", testWorkflow, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: "listen-abort"})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("sig-1", map[string]any{"value": "ignored"})
	}, testCancelAfter)

	env.ExecuteWorkflow("listen-abort")

	err = env.GetWorkflowError()
	require.Error(t, err)
	assert.False(t, temporal.IsCanceledError(err),
		"an internal listener abort must not be reported as a cancellation")
}

// TestActivityDispatchPropagatesCancellation covers the task builders
// that reach Temporal through an activity. Cancelling the workflow
// cancels the in-flight activity, and each dispatch path must report
// that as a cancellation rather than as an empty but successful
// activity result.
func TestActivityDispatchPropagatesCancellation(t *testing.T) {
	const activityName = "blocking-activity"

	tests := []struct {
		name string
		run  func(ctx workflow.Context) (any, error)
	}{
		{
			// The shared path used by call.http and call.grpc.
			name: "shared activity dispatch",
			run: func(ctx workflow.Context) (any, error) {
				b := &builder[*model.WaitTask]{name: "call", doc: testWorkflow, task: &model.WaitTask{}}
				return b.executeActivity(ctx, activityName, nil, utils.NewState())
			},
		},
		{
			name: "run command dispatch",
			run: func(ctx workflow.Context) (any, error) {
				b := &RunTaskBuilder{}
				b.name = "run"
				b.doc = testWorkflow
				b.task = &model.RunTask{}
				return b.executeCommand(ctx, activityName, nil, utils.NewState())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			env.RegisterActivityWithOptions(func(
				ctx context.Context, task, input, state any,
			) (any, error) {
				<-ctx.Done()
				return nil, temporal.NewCanceledError()
			}, activity.RegisterOptions{Name: activityName})

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					StartToCloseTimeout: time.Minute,
					HeartbeatTimeout:    time.Second,
					WaitForCancellation: true,
				})
				return tc.run(ctx)
			}, workflow.RegisterOptions{Name: "activity-cancel"})

			env.RegisterDelayedCallback(env.CancelWorkflow, testActivityCancelAfter)
			env.ExecuteWorkflow("activity-cancel")

			assert.True(t, temporal.IsCanceledError(env.GetWorkflowError()),
				"a cancelled activity must return the cancellation")
		})
	}
}

// TestCallActivityTaskBuilderPropagatesCancellation covers
// call.activity, which dispatches a user-named Temporal activity
// through its own code path rather than the shared helper.
func TestCallActivityTaskBuilderPropagatesCancellation(t *testing.T) {
	const activityName = "userBlockingActivity"

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(func(ctx context.Context) (any, error) {
		<-ctx.Done()
		return nil, temporal.NewCanceledError()
	}, activity.RegisterOptions{Name: activityName})

	b, err := NewCallActivityTaskBuilder(nil, &model.CallFunction{
		Call: customCallFunctionActivity,
		With: map[string]any{
			"name":      activityName,
			"taskQueue": "some-task-queue",
		},
	}, "callActivity", testWorkflow, testEvents, nil)
	require.NoError(t, err)

	fn, err := b.Build()
	require.NoError(t, err)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			HeartbeatTimeout:    time.Second,
			WaitForCancellation: true,
		})
		return fn(ctx, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: "call-activity-cancel"})

	env.RegisterDelayedCallback(env.CancelWorkflow, testActivityCancelAfter)
	env.ExecuteWorkflow("call-activity-cancel")

	assert.True(t, temporal.IsCanceledError(env.GetWorkflowError()),
		"a cancelled call.activity must return the cancellation")
}

// newCancellingTestEnv builds a test environment that runs fn as its
// workflow and requests cancellation shortly after it starts.
func newCancellingTestEnv(
	t *testing.T, fn TemporalWorkflowFunc, name string,
) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, map[string]any{}, nil)
	}, workflow.RegisterOptions{Name: name})

	env.RegisterDelayedCallback(env.CancelWorkflow, testCancelAfter)

	return env
}

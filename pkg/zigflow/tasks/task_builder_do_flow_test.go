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
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"sigs.k8s.io/yaml"
)

// newSwitchWorkflowFunc returns a workflowFunc that mimics a matched
// SwitchTaskBuilder: it records its invocation in runOrder and returns
// the supplied flow-directive error.
func newSwitchWorkflowFunc(directive error, runOrder *[]string) workflowFunc {
	tb := newFakeTaskBuilder(testConstTaskSwitch, &model.TaskBase{})
	return workflowFunc{
		TaskBuilder: tb,
		Name:        testConstTaskSwitch,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			*runOrder = append(*runOrder, testConstTaskSwitch)
			return nil, directive
		},
	}
}

func TestDoTaskBuilderIterateTasksContinue(t *testing.T) {
	builder := newTestDoTaskBuilder("iterate-continue")

	runOrder := make([]string, 0, 2)
	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrContinue, &runOrder),
		newSimpleWorkflowFunc(testConstTaskTwo, &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	assert.NoError(t, env.GetWorkflowError())
	assert.Equal(t, []string{testConstTaskSwitch, testConstTaskTwo}, runOrder)
}

func TestDoTaskBuilderIterateTasksExit(t *testing.T) {
	builder := newTestDoTaskBuilder("iterate-exit")

	runOrder := make([]string, 0, 1)
	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrExit, &runOrder),
		newSimpleWorkflowFunc(testConstTaskTwo, &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	// flow.ErrExit ends the current do scope cleanly, without
	// propagating an error and without running subsequent tasks.
	assert.NoError(t, env.GetWorkflowError())
	assert.Equal(t, []string{testConstTaskSwitch}, runOrder)
}

func TestDoTaskBuilderIterateTasksEndPropagates(t *testing.T) {
	builder := newTestDoTaskBuilder("iterate-end-propagate")

	runOrder := make([]string, 0, 1)
	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrEnd, &runOrder),
		newSimpleWorkflowFunc(testConstTaskTwo, &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	// iterateTasks propagates flow.ErrEnd outward so the enclosing
	// workflowExecutor can interpret it as a clean termination.
	err := env.GetWorkflowError()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), flow.ErrEnd.Error())
	assert.Equal(t, []string{testConstTaskSwitch}, runOrder)
}

func TestDoTaskBuilderWorkflowExecutorEndsCleanly(t *testing.T) {
	builder := newTestDoTaskBuilder("executor-end")
	expectedOutput := map[string]any{testConstValue: "captured"}

	runOrder := make([]string, 0, 1)
	taskOne := workflowFunc{
		TaskBuilder: newFakeTaskBuilder(testConstTaskOne, &model.TaskBase{}),
		Name:        testConstTaskOne,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			runOrder = append(runOrder, testConstTaskOne)
			return expectedOutput, nil
		},
	}
	taskEnd := newSwitchWorkflowFunc(flow.ErrEnd, &runOrder)
	taskAfter := newSimpleWorkflowFunc("task-after", &model.TaskBase{}, &runOrder)

	wf := builder.workflowExecutor([]workflowFunc{taskOne, taskEnd, taskAfter})

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return wf(ctx, nil, nil)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	// At the workflow boundary flow.ErrEnd becomes a successful
	// completion that returns the last computed state.Output.
	assert.NoError(t, env.GetWorkflowError())

	var result map[string]any
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, expectedOutput, result)
	assert.Equal(t, []string{testConstTaskOne, testConstTaskSwitch}, runOrder)
}

func TestDoTaskBuilderIterateTasksRedirectExecutesChildWorkflow(t *testing.T) {
	const redirectTarget = "redirect-child"

	builder := newTestDoTaskBuilder("iterate-redirect")

	runOrder := make([]string, 0, 2)
	childRan := false

	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.RedirectError{Target: redirectTarget}, &runOrder),
		newSimpleWorkflowFunc(testConstTaskTwo, &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		childRan = true
		return nil, nil
	}, workflow.RegisterOptions{Name: redirectTarget})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	// The redirect target should be dispatched as a child workflow,
	// after which iteration resumes with subsequent tasks.
	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, childRan, "redirect target child workflow should have run")
	assert.Equal(t, []string{testConstTaskSwitch, testConstTaskTwo}, runOrder)
}

// TestDoTaskBuilderIterateTasksRedirectChildEndsHaltsParent proves that a
// `then: end` directive raised inside a redirect target child workflow is
// observable in the parent's iterateTasks loop. The redirect target
// returns the Temporal end ApplicationError minted by
// flow.NewEndApplicationError; executeRedirect must decode it back to
// flow.ErrEnd so the parent stops running subsequent tasks and surfaces
// "end" outward instead of treating it as success.
func TestDoTaskBuilderIterateTasksRedirectChildEndsHaltsParent(t *testing.T) {
	const redirectTarget = "redirect-child-ends"

	builder := newTestDoTaskBuilder("iterate-redirect-end")

	runOrder := make([]string, 0, 2)

	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.RedirectError{Target: redirectTarget}, &runOrder),
		newSimpleWorkflowFunc(testConstTaskTwo, &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return nil, flow.NewEndApplicationError(nil)
	}, workflow.RegisterOptions{Name: redirectTarget})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	err := env.GetWorkflowError()
	require.Error(t, err)
	// The Temporal env surfaces propagated errors as ApplicationErrors;
	// the message must still mention flow.ErrEnd so callers can identify
	// it as an end directive rather than a generic failure.
	assert.Contains(t, err.Error(), flow.ErrEnd.Error())
	// Crucially, the task that follows the redirect must not have run.
	assert.Equal(t, []string{testConstTaskSwitch}, runOrder)
}

// TestDoTaskBuilderWorkflowExecutorNestedEndReEmitsTemporal verifies that
// when a workflowExecutor is invoked as a Temporal child workflow (i.e.
// not the root) and one of its tasks emits flow.ErrEnd, the executor
// returns the serialisable Temporal end ApplicationError to its parent
// rather than swallowing the directive into a clean completion. Only the
// root workflow should treat ErrEnd as success.
func TestDoTaskBuilderWorkflowExecutorNestedEndReEmitsTemporal(t *testing.T) {
	const childName = "nested-end-child"
	const parentName = "nested-end-parent"

	builder := newTestDoTaskBuilder(childName)

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
	// errors.As must descend through the ChildWorkflowExecutionError
	// wrapper into the underlying ApplicationError so the parent can
	// recognise the directive without string-matching.
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected a Temporal ApplicationError, got %T: %v", err, err)
	assert.Equal(t, flow.EndApplicationErrorType, appErr.Type())
	assert.True(t, flow.IsEndApplicationError(err))
	// The task that followed the end directive inside the child must not
	// have run; the child should stop iterating immediately.
	assert.Equal(t, []string{testConstTaskSwitch}, runOrder)
}

// TestDoTaskBuilderRootWorkflowExecutorAcrossRedirectEndsCleanly is the
// end-to-end shape of the fix: a root workflowExecutor performs a redirect
// to a child workflowExecutor which signals `then: end`. The end directive
// must travel back to the root over the child workflow boundary, the root
// must complete cleanly (no error), and no tasks after the redirect in
// the root scope must execute.
func TestDoTaskBuilderRootWorkflowExecutorAcrossRedirectEndsCleanly(t *testing.T) {
	const redirectTarget = "root-redirect-end-target"

	rootBuilder := newTestDoTaskBuilder("root-redirect-end")
	childBuilder := newTestDoTaskBuilder(redirectTarget)

	runOrder := make([]string, 0, 3)

	childTasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrEnd, &runOrder),
		newSimpleWorkflowFunc("child-after-end", &model.TaskBase{}, &runOrder),
	}
	childWf := childBuilder.workflowExecutor(childTasks)

	parentTasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.RedirectError{Target: redirectTarget}, &runOrder),
		newSimpleWorkflowFunc("root-after-redirect", &model.TaskBase{}, &runOrder),
	}
	rootWf := rootBuilder.workflowExecutor(parentTasks)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return childWf(ctx, input, st)
	}, workflow.RegisterOptions{Name: redirectTarget})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return rootWf(ctx, nil, nil)
	}, workflow.RegisterOptions{Name: rootBuilder.GetTaskName()})

	env.ExecuteWorkflow(rootBuilder.GetTaskName())

	// At the root the end directive becomes a clean termination.
	assert.NoError(t, env.GetWorkflowError())
	// The child's task after end and the root's task after the redirect
	// both must not have run.
	assert.Equal(t, []string{testConstTaskSwitch, testConstTaskSwitch}, runOrder)
}

// newRecordingEvents wires up a cloudevents.Events that writes to a temp
// directory using the file protocol. The returned reader returns the list
// of emitted event types in the order they were appended, so tests can
// assert which task lifecycle events fired.
func newRecordingEvents(t *testing.T) (events *cloudevents.Events, readEventTypes func() []string) {
	t.Helper()
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	require.NoError(t, os.MkdirAll(eventsDir, 0o755))

	config := fmt.Sprintf("clients:\n  - name: recorder\n    protocol: file\n    target: %s\n", eventsDir)
	configPath := filepath.Join(dir, "cloudevents.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

	validator, err := utils.NewValidator()
	require.NoError(t, err)
	events, err = cloudevents.Load(configPath, validator, testWorkflow)
	require.NoError(t, err)

	readEventTypes = func() []string {
		entries, err := os.ReadDir(eventsDir)
		require.NoError(t, err)
		var types []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
			require.NoError(t, err)
			for _, doc := range bytes.Split(data, []byte("---\n")) {
				doc = bytes.TrimSpace(doc)
				if len(doc) == 0 {
					continue
				}
				var meta struct {
					Type string `json:"type"`
				}
				require.NoError(t, yaml.Unmarshal(doc, &meta))
				if meta.Type != "" {
					types = append(types, meta.Type)
				}
			}
		}
		return types
	}

	return events, readEventTypes
}

// switchTaskWithBase mimics a SwitchTaskBuilder's runtime behaviour: it
// returns nil output plus the supplied flow directive, but carries an
// arbitrary TaskBase so the do-task pipeline can apply the task's output
// and export directives.
func switchTaskWithBase(name string, directive error, base *model.TaskBase, runOrder *[]string) workflowFunc {
	tb := newFakeTaskBuilder(name, base)
	return workflowFunc{
		TaskBuilder: tb,
		Name:        name,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			*runOrder = append(*runOrder, name)
			return nil, directive
		},
	}
}

// TestDoTaskBuilderSwitchWithOutputAndExportThenContinue exercises
// Issue 2: a switch task that returns a control directive must still
// run through the normal completion pipeline so its output and export
// expressions update workflow state and downstream tasks observe the
// updates.
//
// The follow-up "observer" task captures state at the moment it runs.
// That is sufficient evidence that the switch's output/export had
// already been applied by the time the next task started, without
// depending on a final state.Output that the observer would overwrite.
func TestDoTaskBuilderSwitchWithOutputAndExportThenContinue(t *testing.T) {
	builder := newTestDoTaskBuilder("switch-output-export")

	runOrder := make([]string, 0, 2)

	switchBase := &model.TaskBase{
		Output: &model.Output{
			As: model.NewObjectOrRuntimeExpr(map[string]any{
				"branchTaken": testConstFlowContinue,
			}),
		},
		Export: &model.Export{
			As: model.NewObjectOrRuntimeExpr(map[string]any{
				"switchObserved": true,
			}),
		},
	}

	var observedOutput any
	var observedContext any
	observer := workflowFunc{
		TaskBuilder: newFakeTaskBuilder("observer", &model.TaskBase{}),
		Name:        "observer",
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			runOrder = append(runOrder, "observer")
			observedOutput = state.Output
			observedContext = state.Context
			return nil, nil
		},
	}

	tasks := []workflowFunc{
		switchTaskWithBase("switcher", flow.ErrContinue, switchBase, &runOrder),
		observer,
	}

	state := utils.NewState()
	state.Context = map[string]any{"existing": "context"}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	require.NoError(t, env.GetWorkflowError())

	// At the moment the next task started, the switch's output and
	// export expressions had been applied.
	assert.Equal(t, map[string]any{"branchTaken": testConstFlowContinue}, observedOutput)
	assert.Equal(t, map[string]any{"switchObserved": true}, observedContext)
	// Continue must let the next task run.
	assert.Equal(t, []string{"switcher", "observer"}, runOrder)
}

// TestDoTaskBuilderSwitchEmitsTaskCompletedForControlDirective exercises
// the second half of Issue 2: switches that emit a control directive
// must still emit task.completed so downstream event consumers see the
// task lifecycle. We wire up a file-sink Events instance and inspect the
// resulting yaml.
func TestDoTaskBuilderSwitchEmitsTaskCompletedForControlDirective(t *testing.T) {
	events, readEventTypes := newRecordingEvents(t)

	builder := &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			doc:          testWorkflow,
			eventEmitter: events,
			name:         "switch-events",
			task:         &model.DoTask{},
		},
	}

	runOrder := make([]string, 0, 1)
	tasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.ErrContinue, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())
	require.NoError(t, env.GetWorkflowError())

	types := readEventTypes()
	assert.Contains(t, types, "dev.zigflow.task.started",
		"a switch emitting a flow directive must still fire task.started")
	assert.Contains(t, types, "dev.zigflow.task.completed",
		"a switch emitting a flow directive must still fire task.completed")
}

// TestDoTaskBuilderTaskLevelThenEndPropagatesAcrossChildBoundary covers
// Issue 1 end-to-end: a regular task with `then: end` inside a redirect
// target child workflow must terminate the parent workflow, not just the
// child. Tasks scheduled in the parent after the redirect must not run.
func TestDoTaskBuilderTaskLevelThenEndPropagatesAcrossChildBoundary(t *testing.T) {
	const redirectTarget = "task-level-end-target"

	rootBuilder := newTestDoTaskBuilder("task-level-end-root")
	childBuilder := newTestDoTaskBuilder(redirectTarget)

	runOrder := make([]string, 0, 4)

	// Inside the child: a regular task carrying taskBase.Then = end.
	// This is exactly the YAML pattern from the issue:
	//   - someTask:
	//       call: ...
	//       then: end
	endingTaskBase := &model.TaskBase{
		Then: &model.FlowDirective{Value: string(model.FlowDirectiveEnd)},
	}
	endingTask := workflowFunc{
		TaskBuilder: newFakeTaskBuilder("ending-task", endingTaskBase),
		Name:        "ending-task",
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			runOrder = append(runOrder, "ending-task")
			return map[string]any{"work": "done"}, nil
		},
	}
	childTasks := []workflowFunc{
		endingTask,
		newSimpleWorkflowFunc("child-after-end", &model.TaskBase{}, &runOrder),
	}
	childWf := childBuilder.workflowExecutor(childTasks)

	// Root scope: switch redirects to the child, then a sibling task that
	// must not run because the child ended the overall workflow.
	parentTasks := []workflowFunc{
		newSwitchWorkflowFunc(flow.RedirectError{Target: redirectTarget}, &runOrder),
		newSimpleWorkflowFunc("root-after-redirect", &model.TaskBase{}, &runOrder),
	}
	rootWf := rootBuilder.workflowExecutor(parentTasks)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return childWf(ctx, input, st)
	}, workflow.RegisterOptions{Name: redirectTarget})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return rootWf(ctx, nil, nil)
	}, workflow.RegisterOptions{Name: rootBuilder.GetTaskName()})

	env.ExecuteWorkflow(rootBuilder.GetTaskName())

	// The root is the topmost workflow, so flow.ErrEnd becomes a clean
	// completion. The decisive evidence that propagation works is that
	// neither "child-after-end" nor "root-after-redirect" ran.
	require.NoError(t, env.GetWorkflowError())
	assert.Equal(t, []string{testConstTaskSwitch, "ending-task"}, runOrder)
}

// TestDoTaskBuilderTaskLevelThenDirectivesDispatchConsistently confirms
// that the four task-level then directives go through the same dispatch
// path as switch-emitted directives:
//   - continue: fall through to the next task
//   - exit: stop the current scope cleanly
//   - end: propagate flow.ErrEnd outward
//   - <name>: jump to a sibling task in the current scope
func TestDoTaskBuilderTaskLevelThenDirectivesDispatchConsistently(t *testing.T) {
	type tc struct {
		name        string
		then        string
		expectedRun []string
		wantErr     error
		wantErrText string
	}

	cases := []tc{
		{
			name:        "continue lets iteration proceed to every subsequent task",
			then:        string(model.FlowDirectiveContinue),
			expectedRun: []string{testConstTaskWithThen, "task-after", testConstTaskTarget},
		},
		{
			name:        "exit stops current scope cleanly",
			then:        string(model.FlowDirectiveExit),
			expectedRun: []string{testConstTaskWithThen},
		},
		{
			name:        "end propagates ErrEnd outward",
			then:        string(model.FlowDirectiveEnd),
			expectedRun: []string{testConstTaskWithThen},
			wantErr:     flow.ErrEnd,
		},
		{
			name:        "named target skips past intermediate task",
			then:        testConstTaskTarget,
			expectedRun: []string{testConstTaskWithThen, testConstTaskTarget},
		},
		{
			// Both task-after and task-target are skipped because they
			// do not match the nextTargetName, then iteration falls off
			// the end with nextTargetName still set, which surfaces the
			// descriptive error.
			name:        "missing named target reports descriptive error",
			then:        "does-not-exist",
			expectedRun: []string{testConstTaskWithThen},
			wantErrText: "next target specified but not found: does-not-exist",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			builder := newTestDoTaskBuilder("task-level-then-" + c.name)
			runOrder := make([]string, 0, 3)

			withThen := newSimpleWorkflowFunc(testConstTaskWithThen, &model.TaskBase{
				Then: &model.FlowDirective{Value: c.then},
			}, &runOrder)
			afterTask := newSimpleWorkflowFunc("task-after", &model.TaskBase{}, &runOrder)
			targetTask := newSimpleWorkflowFunc(testConstTaskTarget, &model.TaskBase{}, &runOrder)
			tasks := []workflowFunc{withThen, afterTask, targetTask}

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				return nil, builder.iterateTasks(ctx, tasks, nil, utils.NewState())
			}, workflow.RegisterOptions{Name: builder.GetTaskName()})

			env.ExecuteWorkflow(builder.GetTaskName())

			err := env.GetWorkflowError()
			switch {
			case c.wantErr != nil:
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErr.Error())
			case c.wantErrText != "":
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErrText)
			default:
				assert.NoError(t, err)
			}
			assert.Equal(t, c.expectedRun, runOrder)
		})
	}
}

// TestDoTaskBuilderRedirectEndCarriesChildOutputToRoot is the
// end-to-end shape of Issue 1: a switch redirects to a child workflow,
// the child sets an output and then signals `then: end`, and the root
// workflow must complete with the child's effective output rather than
// the pre-redirect output, with no parent task running after the
// redirect.
func TestDoTaskBuilderRedirectEndCarriesChildOutputToRoot(t *testing.T) {
	const redirectTarget = "redirect-end-carries-output"
	expectedChildOutput := map[string]any{testConstValue: "child-output"}
	prevOutput := map[string]any{testConstValue: "pre-redirect"}

	rootBuilder := newTestDoTaskBuilder("redirect-end-output-root")
	childBuilder := newTestDoTaskBuilder(redirectTarget)

	runOrder := make([]string, 0, 3)

	// First parent task: writes a stale pre-redirect output so the test
	// can prove the root's final output is the child's, not the prior
	// value.
	priorTask := workflowFunc{
		TaskBuilder: newFakeTaskBuilder(testConstTaskPrior, &model.TaskBase{}),
		Name:        testConstTaskPrior,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			runOrder = append(runOrder, testConstTaskPrior)
			return prevOutput, nil
		},
	}

	// Child task: writes the eventual output, then emits end.
	settingTask := workflowFunc{
		TaskBuilder: newFakeTaskBuilder("set-child-output", &model.TaskBase{}),
		Name:        testConstTaskSetChildOutput,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			runOrder = append(runOrder, "set-child-output")
			return expectedChildOutput, nil
		},
	}
	childEndTask := newSwitchWorkflowFunc(flow.ErrEnd, &runOrder)
	childWf := childBuilder.workflowExecutor([]workflowFunc{settingTask, childEndTask})

	// Root scope: stale output, switch redirect, then a sibling that
	// must never run because the redirect target ended the workflow.
	parentTasks := []workflowFunc{
		priorTask,
		newSwitchWorkflowFunc(flow.RedirectError{Target: redirectTarget}, &runOrder),
		newSimpleWorkflowFunc("root-after-redirect", &model.TaskBase{}, &runOrder),
	}
	rootWf := rootBuilder.workflowExecutor(parentTasks)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return childWf(ctx, input, st)
	}, workflow.RegisterOptions{Name: redirectTarget})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return rootWf(ctx, nil, nil)
	}, workflow.RegisterOptions{Name: rootBuilder.GetTaskName()})

	env.ExecuteWorkflow(rootBuilder.GetTaskName())

	require.NoError(t, env.GetWorkflowError())

	// The child's output must have survived the end propagation. If
	// Issue 1 regresses, the workflow result is either the stale prior
	// output or nil, because the child's effective output was discarded
	// when end crossed the boundary.
	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, expectedChildOutput, result,
		"root workflow output must be the child's effective output, not the pre-redirect value")
	assert.Equal(t, []string{testConstTaskPrior, testConstTaskSwitch, "set-child-output", testConstTaskSwitch}, runOrder)
}

// TestDoTaskBuilderRedirectEndAppliesSwitchOutputToChildPayload proves
// that the originating switch task's own output: directive still
// applies to the child's payload when the child ends. Without this,
// users who shape redirect results in YAML would silently lose their
// transformation on the end path.
func TestDoTaskBuilderRedirectEndAppliesSwitchOutputToChildPayload(t *testing.T) {
	const redirectTarget = "redirect-end-applies-switch-output"

	childOutput := map[string]any{"raw": "child"}
	rootBuilder := newTestDoTaskBuilder("redirect-end-shaped-root")
	childBuilder := newTestDoTaskBuilder(redirectTarget)

	runOrder := make([]string, 0, 2)
	settingTask := workflowFunc{
		TaskBuilder: newFakeTaskBuilder("set-child-output", &model.TaskBase{}),
		Name:        testConstTaskSetChildOutput,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			runOrder = append(runOrder, "set-child-output")
			return childOutput, nil
		},
	}
	childEndTask := newSwitchWorkflowFunc(flow.ErrEnd, &runOrder)
	childWf := childBuilder.workflowExecutor([]workflowFunc{settingTask, childEndTask})

	// The switch carries an output directive that wraps the redirect
	// target's result. The root must complete with the wrapped value.
	switchBase := &model.TaskBase{
		Output: &model.Output{
			As: model.NewObjectOrRuntimeExpr(map[string]any{
				"wrappedFrom": "switch",
			}),
		},
	}
	switchTask := switchTaskWithBase(testConstTaskSwitch, flow.RedirectError{Target: redirectTarget}, switchBase, &runOrder)

	rootWf := rootBuilder.workflowExecutor([]workflowFunc{switchTask})

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, st *utils.State) (any, error) {
		return childWf(ctx, input, st)
	}, workflow.RegisterOptions{Name: redirectTarget})

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return rootWf(ctx, nil, nil)
	}, workflow.RegisterOptions{Name: rootBuilder.GetTaskName()})

	env.ExecuteWorkflow(rootBuilder.GetTaskName())

	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, map[string]any{"wrappedFrom": "switch"}, result,
		"the switch task's output directive must apply to the child's end-time output")
}

// TestDoTaskBuilderTaskCompletedUsesProcessedOutput is the focused
// Issue 2 test: a switch with an output: directive and then: continue
// must fire task.completed with the processed output, not the raw nil
// the switch returned from its Func.
func TestDoTaskBuilderTaskCompletedUsesProcessedOutput(t *testing.T) {
	events, readEventPayloads := newRecordingEventsWithPayload(t)

	builder := &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			doc:          testWorkflow,
			eventEmitter: events,
			name:         "task-completed-processed-output",
			task:         &model.DoTask{},
		},
	}

	processed := map[string]any{testConstValue: "processed-output"}
	runOrder := make([]string, 0, 1)
	switchBase := &model.TaskBase{
		Output: &model.Output{
			As: model.NewObjectOrRuntimeExpr(processed),
		},
	}
	tasks := []workflowFunc{
		switchTaskWithBase(testConstTaskSwitch, flow.ErrContinue, switchBase, &runOrder),
	}

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())
	require.NoError(t, env.GetWorkflowError())

	// State must reflect the processed output.
	assert.Equal(t, processed, state.Output)

	// The emitted task.completed event must carry the processed output
	// rather than the switch's raw nil return value.
	payload := payloadForEvent(t, readEventPayloads(), "dev.zigflow.task.completed")
	require.NotNil(t, payload, "expected a task.completed event to have been emitted")
	assert.Equal(t, processed, payload["output"])
}

// newRecordingEventsWithPayload is like newRecordingEvents but also
// returns the decoded payloads, in order, so tests can assert the
// shape of an event rather than just its type.
func newRecordingEventsWithPayload(t *testing.T) (events *cloudevents.Events, readEvents func() []recordedEvent) {
	t.Helper()
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	require.NoError(t, os.MkdirAll(eventsDir, 0o755))

	config := fmt.Sprintf("clients:\n  - name: recorder\n    protocol: file\n    target: %s\n", eventsDir)
	configPath := filepath.Join(dir, "cloudevents.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

	validator, err := utils.NewValidator()
	require.NoError(t, err)
	events, err = cloudevents.Load(configPath, validator, testWorkflow)
	require.NoError(t, err)

	readEvents = func() []recordedEvent {
		entries, err := os.ReadDir(eventsDir)
		require.NoError(t, err)
		var recorded []recordedEvent
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
			require.NoError(t, err)
			for _, doc := range bytes.Split(data, []byte("---\n")) {
				doc = bytes.TrimSpace(doc)
				if len(doc) == 0 {
					continue
				}
				var ev recordedEvent
				require.NoError(t, yaml.Unmarshal(doc, &ev))
				if ev.Type != "" {
					recorded = append(recorded, ev)
				}
			}
		}
		return recorded
	}
	return events, readEvents
}

type recordedEvent struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// payloadForEvent returns the data payload of the first event of the
// given type, or nil if no such event was recorded.
func payloadForEvent(t *testing.T, events []recordedEvent, eventType string) map[string]any {
	t.Helper()
	for _, ev := range events {
		if ev.Type == eventType {
			return ev.Data
		}
	}
	return nil
}

// recordedEventTypes returns the list of event types in emit order so
// tests can assert presence/absence with simple slice contains.
func recordedEventTypes(events []recordedEvent) []string {
	types := make([]string, 0, len(events))
	for _, ev := range events {
		types = append(types, ev.Type)
	}
	return types
}

// TestDoTaskBuilderCancelledTaskSkipsCompletionPipeline is the focused
// Bug 2 test: a task whose Func returns a Temporal CanceledError must
//
//   - emit task.cancelled
//   - NOT emit task.completed
//   - NOT process output (state.Output remains the previous task's value)
//   - NOT process export (state.Context remains the previous value)
//
// A cancelled task is not a successful completion and must not be
// treated as one. Iteration still continues (cancellation is not a
// flow directive) so any later task in the same scope runs normally.
func TestDoTaskBuilderCancelledTaskSkipsCompletionPipeline(t *testing.T) {
	events, readEvents := newRecordingEventsWithPayload(t)

	builder := &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			doc:          testWorkflow,
			eventEmitter: events,
			name:         "cancellation-skip-pipeline",
			task:         &model.DoTask{},
		},
	}

	priorOutput := map[string]any{testConstValue: testConstTaskPrior}
	priorContext := map[string]any{"context": "before"}

	state := utils.NewState()
	state.Output = priorOutput
	state.Context = priorContext

	runOrder := make([]string, 0, 2)

	// First task primes state; this is what state.Output/Context should
	// still equal after the cancelled task runs.
	priorTask := workflowFunc{
		TaskBuilder: newFakeTaskBuilder(testConstTaskPrior, &model.TaskBase{}),
		Name:        testConstTaskPrior,
		Func: func(ctx workflow.Context, input any, st *utils.State) (any, error) {
			runOrder = append(runOrder, testConstTaskPrior)
			return priorOutput, nil
		},
	}

	// Cancelled task: defines output and export directives that MUST
	// NOT fire. If the cancellation path mistakenly hits the completion
	// pipeline, state.Output/Context would be overwritten.
	cancelledBase := &model.TaskBase{
		Output: &model.Output{
			As: model.NewObjectOrRuntimeExpr(map[string]any{
				"shouldNot": "appear",
			}),
		},
		Export: &model.Export{
			As: model.NewObjectOrRuntimeExpr(map[string]any{
				"shouldNotExport": true,
			}),
		},
	}
	cancelled := workflowFunc{
		TaskBuilder: newFakeTaskBuilder("cancelled", cancelledBase),
		Name:        "cancelled",
		Func: func(ctx workflow.Context, input any, st *utils.State) (any, error) {
			runOrder = append(runOrder, "cancelled")
			return nil, temporal.NewCanceledError()
		},
	}

	tasks := []workflowFunc{priorTask, cancelled}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())
	require.NoError(t, env.GetWorkflowError())

	// state.Output and state.Context must be the prior values: neither
	// the default state.Output assignment nor the explicit output/export
	// directives should have fired for the cancelled task.
	assert.Equal(t, priorOutput, state.Output,
		"cancelled task must not overwrite state.Output")
	assert.Equal(t, priorContext, state.Context,
		"cancelled task must not overwrite state.Context")
	assert.Equal(t, []string{testConstTaskPrior, "cancelled"}, runOrder)

	// Events: task.cancelled fires; task.completed for the cancelled
	// task does NOT. (The prior task emits its own task.completed, which
	// we accept by checking for an exact match on the cancelled task's
	// completion event subject in the payload.)
	emittedTypes := recordedEventTypes(readEvents())
	assert.Contains(t, emittedTypes, "dev.zigflow.task.cancelled")

	// There is at most one task.completed (from the prior task). Crucially
	// the cancelled task must not produce a second one.
	completedCount := 0
	for _, ty := range emittedTypes {
		if ty == "dev.zigflow.task.completed" {
			completedCount++
		}
	}
	assert.Equal(t, 1, completedCount,
		"only the prior (successful) task may emit task.completed")
}

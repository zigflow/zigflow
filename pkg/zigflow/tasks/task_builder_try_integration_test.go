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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"sigs.k8s.io/yaml"
)

// The tests in this file are deliberately integration-style: they drive real
// task builders (DoTaskBuilder -> TryTaskBuilder -> nested DoTaskBuilder ->
// leaf builders) and execute the resulting workflow function in Temporal's
// test environment. The synthetic TemporalWorkflowFunc closures in
// task_builder_try_test.go prove exec() propagates an error it is handed; they
// cannot prove a *built* nested DoTaskBuilder hands one over in the first
// place, which is exactly where inline `then: end` was being swallowed.

const (
	// testConstTryTask is the try task key used by the inline try/catch
	// integration workflows.
	testConstTryTask = "guarded"
	// testConstMustNotRun is the key of the task placed after the try task; it
	// must never run once a body has signalled `then: end`.
	testConstMustNotRun = "mustNotRun"
	// testConstTaskStartedEvent is the emitted cloudevent type used to work out
	// which tasks actually executed.
	testConstTaskStartedEvent = "dev.zigflow.task.started"
	// testConstErrorKey is the default catch.as key the caught error is exposed
	// under.
	testConstErrorKey = "error"
	// testConstAfterTry is the data key set by the task that follows the try
	// task.
	testConstAfterTry = "afterTry"
	// testConstFailingTask is the key of the raise task used to fail a try body.
	testConstFailingTask = "failing"
	// testConstFinishTask is the key of the try-body task that emits `then: end`.
	testConstFinishTask = "finish"
)

// newStartedTaskRecorder wires a cloudevents.Events instance to a temp
// directory using the file protocol, and returns a reader that yields the
// subjects of the emitted task.started events in order. Because every task —
// at the root and inside the inline try/catch bodies — emits task.started
// through the same emitter, the resulting slice is an exact record of which
// tasks ran.
func newStartedTaskRecorder(
	t *testing.T, doc *model.Workflow,
) (events *cloudevents.Events, readStarted func() []string) {
	t.Helper()

	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	require.NoError(t, os.MkdirAll(eventsDir, 0o755))

	config := fmt.Sprintf("clients:\n  - name: recorder\n    protocol: file\n    target: %s\n", eventsDir)
	configPath := filepath.Join(dir, "cloudevents.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

	validator, err := utils.NewValidator()
	require.NoError(t, err)

	events, err = cloudevents.Load(configPath, validator, doc)
	require.NoError(t, err)

	readStarted = func() []string {
		entries, err := os.ReadDir(eventsDir)
		require.NoError(t, err)

		var subjects []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
			require.NoError(t, err)
			for doc := range bytes.SplitSeq(data, []byte("---\n")) {
				doc = bytes.TrimSpace(doc)
				if len(doc) == 0 {
					continue
				}
				var ev struct {
					Type    string `json:"type"`
					Subject string `json:"subject"`
				}
				require.NoError(t, yaml.Unmarshal(doc, &ev))
				if ev.Type == testConstTaskStartedEvent {
					subjects = append(subjects, ev.Subject)
				}
			}
		}
		return subjects
	}

	return events, readStarted
}

// newTestWorkflowDoc returns a workflow document whose name namespaces the
// per-task activity aliases the builders register.
func newTestWorkflowDoc(name string) *model.Workflow {
	return &model.Workflow{Document: model.Document{Name: name, DSL: "1.0.0"}}
}

// newRegistryMock returns a worker mock that accepts the root workflow
// registration Build() performs. Tests that assert on activity aliases layer
// their own strict expectations on top.
func newRegistryMock() *WorkflowRegistryMock {
	w := new(WorkflowRegistryMock)
	w.On("RegisterWorkflowWithOptions", mock.Anything, mock.Anything).Maybe()
	return w
}

// newPermissiveRegistryMock also accepts any activity registration, for tests
// that assert on runtime dispatch rather than on the registered alias names.
func newPermissiveRegistryMock() *WorkflowRegistryMock {
	w := newRegistryMock()
	w.On("RegisterActivityWithOptions", mock.Anything, mock.Anything).Maybe()
	return w
}

// buildRootWorkflow runs the real load-time and build-time pipeline over do
// and returns the root workflow function, exactly as NewWorkflow would.
func buildRootWorkflow(
	t *testing.T,
	w worker.Worker,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	do *model.TaskList,
) TemporalWorkflowFunc {
	t.Helper()

	b, err := NewDoTaskBuilder(w, &model.DoTask{Do: do}, doc.Document.Name, doc, emitter, nil)
	require.NoError(t, err)
	require.NoError(t, b.PostLoad())
	require.NoError(t, b.Validate())

	fn, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, fn)

	return fn
}

// runRootWorkflow executes fn as the root workflow (state nil, so the real
// root-boundary code path runs) and returns its result and error.
func runRootWorkflow(
	t *testing.T,
	name string,
	fn TemporalWorkflowFunc,
	registerActivities func(env *testsuite.TestWorkflowEnvironment),
) (any, error) {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	if registerActivities != nil {
		registerActivities(env)
	}

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, nil)
	}, workflow.RegisterOptions{Name: name})

	env.ExecuteWorkflow(name)
	require.True(t, env.IsWorkflowCompleted())

	if err := env.GetWorkflowError(); err != nil {
		return nil, err
	}

	var output any
	require.NoError(t, env.GetWorkflowResult(&output))

	return output, nil
}

// newEndingSetTask returns a set task carrying a `then: end` directive.
func newEndingSetTask(set map[string]any) *model.SetTask {
	return &model.SetTask{
		TaskBase: model.TaskBase{
			Then: &model.FlowDirective{Value: string(model.FlowDirectiveEnd)},
		},
		Set: model.NewObjectOrRuntimeExpr(set),
	}
}

// newSetTask returns a plain set task with no flow directive.
func newSetTask(set map[string]any) *model.SetTask {
	return &model.SetTask{Set: model.NewObjectOrRuntimeExpr(set)}
}

// newFailingRaiseTask returns a raise task that always fails the surrounding
// scope, which is how these tests drive the try body into the catch body
// without depending on an external service.
func newFailingRaiseTask() *model.RaiseTask {
	return &model.RaiseTask{
		Raise: model.RaiseTaskConfiguration{
			Error: model.RaiseTaskError{
				Definition: &model.Error{
					Type:   model.NewUriTemplate(model.ErrorTypeValidation),
					Status: 400,
					Title:  model.NewStringOrRuntimeExpr("Boom"),
					Detail: model.NewStringOrRuntimeExpr("try body failed"),
				},
			},
		},
	}
}

// TestBuiltTryBodyEndTerminatesWholeWorkflow is the regression test for the
// swallowed `then: end`. The try body is a real, built nested DoTaskBuilder
// running inline in the root workflow's Temporal context, so
// isRootWorkflow(ctx) reports true for it. Before the explicit nested marker
// the nested executor consumed flow.ErrEnd as a clean root completion, exec()
// saw no error, and both the catch body and every task after the try kept
// running.
//
// The DSL equivalent:
//
//	do:
//	  - guarded:
//	      try:
//	        - finish:
//	            set:
//	              result: ended
//	            then: end
//	      catch:
//	        as: error
//	        do:
//	          - caught:
//	              set:
//	                catchRan: true
//	  - mustNotRun:
//	      set:
//	        afterTry: true
func TestBuiltTryBodyEndTerminatesWholeWorkflow(t *testing.T) {
	doc := newTestWorkflowDoc("wf-inline-try-end")
	events, readStarted := newStartedTaskRecorder(t, doc)

	do := &model.TaskList{
		&model.TaskItem{Key: testConstTryTask, Task: &model.TryTask{
			Try: &model.TaskList{
				&model.TaskItem{Key: testConstFinishTask, Task: newEndingSetTask(map[string]any{
					testConstResult: "ended",
				})},
			},
			Catch: &model.TryTaskCatch{
				As: testConstErrorKey,
				Do: &model.TaskList{
					&model.TaskItem{Key: "caught", Task: newSetTask(map[string]any{"catchRan": true})},
				},
			},
		}},
		&model.TaskItem{Key: testConstMustNotRun, Task: newSetTask(map[string]any{testConstAfterTry: true})},
	}

	fn := buildRootWorkflow(t, newRegistryMock(), doc, events, do)
	output, err := runRootWorkflow(t, doc.Document.Name, fn, nil)

	// At the true root boundary the propagated end is a clean completion.
	require.NoError(t, err)

	// The ending task's output survives: the nested body shares the root
	// state, so state.Output still holds what `finish` set, and the try task
	// (a control-directive task with no output directive of its own) leaves it
	// untouched.
	assert.Equal(t, map[string]any{testConstResult: "ended"}, output)

	// Only the try task and the ending task inside it ran.
	assert.Equal(t, []string{testConstTryTask, testConstFinishTask}, readStarted())
}

// TestBuiltCatchBodyEndTerminatesWholeWorkflow is the symmetric case: the try
// body fails for a real reason, the catch body runs, and a real task inside
// the catch body emits `then: end`. The end must terminate the whole workflow
// and the task after the try must not run.
//
// The try task carries an output directive so the test can also prove the
// carried output of the ended catch body reaches the enclosing task rather
// than being discarded at the boundary.
func TestBuiltCatchBodyEndTerminatesWholeWorkflow(t *testing.T) {
	doc := newTestWorkflowDoc("wf-inline-catch-end")
	events, readStarted := newStartedTaskRecorder(t, doc)

	do := &model.TaskList{
		&model.TaskItem{Key: testConstTryTask, Task: &model.TryTask{
			TaskBase: model.TaskBase{
				Output: &model.Output{
					As: model.NewObjectOrRuntimeExpr(map[string]any{"carried": "${ . }"}),
				},
			},
			Try: &model.TaskList{
				&model.TaskItem{Key: testConstFailingTask, Task: newFailingRaiseTask()},
			},
			Catch: &model.TryTaskCatch{
				As: testConstErrorKey,
				Do: &model.TaskList{
					&model.TaskItem{Key: testConstHandledKey, Task: newEndingSetTask(map[string]any{
						testConstHandledKey: true,
					})},
				},
			},
		}},
		&model.TaskItem{Key: testConstMustNotRun, Task: newSetTask(map[string]any{testConstAfterTry: true})},
	}

	fn := buildRootWorkflow(t, newRegistryMock(), doc, events, do)
	output, err := runRootWorkflow(t, doc.Document.Name, fn, nil)

	require.NoError(t, err)
	assert.Equal(t,
		map[string]any{"carried": map[string]any{testConstHandledKey: true}},
		output,
		"the ended catch body's output must reach the try task's output directive")

	// The failing try task ran, the catch handler ran, and nothing after the
	// try task ran.
	assert.Equal(t,
		[]string{testConstTryTask, testConstFailingTask, testConstHandledKey},
		readStarted())
}

// TestBuiltTryCatchOrdinaryFailureRegression guards the behaviour the end
// propagation must not disturb: an ordinary try-body failure still enters the
// catch body, the catch output becomes the try task's output, and tasks after
// the try task keep running when the catch completes normally.
func TestBuiltTryCatchOrdinaryFailureRegression(t *testing.T) {
	newTryTask := func() *model.TryTask {
		return &model.TryTask{
			Try: &model.TaskList{
				&model.TaskItem{Key: testConstFailingTask, Task: newFailingRaiseTask()},
			},
			Catch: &model.TryTaskCatch{
				As: testConstErrorKey,
				Do: &model.TaskList{
					&model.TaskItem{Key: testConstHandledKey, Task: newSetTask(map[string]any{
						testConstHandledKey: true,
					})},
				},
			},
		}
	}

	t.Run("catch output is returned", func(t *testing.T) {
		doc := newTestWorkflowDoc("wf-inline-catch-output")
		events, readStarted := newStartedTaskRecorder(t, doc)

		do := &model.TaskList{
			&model.TaskItem{Key: testConstTryTask, Task: newTryTask()},
		}

		fn := buildRootWorkflow(t, newRegistryMock(), doc, events, do)
		output, err := runRootWorkflow(t, doc.Document.Name, fn, nil)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{testConstHandledKey: true}, output)
		assert.Equal(t,
			[]string{testConstTryTask, testConstFailingTask, testConstHandledKey},
			readStarted())
	})

	t.Run("subsequent tasks run", func(t *testing.T) {
		doc := newTestWorkflowDoc("wf-inline-catch-continues")
		events, readStarted := newStartedTaskRecorder(t, doc)

		do := &model.TaskList{
			&model.TaskItem{Key: testConstTryTask, Task: newTryTask()},
			&model.TaskItem{Key: "after", Task: newSetTask(map[string]any{testConstAfterTry: true})},
		}

		fn := buildRootWorkflow(t, newRegistryMock(), doc, events, do)
		output, err := runRootWorkflow(t, doc.Document.Name, fn, nil)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{testConstAfterTry: true}, output)
		assert.Equal(t,
			[]string{testConstTryTask, testConstFailingTask, testConstHandledKey, "after"},
			readStarted())
	})
}

// newCollisionTaskList builds a workflow with the same activity-backed task
// key ("step") in three places: outside the try task, inside the try body and
// inside the catch body. Each location must resolve to its own per-task
// activity alias.
func newCollisionTaskList() *model.TaskList {
	return &model.TaskList{
		&model.TaskItem{Key: testConstStep, Task: newTestHTTPTask()},
		&model.TaskItem{Key: testConstTryTask, Task: &model.TryTask{
			Try: &model.TaskList{
				&model.TaskItem{Key: testConstStep, Task: newTestHTTPTask()},
			},
			Catch: &model.TryTaskCatch{
				As: testConstErrorKey,
				Do: &model.TaskList{
					&model.TaskItem{Key: testConstStep, Task: newTestHTTPTask()},
				},
			},
		}},
	}
}

// TestBuiltTryCatchRegistersDistinctActivityAliases proves the inline try and
// catch bodies keep the parent try task's path plus their own branch segment.
// Before the fix both bodies were built with an empty task path, so all three
// "step" tasks collapsed onto one alias: registerActivityOnce would dedup the
// second and third registrations and every location would dispatch to
// whichever implementation registered first.
func TestBuiltTryCatchRegistersDistinctActivityAliases(t *testing.T) {
	doc := newTestWorkflowDoc("wf-alias")

	w := newRegistryMock()
	for _, name := range []string{
		"wf-alias.step",
		"wf-alias.guarded.try.step",
		"wf-alias.guarded.catch.step",
	} {
		w.
			On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{Name: name}).
			Once()
	}

	buildRootWorkflow(t, w, doc, testEvents, newCollisionTaskList())

	w.AssertExpectations(t)
}

// TestBuiltTryCatchInvokesBranchSpecificActivity closes the loop on the alias
// fix by actually executing the workflow with three distinct implementations
// registered under the three aliases. Each branch must invoke its own
// implementation and no other.
func TestBuiltTryCatchInvokesBranchSpecificActivity(t *testing.T) {
	const (
		outerAlias = "wf-dispatch.step"
		tryAlias   = "wf-dispatch.guarded.try.step"
		catchAlias = "wf-dispatch.guarded.catch.step"
	)

	cases := []struct {
		name        string
		tryFails    bool
		wantInvoked []string
		wantOutput  any
	}{
		{
			name:        "try branch",
			tryFails:    false,
			wantInvoked: []string{outerAlias, tryAlias},
			wantOutput:  tryAlias,
		},
		{
			name:        "catch branch",
			tryFails:    true,
			wantInvoked: []string{outerAlias, tryAlias, catchAlias},
			wantOutput:  catchAlias,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := newTestWorkflowDoc("wf-dispatch")

			var (
				mu       sync.Mutex
				invoked  []string
				recorder = func(alias string, fail bool) any {
					return func(_ context.Context, _, _, _ any) (any, error) {
						mu.Lock()
						invoked = append(invoked, alias)
						mu.Unlock()
						if fail {
							return nil, temporal.NewNonRetryableApplicationError(
								"activity failed", "Boom", nil,
							)
						}
						return alias, nil
					}
				}
			)

			fn := buildRootWorkflow(t, newPermissiveRegistryMock(), doc, testEvents, newCollisionTaskList())

			output, err := runRootWorkflow(t, doc.Document.Name, fn, func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterActivityWithOptions(recorder(outerAlias, false),
					activity.RegisterOptions{Name: outerAlias})
				env.RegisterActivityWithOptions(recorder(tryAlias, tc.tryFails),
					activity.RegisterOptions{Name: tryAlias})
				env.RegisterActivityWithOptions(recorder(catchAlias, false),
					activity.RegisterOptions{Name: catchAlias})
			})

			require.NoError(t, err)
			assert.Equal(t, tc.wantOutput, output)

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.wantInvoked, invoked,
				"each task location must dispatch to its own per-task activity alias")
		})
	}
}

// TestNestedDoTaskBuilderNeverContinuesAsNew guards the other
// workflow-boundary behaviour an inline body must not perform. A nested body
// has no registered workflow type of its own (its name is empty), so calling
// continueAsNew from inside it would ask Temporal to continue as an unnamed
// workflow. Continue-As-New belongs to the enclosing root workflow, which
// takes the decision between its own tasks.
func TestNestedDoTaskBuilderNeverContinuesAsNew(t *testing.T) {
	nested := newTestDoTaskBuilder("", DoTaskOpts{
		DisableRegisterWorkflow: true,
		Nested:                  true,
	})
	root := newTestDoTaskBuilder("root")

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	// Temporal suggesting Continue-As-New is the condition both builders see;
	// only the non-nested one may act on it.
	env.SetContinueAsNewSuggested(true)

	var nestedWantsCAN, rootWantsCAN bool
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		nestedWantsCAN = nested.shouldContinueAsNew(ctx)
		rootWantsCAN = root.shouldContinueAsNew(ctx)
		return nil, nil
	}, workflow.RegisterOptions{Name: "nested-can"})

	env.ExecuteWorkflow("nested-can")
	require.NoError(t, env.GetWorkflowError())

	assert.False(t, nestedWantsCAN, "a nested inline task list must never initiate Continue-As-New")
	assert.True(t, rootWantsCAN, "the non-nested builder must still honour the history-length override")
}

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

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestRunTaskBuilderValidateNeedsPostLoadAwaitDefault confirms the
// PostLoad → Validate ordering. Run.Await is a *bool; Validate's script
// branch contains `if !*t.task.Run.Await {...}`, so the pointer must be
// non-nil at that point. PostLoad's job is to default a nil Await to
// true. If the order were inverted, Validate would nil-panic here.
//
// The "without PostLoad" subtest is a tripwire: it verifies that
// Validate currently DOES nil-panic on nil Await. If that ever stops
// being true (e.g. Validate becomes nil-tolerant), the positive
// subtest is no longer testing whether the order is correct, and both
// halves must be re-examined together.
func TestRunTaskBuilderValidateNeedsPostLoadAwaitDefault(t *testing.T) {
	makeTask := func() *model.RunTask {
		return &model.RunTask{
			Run: model.RunTaskConfiguration{
				Await: nil, // explicit: PostLoad must populate this before Validate
				Script: &model.Script{
					Language:   constScriptLanguagePython,
					InlineCode: utils.Ptr("print(1)"),
				},
			},
		}
	}

	t.Run("PostLoad then Validate succeeds", func(t *testing.T) {
		task := makeTask()
		builder, err := NewRunTaskBuilder(nil, task, "ordering", nil, testEvents, nil)
		require.NoError(t, err)
		require.NoError(t, builder.PostLoad())
		require.NotNil(t, task.Run.Await, "PostLoad must set Await before Validate runs")
		assert.NoError(t, builder.Validate(), "Validate must not panic on Await deref once PostLoad has run")
	})

	t.Run("Validate without PostLoad nil-panics (tripwire)", func(t *testing.T) {
		task := makeTask()
		builder, err := NewRunTaskBuilder(nil, task, "ordering", nil, testEvents, nil)
		require.NoError(t, err)
		assert.Panics(t, func() {
			_ = builder.Validate()
		}, "Validate must currently nil-panic on nil Await; if this stops being true the test is no longer testing if the order is correct")
	})
}

func TestRunTaskBuilderPostLoadSetsAwaitDefault(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Workflow: &model.RunWorkflow{
				Namespace: constDefaultNamespace,
				Name:      "child-runner",
				Version:   testConstRunWorkflowVersion,
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)

	// PostLoad must run before Build in production; tests must reflect the same order.
	assert.NoError(t, builder.PostLoad())

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
						Namespace: constDefaultNamespace,
						Name:      "child-runner",
						Version:   testConstRunWorkflowVersion,
					},
				},
			}

			builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
			assert.NoError(t, err)

			fn, err := builder.Build()
			assert.NoError(t, err)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				return map[string]any{
					testConstChild: testConstDone,
				}, nil
			}, workflow.RegisterOptions{Name: task.Run.Workflow.Name})

			state := utils.NewState()

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				return fn(ctx, map[string]any{testConstRequest: testConstData}, state)
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
				assert.Equal(t, map[string]any{testConstChild: testConstDone}, result)
			}

			val, ok := state.Data["run-task"]
			assert.True(t, ok)
			if tc.expectNilResp {
				assert.Nil(t, val)
			} else {
				assert.Equal(t, map[string]any{testConstChild: testConstDone}, val)
			}
		})
	}
}

// TestRunTaskBuilderRunWorkflowPropagatesEndFromChild proves that a
// `run.workflow` task whose child workflow signalled `then: end` does
// NOT wrap that signal as a child-workflow failure. Instead, runWorkflow
// must surface flow.ErrEnd carrying the child's effective output so the
// surrounding do-task pipeline keeps propagating end upward toward the
// root workflow with the right payload.
func TestRunTaskBuilderRunWorkflowPropagatesEndFromChild(t *testing.T) {
	const childWorkflowName = "child-end-runner"
	childOutput := map[string]any{testConstChild: testConstDone}

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Await: utils.Ptr(true),
			Workflow: &model.RunWorkflow{
				Namespace: constDefaultNamespace,
				Name:      childWorkflowName,
				Version:   testConstRunWorkflowVersion,
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-end-task", nil, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	// The child workflow ends via Zigflow's typed end ApplicationError.
	// It also returns a non-nil result so that the result and the error
	// are both meaningful values; Temporal surfaces only the error to
	// the parent's future.Get, and runWorkflow must reconstruct the
	// carried output from the EndPayload, not from the discarded result.
	env.RegisterWorkflowWithOptions(func(_ workflow.Context, _ any, _ *utils.State) (map[string]any, error) {
		return childOutput, flow.NewEndApplicationError(childOutput)
	}, workflow.RegisterOptions{Name: childWorkflowName})

	captured := struct {
		output any
		err    error
	}{}

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		output, runErr := fn(ctx, map[string]any{testConstRequest: testConstData}, utils.NewState())
		captured.output = output
		captured.err = runErr
		return nil
	}, workflow.RegisterOptions{Name: "run-end-host"})

	env.ExecuteWorkflow("run-end-host")
	require.NoError(t, env.GetWorkflowError())

	// runWorkflow must report the end via flow.ErrEnd (not the wrapped
	// "error executiing child workflow" string), and the child's
	// effective output must have crossed the boundary intact.
	require.Error(t, captured.err)
	assert.ErrorIs(t, captured.err, flow.ErrEnd)
	assert.NotContains(t, captured.err.Error(), "error executiing child workflow",
		"a child-emitted end must not be wrapped as a child-workflow failure")
	assert.Equal(t, childOutput, captured.output,
		"the child workflow's end payload output must propagate through runWorkflow")
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
						Language: constScriptLanguagePython,
					},
				},
			},
			assertErr: "run script has no inline or external code defined: script-task",
		},
		{
			name: "await disabled",
			task: &model.RunTask{
				Run: model.RunTaskConfiguration{
					Await: utils.Ptr(false),
					Script: &model.Script{
						Language:   constScriptLanguagePython,
						InlineCode: utils.Ptr(inline),
					},
				},
			},
			assertErr: "run scripts must be run with await: script-task",
		},
		{
			name: "both inline code and external source set",
			task: &model.RunTask{
				Run: model.RunTaskConfiguration{
					Script: &model.Script{
						Language:   constScriptLanguagePython,
						InlineCode: utils.Ptr(inline),
						External: &model.ExternalResource{
							Endpoint: model.NewEndpoint("file:///scripts/run.py"),
						},
					},
				},
			},
			assertErr: "run script must not set both inline code and external source: script-task",
		},
		{
			name: "external source with nil endpoint",
			task: &model.RunTask{
				Run: model.RunTaskConfiguration{
					Script: &model.Script{
						Language: constScriptLanguagePython,
						External: &model.ExternalResource{
							// Endpoint intentionally nil
						},
					},
				},
			},
			assertErr: "run script external source has no endpoint: script-task",
		},
	}

	// Verify that Build succeeds when an external source is provided and
	// inline code is absent — the validation path must accept this as valid.
	t.Run("valid when external source set and inline code absent", func(t *testing.T) {
		t.Parallel()
		task := &model.RunTask{
			Run: model.RunTaskConfiguration{
				Script: &model.Script{
					Language: constScriptLanguagePython,
					External: &model.ExternalResource{
						Endpoint: model.NewEndpoint("file:///scripts/run.py"),
					},
				},
			},
		}
		builder, err := NewRunTaskBuilder(nil, task, "script-task", nil, testEvents, nil)
		assert.NoError(t, err)
		assert.NoError(t, builder.PostLoad())
		fn, err := builder.Build()
		assert.NoError(t, err)
		assert.NotNil(t, fn)
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder, err := NewRunTaskBuilder(nil, tc.task, "script-task", nil, testEvents, nil)
			assert.NoError(t, err)

			// PostLoad must precede Validate in the production lifecycle so Validate
			// can dereference Await safely.
			assert.NoError(t, builder.PostLoad())

			err = builder.Validate()
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
				Language:   constScriptLanguagePython,
				InlineCode: utils.Ptr("print('hello')"),
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "script-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	// Per-task registration: Build() dispatches by per-task name; the
	// testsuite needs the activity registered under that name to resolve.
	env.RegisterActivityWithOptions(runActivities.CallScriptActivity, activity.RegisterOptions{Name: "script-task"})
	env.OnActivity(
		"script-task",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return("script-success", nil).Once()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, map[string]any{testConstRequest: testConstData}, state)
	}, workflow.RegisterOptions{Name: "script-run"})

	env.ExecuteWorkflow("script-run")
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "script-success", result)

	assert.Equal(t, "script-success", state.Data["script-task"])
}

func TestRunTaskBuilderPostLoadSetsAwaitToTrueWhenNil(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Shell: &model.Shell{Command: testConstEcho},
			// Await intentionally omitted
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.NotNil(t, task.Run.Await, "Await must not be nil after PostLoad")
	assert.True(t, *task.Run.Await, "Await must default to true")
}

func TestRunTaskBuilderPostLoadPreservesExplicitAwait(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Shell: &model.Shell{Command: testConstEcho},
			Await: utils.Ptr(false),
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.False(t, *task.Run.Await, "explicit Await=false must not be overwritten")
}

func TestRunTaskBuilderPostLoadSetsContainerLifetimeDefault(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image: testConstAlpineImage,
				// Lifetime intentionally omitted
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.NotNil(t, task.Run.Container.Lifetime, "Lifetime must not be nil after PostLoad")
	assert.Equal(t, "always", task.Run.Container.Lifetime.Cleanup)
}

func TestRunTaskBuilderPostLoadPreservesExplicitContainerLifetime(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image:    testConstAlpineImage,
				Lifetime: &model.ContainerLifetime{Cleanup: "never"},
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.Equal(t, "never", task.Run.Container.Lifetime.Cleanup, "explicit Cleanup must not be overwritten")
}

func TestRunTaskBuilderPostLoadPreservesExplicitNamespaceAndVersion(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Workflow: &model.RunWorkflow{
				Name:      "child-workflow",
				Namespace: "production",
				Version:   "2.3.0",
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.Equal(t, "production", task.Run.Workflow.Namespace, "explicit namespace must not be overwritten")
	assert.Equal(t, "2.3.0", task.Run.Workflow.Version, "explicit version must not be overwritten")
}

func TestRunTaskBuilderPostLoadDefaultsEmptyNamespaceAndVersion(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Workflow: &model.RunWorkflow{
				Name: "child-workflow",
				// Namespace and Version intentionally omitted
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.Equal(t, constDefaultNamespace, task.Run.Workflow.Namespace, "empty namespace should receive default")
	assert.Equal(t, "0.0.1", task.Run.Workflow.Version, "empty version should receive default")
}

func TestRunTaskBuilderPostLoadNilWorkflowIsNoop(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Shell: &model.Shell{Command: testConstEcho},
			// Workflow is nil — PostLoad must not panic
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "run-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())
}

// TestRunTaskBuilderRunContainerPassesRuntimeOptions verifies that
// runContainer forwards the namespace, runtime and service account from the
// supplied TaskOpts into the container activity. These are positional args
// after (ctx, task, input, state).
func TestRunTaskBuilderRunContainerPassesRuntimeOptions(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	runActivities := &activities.Run{}

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image: testConstAlpineImage,
			},
		},
	}

	taskOpts := &TaskOpts{
		Run: &RunTaskOpts{
			Namespace:      "workflows-ns",
			Runtime:        activities.ContainerRuntimeKubernetes,
			ServiceAccount: "workflows-sa",
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "container-task", nil, testEvents, taskOpts)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()

	env.RegisterActivityWithOptions(runActivities.CallContainerActivity, activity.RegisterOptions{Name: "container-task"})
	// Matchers: ctx, task, input, state, namespace, runtime, serviceAccount.
	// The runtime arg is the strongly typed activities.ContainerRuntime, so the
	// matcher must use the same type rather than the raw string value.
	env.OnActivity(
		"container-task",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		"workflows-ns",
		activities.ContainerRuntimeKubernetes,
		"workflows-sa",
	).Return("container-success", nil).Once()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, map[string]any{testConstRequest: testConstData}, state)
	}, workflow.RegisterOptions{Name: "container-run"})

	env.ExecuteWorkflow("container-run")
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "container-success", result)
	assert.Equal(t, "container-success", state.Data["container-task"])
}

// TestRunTaskBuilderRunContainerPassesEmptyRuntimeOptionsWhenTaskOptsNil
// pins the behaviour that a nil TaskOpts produces empty strings for runtime,
// namespace and service account. The activity receives the args either way,
// so this prevents a silent regression where a nil opts crashes the builder
// or skips the args entirely.
func TestRunTaskBuilderRunContainerPassesEmptyRuntimeOptionsWhenTaskOptsNil(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	runActivities := &activities.Run{}

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image: testConstAlpineImage,
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "container-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()

	env.RegisterActivityWithOptions(runActivities.CallContainerActivity, activity.RegisterOptions{Name: "container-task"})
	env.OnActivity(
		"container-task",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		"",
		activities.ContainerRuntime(""),
		"",
	).Return("container-success", nil).Once()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, map[string]any{testConstRequest: testConstData}, state)
	}, workflow.RegisterOptions{Name: "container-run-nil-opts"})

	env.ExecuteWorkflow("container-run-nil-opts")
	assert.NoError(t, env.GetWorkflowError())
}

func TestRunTaskBuilderRunShellExecutesActivity(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	runActivities := &activities.Run{}

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Shell: &model.Shell{
				Command: testConstEcho,
			},
		},
	}

	builder, err := NewRunTaskBuilder(nil, task, "shell-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()
	env.RegisterActivityWithOptions(runActivities.CallShellActivity, activity.RegisterOptions{Name: "shell-task"})
	env.OnActivity(
		"shell-task",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return("shell-success", nil).Once()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, map[string]any{testConstRequest: testConstData}, state)
	}, workflow.RegisterOptions{Name: "shell-run"})

	env.ExecuteWorkflow("shell-run")
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "shell-success", result)

	assert.Equal(t, "shell-success", state.Data["shell-task"])
}

func TestRunTaskBuilderRegistersPerTaskActivityName(t *testing.T) {
	cases := []struct {
		name        string
		makeTask    func() *model.RunTask
		taskName    string
		expectedReg string
	}{
		{
			name: "container variant",
			makeTask: func() *model.RunTask {
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Await:     utils.Ptr(true),
						Container: &model.Container{Image: "busybox:latest"},
					},
				}
			},
			taskName:    "runContainer",
			expectedReg: "wf-run.runContainer",
		},
		{
			name: "script variant",
			makeTask: func() *model.RunTask {
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Await: utils.Ptr(true),
						Script: &model.Script{
							Language:   constScriptLanguagePython,
							InlineCode: utils.Ptr("print(1)"),
						},
					},
				}
			},
			taskName:    "runScript",
			expectedReg: "wf-run.runScript",
		},
		{
			name: "shell variant",
			makeTask: func() *model.RunTask {
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Await: utils.Ptr(true),
						Shell: &model.Shell{Command: "echo hello"},
					},
				}
			},
			taskName:    "runShell",
			expectedReg: "wf-run.runShell",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := &model.Workflow{Document: model.Document{Name: "wf-run"}}

			w := new(WorkflowRegistryMock)
			w.
				On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
					Name: tc.expectedReg,
				}).
				Once()

			b, err := NewRunTaskBuilder(w, tc.makeTask(), tc.taskName, doc, testEvents, nil)
			require.NoError(t, err)
			require.NoError(t, b.PostLoad())

			_, err = b.Build()
			assert.NoError(t, err)
			w.AssertExpectations(t)
		})
	}
}

func TestRunTaskBuilderWorkflowVariantDoesNotRegisterActivity(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-run-child"}}
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Await:    utils.Ptr(true),
			Workflow: &model.RunWorkflow{Namespace: constDefaultNamespace, Name: "child", Version: testConstRunWorkflowVersion},
		},
	}

	w := new(WorkflowRegistryMock)
	// no .On("RegisterActivityWithOptions", ...): runWorkflow uses a child workflow, not an activity

	b, err := NewRunTaskBuilder(w, task, "runWf", doc, testEvents, nil)
	require.NoError(t, err)
	require.NoError(t, b.PostLoad())

	_, err = b.Build()
	assert.NoError(t, err)
	w.AssertExpectations(t)
}

func TestRunTaskBuilderBuildWithoutWorker(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-run-nil-worker"}}
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Await: utils.Ptr(true),
			Shell: &model.Shell{Command: "echo hello"},
		},
	}

	b, err := NewRunTaskBuilder(nil, task, "step", doc, testEvents, nil)
	require.NoError(t, err)
	require.NoError(t, b.PostLoad())

	fn, err := b.Build()
	assert.NoError(t, err)
	assert.NotNil(t, fn)
}

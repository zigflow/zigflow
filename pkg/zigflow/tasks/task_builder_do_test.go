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
	"fmt"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestDoTaskBuilderWorkflowExecutor(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		initialState *utils.State
		expectedEnv  map[string]any
	}{
		{
			name:         "initialises new state when not provided",
			initialState: nil,
			expectedEnv: map[string]any{
				"APP_ENV": "test",
			},
		},
		{
			name: "reuses provided state without overriding env",
			initialState: func() *utils.State {
				s := utils.NewState()
				s.Env["from"] = "caller"
				return s
			}(),
			expectedEnv: map[string]any{
				"from": "caller",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			builder := &DoTaskBuilder{
				builder: builder[*model.DoTask]{
					doc:          testWorkflow,
					eventEmitter: testEvents,
					name:         "test-workflow",
					task:         &model.DoTask{},
				},
				opts: DoTaskOpts{
					Envvars: map[string]any{
						"APP_ENV": "test",
					},
				},
			}

			var capturedState *utils.State
			runOrder := make([]string, 0, 2)
			expectedOutput := map[string]any{
				"value": "two",
			}

			tasks := newOutputWorkflowFuncs(&runOrder, &capturedState)
			wf := builder.workflowExecutor(tasks)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			inputPayload := map[string]any{
				"request_id": tc.name,
			}
			workflowName := "workflow-" + tc.name

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				result, err := wf(ctx, inputPayload, tc.initialState)
				return result, err
			}, workflow.RegisterOptions{Name: workflowName})

			env.ExecuteWorkflow(workflowName)

			assert.NoError(t, env.GetWorkflowError())

			var workflowResult map[string]any
			assert.NoError(t, env.GetWorkflowResult(&workflowResult))
			assert.Equal(t, expectedOutput, workflowResult)

			assert.Equal(t, []string{"task-one", "task-two"}, runOrder)
			assert.NotNil(t, capturedState)
			if tc.initialState != nil {
				assert.Same(t, tc.initialState, capturedState)
			} else {
				assert.Equal(t, inputPayload, capturedState.Input)
			}
			assert.Equal(t, tc.expectedEnv, capturedState.Env)
			assert.Equal(t, expectedOutput, capturedState.Output)
		})
	}
}

func TestDoTaskBuilderIterateTasksFlowControl(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		setup       func(runOrder *[]string) []workflowFunc
		expectedRun []string
		expectErr   string
	}{
		{
			name: "non-enum flow directive jumps to named task",
			setup: func(runOrder *[]string) []workflowFunc {
				return []workflowFunc{
					newSimpleWorkflowFunc("task-a", &model.TaskBase{
						Then: &model.FlowDirective{
							Value: "task-c",
						},
					}, runOrder),
					newSimpleWorkflowFunc("task-b", &model.TaskBase{}, runOrder),
					newSimpleWorkflowFunc("task-c", &model.TaskBase{}, runOrder),
				}
			},
			expectedRun: []string{"task-a", "task-c"},
		},
		{
			name: "missing target returns descriptive error",
			setup: func(runOrder *[]string) []workflowFunc {
				return []workflowFunc{
					newSimpleWorkflowFunc("task-a", &model.TaskBase{
						Then: &model.FlowDirective{
							Value: "task-c",
						},
					}, runOrder),
					newSimpleWorkflowFunc("task-b", &model.TaskBase{}, runOrder),
				}
			},
			expectedRun: []string{"task-a"},
			expectErr:   "next target specified but not found: task-c",
		},
		{
			name: "termination directive stops iteration",
			setup: func(runOrder *[]string) []workflowFunc {
				return []workflowFunc{
					newSimpleWorkflowFunc("task-end", &model.TaskBase{
						Then: &model.FlowDirective{
							Value: string(model.FlowDirectiveEnd),
						},
					}, runOrder),
					newSimpleWorkflowFunc("task-b", &model.TaskBase{}, runOrder),
				}
			},
			expectedRun: []string{"task-end"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			builder := &DoTaskBuilder{
				builder: builder[*model.DoTask]{
					doc:          testWorkflow,
					eventEmitter: testEvents,
					name:         "iterate-workflow",
					task:         &model.DoTask{},
				},
			}

			runOrder := make([]string, 0)
			tasks := tc.setup(&runOrder)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()
			workflowName := "iterate-" + tc.name

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				err := builder.iterateTasks(ctx, tasks, nil, utils.NewState())
				return nil, err
			}, workflow.RegisterOptions{Name: workflowName})

			env.ExecuteWorkflow(workflowName)

			err := env.GetWorkflowError()
			if tc.expectErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedRun, runOrder)
		})
	}
}

func TestDoTaskBuilderShouldContinueAsNew(t *testing.T) {
	t.Helper()

	tests := []struct {
		name          string
		opts          DoTaskOpts
		historyLength int
		suggested     bool
		expectResult  bool
	}{
		{
			name:          "no suggestion and no override stays in same run",
			historyLength: 5,
			expectResult:  false,
		},
		{
			name:          "temporal suggestion forces continue-as-new",
			suggested:     true,
			historyLength: 3,
			expectResult:  true,
		},
		{
			name: "custom history limit forces continue-as-new",
			opts: DoTaskOpts{
				MaxHistoryLength: 10,
			},
			historyLength: 11,
			expectResult:  true,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			builder := newTestDoTaskBuilder(fmt.Sprintf("should-continue-%d", i), tc.opts)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()
			env.SetCurrentHistoryLength(tc.historyLength)
			env.SetContinueAsNewSuggested(tc.suggested)

			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
				return builder.shouldContinueAsNew(ctx), nil
			}, workflow.RegisterOptions{Name: builder.GetTaskName()})

			env.ExecuteWorkflow(builder.GetTaskName())

			assert.NoError(t, env.GetWorkflowError())

			var result bool
			assert.NoError(t, env.GetWorkflowResult(&result))
			assert.Equal(t, tc.expectResult, result)
		})
	}
}

func TestDoTaskBuilderContinueAsNew(t *testing.T) {
	t.Helper()

	builder := newTestDoTaskBuilder("continue-as-new")
	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.continueAsNew(ctx, builder.GetTaskName(), "task-one-0", map[string]any{"request_id": "123"}, state)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	err := env.GetWorkflowError()
	assert.Error(t, err)
	assert.True(t, workflow.IsContinueAsNewError(err))
	if assert.NotNil(t, state.CANStartFrom) {
		assert.Equal(t, "task-one-0", *state.CANStartFrom)
	}
}

func TestDoTaskBuilderIterateTasksContinueAsNew(t *testing.T) {
	t.Helper()

	builder := newTestDoTaskBuilder("iterate-continue-as-new")

	runOrder := make([]string, 0)
	state := utils.NewState()
	tasks := []workflowFunc{
		newSimpleWorkflowFunc("task-one", &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetContinueAsNewSuggested(true)
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	err := env.GetWorkflowError()
	assert.Error(t, err)
	assert.True(t, workflow.IsContinueAsNewError(err))
	assert.Empty(t, runOrder)
	if assert.NotNil(t, state.CANStartFrom) {
		assert.Equal(t, "task-one-0", *state.CANStartFrom)
	}
}

func TestDoTaskBuilderIterateTasksSkipsCompletedTasks(t *testing.T) {
	t.Helper()

	builder := newTestDoTaskBuilder("iterate-skip")

	runOrder := make([]string, 0, 2)
	state := utils.NewState()
	state.CANStartFrom = utils.Ptr("task-two-1")

	tasks := []workflowFunc{
		newSimpleWorkflowFunc("task-one", &model.TaskBase{}, &runOrder),
		newSimpleWorkflowFunc("task-two", &model.TaskBase{}, &runOrder),
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return nil, builder.iterateTasks(ctx, tasks, nil, state)
	}, workflow.RegisterOptions{Name: builder.GetTaskName()})

	env.ExecuteWorkflow(builder.GetTaskName())

	assert.NoError(t, env.GetWorkflowError())
	assert.Equal(t, []string{"task-two"}, runOrder)
	assert.Nil(t, state.CANStartFrom)
}

func TestDoTaskBuilderShouldSkip(t *testing.T) {
	t.Helper()

	builder := newTestDoTaskBuilder("skip-check")

	tests := []struct {
		name        string
		task        workflowFunc
		taskID      string
		state       *utils.State
		expectSkip  bool
		expectedCAN *string
	}{
		{
			name: "tasks flagged as never skip are always executed",
			task: func() workflowFunc {
				tb := newFakeTaskBuilder("query-listener", &model.TaskBase{})
				tb.neverSkipCAN = true
				return workflowFunc{
					TaskBuilder: tb,
					Name:        tb.GetTaskName(),
				}
			}(),
			taskID: "query-listener-0",
			state: func() *utils.State {
				s := utils.NewState()
				s.CANStartFrom = utils.Ptr("query-listener-0")
				return s
			}(),
			expectSkip:  false,
			expectedCAN: utils.Ptr("query-listener-0"),
		},
		{
			name: "matching task ID resumes execution",
			task: func() workflowFunc {
				return workflowFunc{
					TaskBuilder: newFakeTaskBuilder("task-two", &model.TaskBase{}),
					Name:        "task-two",
				}
			}(),
			taskID: "task-two-1",
			state: func() *utils.State {
				s := utils.NewState()
				s.CANStartFrom = utils.Ptr("task-two-1")
				return s
			}(),
			expectSkip:  false,
			expectedCAN: nil,
		},
		{
			name: "different task ID keeps skipping",
			task: func() workflowFunc {
				return workflowFunc{
					TaskBuilder: newFakeTaskBuilder("task-one", &model.TaskBase{}),
					Name:        "task-one",
				}
			}(),
			taskID: "task-one-0",
			state: func() *utils.State {
				s := utils.NewState()
				s.CANStartFrom = utils.Ptr("task-two-1")
				return s
			}(),
			expectSkip:  true,
			expectedCAN: utils.Ptr("task-two-1"),
		},
		{
			name: "no continue-as-new state never skips",
			task: func() workflowFunc {
				return workflowFunc{
					TaskBuilder: newFakeTaskBuilder("task-zero", &model.TaskBase{}),
					Name:        "task-zero",
				}
			}(),
			taskID:      "task-zero-0",
			state:       utils.NewState(),
			expectSkip:  false,
			expectedCAN: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			result := builder.shouldSkip(tc.taskID, tc.task, tc.state)

			assert.Equal(t, tc.expectSkip, result)
			if tc.expectedCAN == nil {
				assert.Nil(t, tc.state.CANStartFrom)
			} else if assert.NotNil(t, tc.state.CANStartFrom) {
				assert.Equal(t, *tc.expectedCAN, *tc.state.CANStartFrom)
			}
		})
	}
}

func TestDoTaskBuilderBuildSkipsNestedDoAfterNonDoTask(t *testing.T) {
	// Outer do: first a non-do task, then a nested do task.
	task := &model.DoTask{
		Do: &model.TaskList{
			{
				Key: "step1",
				Task: &model.SetTask{
					Set: map[string]any{"a": "Homer"},
				},
			},
			{
				Key: "nested",
				Task: &model.DoTask{
					Do: &model.TaskList{
						{
							Key: "step2",
							Task: &model.SetTask{
								Set: map[string]any{"b": "Marge"},
							},
						},
					},
				},
			},
		},
	}

	temporalWorker := new(WorkflowRegistryMock)

	temporalWorker.
		On("RegisterWorkflowWithOptions", mock.Anything, workflow.RegisterOptions{
			Name: "nested",
		}).
		Once()
	temporalWorker.
		On("RegisterWorkflowWithOptions", mock.Anything, workflow.RegisterOptions{
			Name: "do-task",
		}).
		Once()

	builder, err := NewDoTaskBuilder(temporalWorker, task, "do-task", testWorkflow, testEvents)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "do-task"})

	env.ExecuteWorkflow("do-task")
	assert.NoError(t, env.GetWorkflowError())

	var result any
	assert.NoError(t, env.GetWorkflowResult(&result))

	// Because a non-do task ran first, the nested do should be skipped,
	// so the result and state should only reflect step1.
	assert.Equal(t, map[string]any{"a": "Homer"}, result)
	assert.Equal(t, "Homer", state.Data["a"])
	assert.Nil(t, state.Data["b"])
}

func newOutputWorkflowFuncs(runOrder *[]string, capturedState **utils.State) []workflowFunc {
	taskOneBase := &model.TaskBase{
		Export: &model.Export{
			As: model.NewObjectOrRuntimeExpr("first"),
		},
	}
	taskTwoBase := &model.TaskBase{
		Export: &model.Export{
			As: model.NewObjectOrRuntimeExpr("second"),
		},
	}

	taskOneBuilder := newFakeTaskBuilder("task-one", taskOneBase)
	taskTwoBuilder := newFakeTaskBuilder("task-two", taskTwoBase)

	return []workflowFunc{
		{
			TaskBuilder: taskOneBuilder,
			Name:        taskOneBuilder.GetTaskName(),
			Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				*capturedState = state
				*runOrder = append(*runOrder, "task-one")
				return map[string]any{
					"value": "one",
				}, nil
			},
		},
		{
			TaskBuilder: taskTwoBuilder,
			Name:        taskTwoBuilder.GetTaskName(),
			Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
				*runOrder = append(*runOrder, "task-two")
				return map[string]any{
					"value": "two",
				}, nil
			},
		},
	}
}

func newSimpleWorkflowFunc(name string, base *model.TaskBase, runOrder *[]string) workflowFunc {
	tb := newFakeTaskBuilder(name, base)
	return workflowFunc{
		TaskBuilder: tb,
		Name:        name,
		Func: func(ctx workflow.Context, input any, state *utils.State) (any, error) {
			*runOrder = append(*runOrder, name)
			return nil, nil
		},
	}
}

type fakeTaskBuilder struct {
	name         string
	neverSkipCAN bool
	task         model.Task
	shouldRun    bool
	shouldRunErr error
	parseErr     error
}

func newFakeTaskBuilder(name string, base *model.TaskBase) *fakeTaskBuilder {
	return &fakeTaskBuilder{
		name:         name,
		neverSkipCAN: false,
		task:         &mockTask{base: base},
		shouldRun:    true,
	}
}

func (f *fakeTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return nil, nil
}

func (f *fakeTaskBuilder) GetTask() model.Task {
	return f.task
}

func (f *fakeTaskBuilder) GetTaskName() string {
	return f.name
}

func (f *fakeTaskBuilder) NeverSkipCAN() bool {
	return f.neverSkipCAN
}

func (f *fakeTaskBuilder) ParseMetadata(workflow.Context, *utils.State) error {
	return f.parseErr
}

func (f *fakeTaskBuilder) PostLoad() error {
	return nil
}

func (f *fakeTaskBuilder) ShouldRun(*utils.State) (bool, error) {
	if f.shouldRunErr != nil {
		return false, f.shouldRunErr
	}
	return f.shouldRun, nil
}

func newTestDoTaskBuilder(name string, opts ...DoTaskOpts) *DoTaskBuilder {
	var doOpts DoTaskOpts
	if len(opts) == 1 {
		doOpts = opts[0]
	}

	return &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			doc:          testWorkflow,
			eventEmitter: testEvents,
			name:         name,
			task:         &model.DoTask{},
		},
		opts: doOpts,
	}
}

type WorkflowRegistryMock struct {
	mock.Mock
}

// RegisterActivity implements [worker.Worker].
func (m *WorkflowRegistryMock) RegisterActivity(a any) {
	panic("unimplemented")
}

// RegisterActivityWithOptions implements [worker.Worker].
func (m *WorkflowRegistryMock) RegisterActivityWithOptions(a any, options activity.RegisterOptions) {
	panic("unimplemented")
}

// RegisterDynamicActivity implements [worker.Worker].
func (m *WorkflowRegistryMock) RegisterDynamicActivity(a any, options activity.DynamicRegisterOptions) {
	panic("unimplemented")
}

// RegisterDynamicWorkflow implements [worker.Worker].
func (m *WorkflowRegistryMock) RegisterDynamicWorkflow(w any, options workflow.DynamicRegisterOptions) {
	panic("unimplemented")
}

// RegisterNexusService implements [worker.Worker].
func (m *WorkflowRegistryMock) RegisterNexusService(*nexus.Service) {
	panic("unimplemented")
}

// RegisterWorkflow implements [worker.Worker].
func (m *WorkflowRegistryMock) RegisterWorkflow(w any) {
	panic("unimplemented")
}

// Run implements [worker.Worker].
func (m *WorkflowRegistryMock) Run(interruptCh <-chan any) error {
	panic("unimplemented")
}

// Start implements [worker.Worker].
func (m *WorkflowRegistryMock) Start() error {
	panic("unimplemented")
}

// Stop implements [worker.Worker].
func (m *WorkflowRegistryMock) Stop() {
	panic("unimplemented")
}

func (m *WorkflowRegistryMock) RegisterWorkflowWithOptions(w any, opts workflow.RegisterOptions) {
	m.Called(w, opts)
}

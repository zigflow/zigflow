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

package zigflow_test

import (
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// stubWorker is a minimal worker.Worker implementation for tests that only
// need workflow registration to succeed. All other methods panic to catch
// accidental calls during the build/normalisation phase.
type stubWorker struct{}

func (stubWorker) RegisterWorkflowWithOptions(_ any, _ workflow.RegisterOptions) {}
func (stubWorker) RegisterWorkflow(_ any)                                        {}
func (stubWorker) RegisterActivity(_ any)                                        { panic("unimplemented") }
func (stubWorker) RegisterActivityWithOptions(_ any, _ activity.RegisterOptions) {
	panic("unimplemented")
}

func (stubWorker) RegisterDynamicActivity(_ any, _ activity.DynamicRegisterOptions) {
	panic("unimplemented")
}

func (stubWorker) RegisterDynamicWorkflow(_ any, _ workflow.DynamicRegisterOptions) {
	panic("unimplemented")
}
func (stubWorker) RegisterNexusService(_ *nexus.Service) { panic("unimplemented") }
func (stubWorker) Run(_ <-chan any) error                { panic("unimplemented") }
func (stubWorker) Start() error                          { panic("unimplemented") }
func (stubWorker) Stop()                                 { panic("unimplemented") }

// newTestWorkflow builds a minimal *model.Workflow whose Do list contains
// exactly one task. The DSL version satisfies the version constraint checked
// by LoadFromFile but is not checked by NewWorkflow, so any value is fine.
func newTestWorkflow(name string, task model.Task) *model.Workflow {
	return &model.Workflow{
		Document: model.Document{
			Name: name,
			DSL:  "1.0.0",
		},
		Do: &model.TaskList{
			{Key: "step", Task: task},
		},
	}
}

// TestNewWorkflowRunsPostLoadBeforeBuild_RunScriptNilAwait is a regression
// test for the case where NewWorkflow() is called without going through
// LoadFromFile. Before the fix, a script RunTask with nil Await caused a nil-
// pointer panic inside Build() when it checked !*t.task.Run.Await. After the
// fix, PostLoad() runs inside NewWorkflow() and sets the default before Build.
func TestNewWorkflowRunsPostLoadBeforeBuild_RunScriptNilAwait(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Script: &model.Script{
				Language:   "python",
				InlineCode: utils.Ptr("print('hello')"),
				// Await is intentionally nil — PostLoad must set the default
			},
		},
	}

	err := zigflow.NewWorkflow(stubWorker{}, newTestWorkflow("run-script-test", task), nil, nil, nil)
	require.NoError(t, err, "NewWorkflow must not panic or error when Await is nil before PostLoad")

	assert.NotNil(t, task.Run.Await, "PostLoad must have set Await before Build ran")
	assert.True(t, *task.Run.Await, "PostLoad default for Await must be true")
}

// TestNewWorkflowRunsPostLoadBeforeBuild_GRPCEmptyHostPort is a regression
// test for CallGRPC tasks constructed programmatically without host/port.
// Before the fix, Build() no longer set these defaults (they moved to
// PostLoad), so activity calls would receive an empty address. After the fix,
// PostLoad() in NewWorkflow() fills in the defaults.
func TestNewWorkflowRunsPostLoadBeforeBuild_GRPCEmptyHostPort(t *testing.T) {
	task := &model.CallGRPC{
		With: model.GRPCArguments{
			Service: model.GRPCService{
				// Host and Port intentionally left at zero values
			},
		},
	}

	err := zigflow.NewWorkflow(stubWorker{}, newTestWorkflow("grpc-test", task), nil, nil, nil)
	require.NoError(t, err, "NewWorkflow must not error when gRPC host/port are absent")

	assert.Equal(t, "localhost", task.With.Service.Host, "PostLoad must have set the default Host")
	assert.Equal(t, 50051, task.With.Service.Port, "PostLoad must have set the default Port")
}

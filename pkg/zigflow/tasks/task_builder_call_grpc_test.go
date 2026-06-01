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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestCallGRPCTaskBuilderEvaluatesWithBeforeActivity(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	var captured *model.CallGRPC
	env.RegisterActivityWithOptions(func(_ context.Context, task *model.CallGRPC, _ any, _ *utils.State) (any, error) {
		captured = task
		return map[string]any{testConstOK: true}, nil
	}, activity.RegisterOptions{Name: "CallGRPCActivity"})

	task := &model.CallGRPC{
		Call: "grpc",
		With: model.GRPCArguments{
			Method: "Command1",
			Arguments: map[string]any{
				"input": "${ $env." + testConstGRPCInputEnv + " }",
			},
		},
	}

	b, err := NewCallGRPCTaskBuilder(nil, task, "grpcTask", nil, testEvents, nil)
	assert.NoError(t, err)

	fn, err := b.Build()
	assert.NoError(t, err)

	workflowFunc := func(ctx workflow.Context) (any, error) {
		state := utils.NewState()
		state.Env[testConstGRPCInputEnv] = testConstHello
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, nil, state)
	}

	env.ExecuteWorkflow(workflowFunc)

	assert.NoError(t, env.GetWorkflowError())
	require.NotNil(t, captured)
	assert.Equal(t, testConstHello, captured.With.Arguments["input"])
}

func TestCallGRPCTaskBuilderPostLoadSetsHostDefault(t *testing.T) {
	task := &model.CallGRPC{
		With: model.GRPCArguments{
			Service: model.GRPCService{
				// Host intentionally omitted
				Port: 9090,
			},
		},
	}

	builder, err := NewCallGRPCTaskBuilder(nil, task, "grpc-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.Equal(t, "localhost", task.With.Service.Host, "empty Host must default to localhost")
	assert.Equal(t, 9090, task.With.Service.Port, "explicit Port must be unchanged")
}

func TestCallGRPCTaskBuilderPostLoadSetsPortDefault(t *testing.T) {
	task := &model.CallGRPC{
		With: model.GRPCArguments{
			Service: model.GRPCService{
				Host: "grpc.internal",
				// Port intentionally omitted (zero value)
			},
		},
	}

	builder, err := NewCallGRPCTaskBuilder(nil, task, "grpc-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.Equal(t, "grpc.internal", task.With.Service.Host, "explicit Host must be unchanged")
	assert.Equal(t, 50051, task.With.Service.Port, "zero Port must default to 50051")
}

func TestCallGRPCTaskBuilderPostLoadPreservesExplicitValues(t *testing.T) {
	task := &model.CallGRPC{
		With: model.GRPCArguments{
			Service: model.GRPCService{
				Host: "grpc.internal",
				Port: 9090,
			},
		},
	}

	builder, err := NewCallGRPCTaskBuilder(nil, task, "grpc-task", nil, testEvents, nil)
	assert.NoError(t, err)
	assert.NoError(t, builder.PostLoad())

	assert.Equal(t, "grpc.internal", task.With.Service.Host, "explicit Host must not be overwritten")
	assert.Equal(t, 9090, task.With.Service.Port, "explicit Port must not be overwritten")
}

func TestCallGRPCTaskBuilderBuildDoesNotMutateTask(t *testing.T) {
	task := &model.CallGRPC{
		With: model.GRPCArguments{
			Service: model.GRPCService{
				// Both fields zero — Build() must not write defaults any more
			},
		},
	}

	builder, err := NewCallGRPCTaskBuilder(nil, task, "grpc-task", nil, testEvents, nil)
	assert.NoError(t, err)

	// Call Build() without PostLoad() to verify Build() no longer sets defaults.
	_, err = builder.Build()
	assert.NoError(t, err)

	assert.Equal(t, "", task.With.Service.Host, "Build() must not set Host default")
	assert.Equal(t, 0, task.With.Service.Port, "Build() must not set Port default")
}

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

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
)

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

func newTestGRPCTask() *model.CallGRPC {
	return &model.CallGRPC{
		With: model.GRPCArguments{
			Service: model.GRPCService{Host: "localhost", Port: 50051},
		},
	}
}

func TestCallGRPCTaskBuilderRegistersPerTaskActivityName(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-grpc-metrics"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-grpc-metrics.callBackend",
		}).
		Once()

	b, err := NewCallGRPCTaskBuilder(w, newTestGRPCTask(), "callBackend", doc, testEvents, nil)
	assert.NoError(t, err)

	_, err = b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

func TestCallGRPCTaskBuilderRegistersOncePerWorker(t *testing.T) {
	assertRegistersOncePerWorker(t, "wf-grpc-dedup", "invoke",
		func(w *WorkflowRegistryMock, doc *model.Workflow, taskName string) (TaskBuilder, error) {
			return NewCallGRPCTaskBuilder(w, newTestGRPCTask(), taskName, doc, testEvents, nil)
		})
}

func TestCallGRPCTaskBuilderBuildWithoutWorker(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-grpc-nil-worker"}}

	b, err := NewCallGRPCTaskBuilder(nil, newTestGRPCTask(), "step", doc, testEvents, nil)
	assert.NoError(t, err)

	fn, err := b.Build()
	assert.NoError(t, err)
	assert.NotNil(t, fn)
}

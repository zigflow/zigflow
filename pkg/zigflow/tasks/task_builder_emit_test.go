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
	"encoding/base64"
	"testing"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func newEmitTaskBuilder(eventType string, additional map[string]any) *EmitTaskBuilder {
	builder, _ := NewEmitTaskBuilder(nil, &model.EmitTask{
		Emit: model.EmitTaskConfiguration{
			Event: model.EmitEventDefinition{
				With: &model.EventProperties{
					Type:       eventType,
					Additional: additional,
				},
			},
		},
	}, "emit-task", testWorkflow, testEvents, nil)

	return builder
}

// runEmitTask executes fn against a fresh workflow stream and returns the
// stream log so tests can inspect exactly what was published.
func runEmitTask(t *testing.T, name string, fn TemporalWorkflowFunc, state *utils.State) ([]workflowstreams.WireItem, error) {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) ([]workflowstreams.WireItem, error) {
		if state == nil {
			stream, err := workflowstreams.NewWorkflowStream(ctx, nil)
			if err != nil {
				return nil, err
			}
			state = utils.NewState()
			state.SetStream(stream)
		}

		if _, err := fn(ctx, nil, state); err != nil {
			return nil, err
		}

		stream, err := state.GetStream(ctx)
		if err != nil {
			return nil, err
		}

		snapshot, err := stream.GetState(time.Minute)
		if err != nil {
			return nil, err
		}

		return snapshot.Log, nil
	}, workflow.RegisterOptions{Name: name})

	env.ExecuteWorkflow(name)
	require.True(t, env.IsWorkflowCompleted())

	if err := env.GetWorkflowError(); err != nil {
		return nil, err
	}

	var items []workflowstreams.WireItem
	require.NoError(t, env.GetWorkflowResult(&items))

	return items, nil
}

// decodeStreamItem converts a stream wire item back into the published value.
func decodeStreamItem(t *testing.T, item workflowstreams.WireItem) any {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(item.Data)
	require.NoError(t, err)

	payload := &commonpb.Payload{}
	require.NoError(t, payload.Unmarshal(raw))

	var value any
	require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payload, &value))

	return value
}

func TestEmitTaskBuilderBuild(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantTopic string
		wantErr   string
	}{
		{
			name:      "simple topic",
			eventType: "io.temporal.streams.topic1",
			wantTopic: "topic1",
		},
		{
			name:      "topic containing dots",
			eventType: "io.temporal.streams.order.status",
			wantTopic: "order.status",
		},
		{
			name:      "only the leading prefix is removed",
			eventType: "io.temporal.streams.io.temporal.streams.nested",
			wantTopic: "io.temporal.streams.nested",
		},
		{
			name:      "prefix with no topic",
			eventType: "io.temporal.streams",
			wantErr:   `unsupported event type "io.temporal.streams"`,
		},
		{
			name:      "prefix with empty topic",
			eventType: "io.temporal.streams.",
			wantErr:   "temporal stream topic must not be empty",
		},
		{
			name:      "empty event type",
			eventType: "",
			wantErr:   `unsupported event type ""`,
		},
		{
			name:      "non-stream event type",
			eventType: "com.example.order.created",
			wantErr:   `unsupported event type "com.example.order.created"`,
		},
		{
			name:      "prefix is case sensitive",
			eventType: "IO.TEMPORAL.STREAMS.topic1",
			wantErr:   `unsupported event type "IO.TEMPORAL.STREAMS.topic1"`,
		},
		{
			name:      "prefix must be at the start",
			eventType: "com.example.io.temporal.streams.topic1",
			wantErr:   `unsupported event type "com.example.io.temporal.streams.topic1"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn, err := newEmitTaskBuilder(tc.eventType, nil).Build()
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				assert.Nil(t, fn)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, fn)

			items, err := runEmitTask(t, "emit-"+tc.name, fn, nil)
			require.NoError(t, err)
			require.Len(t, items, 1)
			assert.Equal(t, tc.wantTopic, items[0].Topic)
		})
	}
}

// The data property is published as-is and is optional.
func TestEmitTaskBuilderPublishesData(t *testing.T) {
	tests := []struct {
		name       string
		additional map[string]any
	}{
		{
			name: "object data",
			additional: map[string]any{
				testConstData: map[string]any{
					"number":         3.141,
					testConstMessage: "Hello world",
					"happy":          true,
				},
			},
		},
		{
			name: "scalar data",
			additional: map[string]any{
				testConstData: testConstHello,
			},
		},
		{
			name: "data omitted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn, err := newEmitTaskBuilder("io.temporal.streams.topic1", tc.additional).Build()
			require.NoError(t, err)

			items, err := runEmitTask(t, "emit-data-"+tc.name, fn, nil)
			require.NoError(t, err)
			require.Len(t, items, 1)
			assert.Equal(t, "topic1", items[0].Topic)
			assert.Equal(t, tc.additional[testConstData], decodeStreamItem(t, items[0]))
		})
	}
}

// When the state carries no live stream (for example after continue-as-new),
// the emit task must create one from the carried stream state.
func TestEmitTaskBuilderCreatesStreamWhenMissing(t *testing.T) {
	fn, err := newEmitTaskBuilder("io.temporal.streams.topic1", nil).Build()
	require.NoError(t, err)

	items, err := runEmitTask(t, "emit-missing-stream", fn, utils.NewState())
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "topic1", items[0].Topic)
}

func TestEmitTaskBuilderErrors(t *testing.T) {
	tests := []struct {
		name       string
		additional map[string]any
		state      *utils.State
		wantErr    string
	}{
		{
			name: "stream cannot be restored",
			state: func() *utils.State {
				s := utils.NewState()
				s.StreamState = &workflowstreams.WorkflowStreamState{
					Log: []workflowstreams.WireItem{{Topic: "topic1", Data: "not-base64!"}},
				}
				return s
			}(),
			wantErr: "error retrieving stream instance",
		},
		{
			name: "data cannot be serialised",
			additional: map[string]any{
				testConstData: make(chan int),
			},
			wantErr: "workflowstreams: convert value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn, err := newEmitTaskBuilder("io.temporal.streams.topic1", tc.additional).Build()
			require.NoError(t, err)

			_, err = runEmitTask(t, "emit-error-"+tc.name, fn, tc.state)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

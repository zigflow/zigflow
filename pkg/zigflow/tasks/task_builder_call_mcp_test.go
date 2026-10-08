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
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const (
	testMCPEndpoint     = "https://example.com/mcp"
	testMCPKeyMethod    = "method"
	testMCPKeyTransport = "transport"
	testMCPKeyHTTP      = "http"
	testMCPKeyName      = "name"
	testMCPKeyArguments = "arguments"
	testMCPKeyMinutes   = "minutes"
	testMCPInputMessage = "${ $input.message }"
)

func newTestMCPTask(with map[string]any) *model.CallFunction {
	return &model.CallFunction{
		Call: customCallMCPActivity,
		With: with,
	}
}

func newTestMCPHTTPWith() map[string]any {
	return map[string]any{
		testMCPKeyMethod: "tools/list",
		testMCPKeyTransport: map[string]any{
			testMCPKeyHTTP: map[string]any{
				"endpoint": testMCPEndpoint,
			},
		},
	}
}

func TestNewCallMCPTaskBuilderDecodesHTTPTransport(t *testing.T) {
	tests := []struct {
		name     string
		endpoint any
	}{
		{
			name:     "string endpoint",
			endpoint: testMCPEndpoint,
		},
		{
			name:     "object endpoint",
			endpoint: map[string]any{"uri": testMCPEndpoint},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestMCPTask(map[string]any{
				"protocolVersion": "2025-03-26",
				testMCPKeyMethod:  "tools/call",
				"parameters": map[string]any{
					testMCPKeyName:      "echo",
					testMCPKeyArguments: map[string]any{"message": testMCPInputMessage},
				},
				"client": map[string]any{testMCPKeyName: "custom-client", "version": "1.2.3"},
				testMCPKeyTransport: map[string]any{
					testMCPKeyHTTP: map[string]any{
						"endpoint": tt.endpoint,
						"headers":  map[string]any{"Authorization": "Bearer token"},
					},
				},
			})

			b, err := NewCallMCPTaskBuilder(nil, task, "callMCP", nil, testEvents, nil)
			require.NoError(t, err)

			with := b.task.With
			assert.Equal(t, customCallMCPActivity, b.task.Call)
			assert.Equal(t, "2025-03-26", with.ProtocolVersion)
			assert.Equal(t, "tools/call", with.Method)
			assert.Equal(t, map[string]any{
				testMCPKeyName:      "echo",
				testMCPKeyArguments: map[string]any{"message": testMCPInputMessage},
			}, with.Parameters)
			require.NotNil(t, with.Client)
			assert.Equal(t, "custom-client", with.Client.Name)
			assert.Equal(t, "1.2.3", with.Client.Version)

			require.NotNil(t, with.Transport)
			assert.Nil(t, with.Transport.STDIO)
			require.NotNil(t, with.Transport.HTTP)
			require.NotNil(t, with.Transport.HTTP.Endpoint)
			assert.Equal(t, testMCPEndpoint, with.Transport.HTTP.Endpoint.String())
			assert.Equal(t, map[string]string{"Authorization": "Bearer token"}, with.Transport.HTTP.Headers)
		})
	}
}

func TestNewCallMCPTaskBuilderDecodesSTDIOTransport(t *testing.T) {
	task := newTestMCPTask(map[string]any{
		testMCPKeyMethod: "tools/list",
		testMCPKeyTransport: map[string]any{
			"stdio": map[string]any{
				"command":           "npx",
				testMCPKeyArguments: []any{"-y", "@modelcontextprotocol/server-everything"},
				"environment": map[string]any{
					"API_KEY": "secret",
					"DEBUG":   "enabled",
				},
			},
		},
	})

	b, err := NewCallMCPTaskBuilder(nil, task, "callMCP", nil, testEvents, nil)
	require.NoError(t, err)

	transport := b.task.With.Transport
	require.NotNil(t, transport)
	assert.Nil(t, transport.HTTP)
	require.NotNil(t, transport.STDIO)
	assert.Equal(t, "npx", transport.STDIO.Command)
	assert.Equal(t, []string{"-y", "@modelcontextprotocol/server-everything"}, transport.STDIO.Arguments)
	assert.Equal(t, map[string]string{"API_KEY": "secret", "DEBUG": "enabled"}, transport.STDIO.Environment)
}

func TestNewCallMCPTaskBuilderTimeout(t *testing.T) {
	t.Run("defaults to 30 seconds when omitted", func(t *testing.T) {
		b, err := NewCallMCPTaskBuilder(nil, newTestMCPTask(newTestMCPHTTPWith()), "callMCP", nil, testEvents, nil)
		require.NoError(t, err)

		require.NotNil(t, b.task.With.Timeout)
		assert.Equal(t, 30*time.Second, utils.ToDuration(b.task.With.Timeout))
	})

	t.Run("configured timeout overrides the default", func(t *testing.T) {
		with := newTestMCPHTTPWith()
		with["timeout"] = map[string]any{testMCPKeyMinutes: 2, "seconds": 5}

		b, err := NewCallMCPTaskBuilder(nil, newTestMCPTask(with), "callMCP", nil, testEvents, nil)
		require.NoError(t, err)

		require.NotNil(t, b.task.With.Timeout)
		assert.Equal(t, 2*time.Minute+5*time.Second, utils.ToDuration(b.task.With.Timeout))
	})
}

func TestNewCallMCPTaskBuilderRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(with map[string]any)
	}{
		{
			name:   "unknown field",
			mutate: func(with map[string]any) { with["unknown"] = true },
		},
		{
			name: "unknown transport field",
			mutate: func(with map[string]any) {
				with[testMCPKeyTransport].(map[string]any)["websocket"] = map[string]any{}
			},
		},
		{
			name: "non-string environment value",
			mutate: func(with map[string]any) {
				with[testMCPKeyTransport] = map[string]any{
					"stdio": map[string]any{
						"command":     "server",
						"environment": map[string]any{"PORT": 8080},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			with := newTestMCPHTTPWith()
			tt.mutate(with)

			_, err := NewCallMCPTaskBuilder(nil, newTestMCPTask(with), "callMCP", nil, testEvents, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "error converting mcp arguments")
		})
	}
}

func TestNewTaskBuilderDispatchesCallMCP(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-mcp-dispatch"}}

	b, err := NewTaskBuilder("callMCP", newTestMCPTask(newTestMCPHTTPWith()), nil, doc, testEvents, nil, []string{"callMCP"})
	require.NoError(t, err)
	assert.IsType(t, &CallMCPTaskBuilder{}, b)
}

func TestCallMCPTaskBuilderRegistersOncePerWorker(t *testing.T) {
	assertRegistersOncePerWorker(t, "wf-mcp-dedup", "invoke",
		func(w *WorkflowRegistryMock, doc *model.Workflow, taskName string) (TaskBuilder, error) {
			return NewCallMCPTaskBuilder(w, newTestMCPTask(newTestMCPHTTPWith()), taskName, doc, testEvents, nil)
		})
}

// TestCallMCPTaskBuilderStartToCloseTimeout verifies the scheduled activity's
// StartToCloseTimeout is never shorter than the MCP call timeout, as the MCP
// timeout is derived from the activity context and would otherwise be capped.
func TestCallMCPTaskBuilderStartToCloseTimeout(t *testing.T) {
	tests := []struct {
		name            string
		mcpTimeout      map[string]any
		activityTimeout map[string]any
		want            time.Duration
	}{
		{
			name: "default MCP timeout raises the default activity timeout",
			want: 30 * time.Second,
		},
		{
			name:       "shorter MCP timeout does not reduce the activity timeout",
			mcpTimeout: map[string]any{"seconds": 5},
			want:       15 * time.Second,
		},
		{
			name:       "longer MCP timeout raises the activity timeout",
			mcpTimeout: map[string]any{testMCPKeyMinutes: 2},
			want:       2 * time.Minute,
		},
		{
			name:            "longer configured activity timeout is preserved",
			activityTimeout: map[string]any{testMCPKeyMinutes: 5},
			want:            5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &model.Workflow{Document: model.Document{Name: "wf-mcp-timeout"}}

			with := newTestMCPHTTPWith()
			if tt.mcpTimeout != nil {
				with["timeout"] = tt.mcpTimeout
			}
			task := newTestMCPTask(with)
			if tt.activityTimeout != nil {
				task.Metadata = map[string]any{
					metadata.MetadataActivityOptions: map[string]any{"startToCloseTimeout": tt.activityTimeout},
				}
			}

			b, err := NewCallMCPTaskBuilder(nil, task, "callMCP", doc, testEvents, nil)
			require.NoError(t, err)
			fn, err := b.Build()
			require.NoError(t, err)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			got := make(chan time.Duration, 1)
			env.RegisterActivityWithOptions(
				func(ctx context.Context, _ *models.CallMCP, _ any, _ *utils.State) (any, error) {
					got <- activity.GetInfo(ctx).StartToCloseTimeout
					return nil, nil
				},
				activity.RegisterOptions{Name: b.perTaskActivityName()},
			)

			env.ExecuteWorkflow(func(ctx workflow.Context) error {
				// Apply the generic activity options as the do task does
				ctx, err := metadata.SetActivityOptions(ctx, doc, task.GetBase(), "callMCP")
				if err != nil {
					return err
				}
				_, err = fn(ctx, nil, utils.NewState())
				return err
			})
			require.NoError(t, env.GetWorkflowError())

			select {
			case d := <-got:
				assert.Equal(t, tt.want, d)
			default:
				t.Fatal("MCP activity was not scheduled")
			}
		})
	}
}

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

package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/ctxpropagator"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	sdkinterceptor "go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	testCorrelationID = "corr-123"
	testMessage       = "a log message"
	testTenantKey     = "tenant"
	testTenant        = "acme"
)

type logCall struct {
	level   string
	msg     string
	keyvals []any
}

// recordingLogger captures everything written to it so tests can assert on the
// key/value pairs the interceptor adds.
type recordingLogger struct {
	calls []logCall
}

func (l *recordingLogger) record(level, msg string, keyvals []any) {
	l.calls = append(l.calls, logCall{level: level, msg: msg, keyvals: keyvals})
}

func (l *recordingLogger) Debug(msg string, keyvals ...any) { l.record("debug", msg, keyvals) }
func (l *recordingLogger) Error(msg string, keyvals ...any) { l.record("error", msg, keyvals) }
func (l *recordingLogger) Info(msg string, keyvals ...any)  { l.record("info", msg, keyvals) }
func (l *recordingLogger) Warn(msg string, keyvals ...any)  { l.record("warn", msg, keyvals) }

var _ log.Logger = (*recordingLogger)(nil)

// keyval returns the value logged against key, and whether it was present.
func keyval(keyvals []any, key string) (any, bool) {
	for i := 0; i+1 < len(keyvals); i += 2 {
		if k, ok := keyvals[i].(string); ok && k == key {
			return keyvals[i+1], true
		}
	}

	return nil, false
}

func TestNewLogger_CorrelationID(t *testing.T) {
	tests := []struct {
		name       string
		propagated any
		setValue   bool
		want       bool
	}{
		{
			name:     "no propagated values",
			setValue: false,
		},
		{
			name:       "propagated values are not a map",
			propagated: "not-a-map",
			setValue:   true,
		},
		{
			name:       "propagated values without a correlation ID",
			propagated: map[string]any{testTenantKey: testTenant},
			setValue:   true,
		},
		{
			name:       "correlation ID is not a string",
			propagated: map[string]any{ctxpropagator.CorrelationID: 42},
			setValue:   true,
		},
		{
			name:       "correlation ID is present",
			propagated: map[string]any{ctxpropagator.CorrelationID: testCorrelationID},
			setValue:   true,
			want:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.setValue {
				ctx = context.WithValue(ctx, ctxpropagator.PropagateKey, tc.propagated)
			}

			base := &recordingLogger{}
			newLogger(ctx, base).Info(testMessage, "extra", "value")

			require.Len(t, base.calls, 1)
			call := base.calls[0]
			assert.Equal(t, testMessage, call.msg)

			// Caller-supplied key/value pairs must always survive.
			extra, ok := keyval(call.keyvals, "extra")
			assert.True(t, ok, "caller key/value pairs must be preserved")
			assert.Equal(t, "value", extra)

			correlationID, ok := keyval(call.keyvals, ctxpropagator.CorrelationID)
			require.Equal(t, tc.want, ok)

			if tc.want {
				assert.Equal(t, testCorrelationID, correlationID)
			}
		})
	}
}

func TestNewLogger_ForwardsEveryLevel(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxpropagator.PropagateKey, map[string]any{
		ctxpropagator.CorrelationID: testCorrelationID,
	})

	tests := []struct {
		name  string
		level string
		logFn func(log.Logger, string, ...any)
	}{
		{"debug", "debug", log.Logger.Debug},
		{"info", "info", log.Logger.Info},
		{"warn", "warn", log.Logger.Warn},
		{"error", "error", log.Logger.Error},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base := &recordingLogger{}
			tc.logFn(newLogger(ctx, base), testMessage, "extra", "value")

			require.Len(t, base.calls, 1)
			call := base.calls[0]

			assert.Equal(t, tc.level, call.level)
			assert.Equal(t, testMessage, call.msg)
			assert.Equal(t, []any{
				ctxpropagator.CorrelationID, testCorrelationID,
				"extra", "value",
			}, call.keyvals)
		})
	}
}

func TestNewLogger_WithoutKeyvals(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxpropagator.PropagateKey, map[string]any{
		ctxpropagator.CorrelationID: testCorrelationID,
	})

	base := &recordingLogger{}
	newLogger(ctx, base).Warn(testMessage)

	require.Len(t, base.calls, 1)
	assert.Equal(t, []any{ctxpropagator.CorrelationID, testCorrelationID}, base.calls[0].keyvals)
}

// headerWithCorrelationID builds a Temporal header carrying propagated values,
// as an upstream caller would.
func headerWithCorrelationID(t *testing.T, values map[string]any) *commonpb.Header {
	t.Helper()

	payload, err := converter.GetDefaultDataConverter().ToPayload(values)
	require.NoError(t, err)

	return &commonpb.Header{
		Fields: map[string]*commonpb.Payload{ctxpropagator.HeaderKey: payload},
	}
}

func loggingWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info(testMessage, "extra", "value")

	return nil
}

func TestLoggerInterceptor_WorkflowLogger(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{
			name:   "correlation ID is added when propagated",
			values: map[string]any{ctxpropagator.CorrelationID: testCorrelationID},
			want:   true,
		},
		{
			name:   "correlation ID is omitted when not propagated",
			values: map[string]any{testTenantKey: testTenant},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base := &recordingLogger{}

			var suite testsuite.WorkflowTestSuite
			suite.SetLogger(base)

			env := suite.NewTestWorkflowEnvironment()
			env.SetWorkerOptions(worker.Options{
				Interceptors: []sdkinterceptor.WorkerInterceptor{NewLoggerInterceptor()},
			})
			env.SetContextPropagators([]workflow.ContextPropagator{ctxpropagator.NewContextPropagator()})
			env.SetHeader(headerWithCorrelationID(t, tc.values))

			env.ExecuteWorkflow(loggingWorkflow)

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())

			var call *logCall
			for i := range base.calls {
				if base.calls[i].msg == testMessage {
					call = &base.calls[i]
				}
			}
			require.NotNil(t, call, "the workflow's log call should reach the underlying logger")

			extra, ok := keyval(call.keyvals, "extra")
			assert.True(t, ok, "caller key/value pairs must be preserved")
			assert.Equal(t, "value", extra)

			correlationID, ok := keyval(call.keyvals, ctxpropagator.CorrelationID)
			require.Equal(t, tc.want, ok)

			if tc.want {
				assert.Equal(t, testCorrelationID, correlationID)
			}
		})
	}
}

func loggingActivity(ctx context.Context) error {
	activity.GetLogger(ctx).Info(testMessage, "extra", "value")

	return nil
}

func TestLoggerInterceptor_ActivityLogger(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{
			name:   "correlation ID is added when propagated",
			values: map[string]any{ctxpropagator.CorrelationID: testCorrelationID},
			want:   true,
		},
		{
			name:   "correlation ID is omitted when not propagated",
			values: map[string]any{testTenantKey: testTenant},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base := &recordingLogger{}

			var suite testsuite.WorkflowTestSuite
			suite.SetLogger(base)

			env := suite.NewTestActivityEnvironment()
			env.SetWorkerOptions(worker.Options{
				Interceptors: []sdkinterceptor.WorkerInterceptor{NewLoggerInterceptor()},
			})
			env.SetContextPropagators([]workflow.ContextPropagator{ctxpropagator.NewContextPropagator()})
			env.SetHeader(headerWithCorrelationID(t, tc.values))
			env.RegisterActivity(loggingActivity)

			_, err := env.ExecuteActivity(loggingActivity)
			require.NoError(t, err)

			var call *logCall
			for i := range base.calls {
				if base.calls[i].msg == testMessage {
					call = &base.calls[i]
				}
			}
			require.NotNil(t, call, "the activity's log call should reach the underlying logger")

			extra, ok := keyval(call.keyvals, "extra")
			assert.True(t, ok, "caller key/value pairs must be preserved")
			assert.Equal(t, "value", extra)

			correlationID, ok := keyval(call.keyvals, ctxpropagator.CorrelationID)
			require.Equal(t, tc.want, ok)

			if tc.want {
				assert.Equal(t, testCorrelationID, correlationID)
			}
		})
	}
}

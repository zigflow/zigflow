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

	"github.com/zigflow/zigflow/pkg/ctxpropagator"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// Hook into the activity interceptor
func (a *activityOutboundInterceptor) GetLogger(ctx context.Context) log.Logger {
	base := a.Next.GetLogger(ctx)

	return newLogger(ctx, base)
}

// Hook into the workflow interceptor
func (w *workflowOutboundInterceptor) GetLogger(
	ctx workflow.Context,
) log.Logger {
	base := w.Next.GetLogger(ctx)

	return newLogger(ctx, base)
}

type logContext interface {
	Value(any) any
}

type loggerInterceptor struct {
	interceptor.WorkerInterceptorBase
}

func (w *loggerInterceptor) InterceptActivity(
	ctx context.Context, next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &activityInboundInterceptor{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{
			Next: next,
		},
	}
}

func (w *loggerInterceptor) InterceptWorkflow(
	ctx workflow.Context, next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &workflowInboundInterceptor{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{
			Next: next,
		},
	}
}

func newLogger(ctx logContext, base log.Logger) log.Logger {
	// Search for context propagator data
	values, ok := ctx.Value(ctxpropagator.PropagateKey).(map[string]any)
	if !ok {
		return base
	}

	// Search for correlation ID
	correlationIDAny, ok := values[ctxpropagator.CorrelationID]
	if !ok {
		return base
	}

	// Check it's a string
	correlationID, ok := correlationIDAny.(string)
	if !ok {
		return base
	}

	// Return our custom logger
	return &logger{
		base:          base,
		correlationID: correlationID,
	}
}

type logger struct {
	base          log.Logger
	correlationID string
}

func (l *logger) toLoggerKeyvals(keyvals []any) []any {
	return append([]any{
		ctxpropagator.CorrelationID,
		l.correlationID,
	}, keyvals...)
}

func (l *logger) Debug(msg string, keyvals ...any) {
	l.base.Debug(msg, l.toLoggerKeyvals(keyvals)...)
}

func (l *logger) Error(msg string, keyvals ...any) {
	l.base.Error(msg, l.toLoggerKeyvals(keyvals)...)
}

func (l *logger) Info(msg string, keyvals ...any) {
	l.base.Info(msg, l.toLoggerKeyvals(keyvals)...)
}

func (l *logger) Warn(msg string, keyvals ...any) {
	l.base.Warn(msg, l.toLoggerKeyvals(keyvals)...)
}

var _ log.Logger = &logger{}

func NewLoggerInterceptor() interceptor.WorkerInterceptor {
	return &loggerInterceptor{}
}

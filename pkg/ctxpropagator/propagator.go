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

package ctxpropagator

import (
	"context"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

type contextKey struct{}

type propagator struct{}

const (
	CorrelationID = "correlationId" // Special key added to logger output
	HeaderKey     = "zigflow.propagated"
)

var PropagateKey = contextKey{}

func (s *propagator) Extract(ctx context.Context, reader workflow.HeaderReader) (context.Context, error) {
	if value, ok := reader.Get(HeaderKey); ok {
		var values map[string]any
		if err := converter.GetDefaultDataConverter().FromPayload(value, &values); err != nil {
			return ctx, err
		}
		ctx = context.WithValue(ctx, PropagateKey, values)
	}

	return ctx, nil
}

func (s *propagator) ExtractToWorkflow(ctx workflow.Context, reader workflow.HeaderReader) (workflow.Context, error) {
	if value, ok := reader.Get(HeaderKey); ok {
		var values map[string]any
		if err := converter.GetDefaultDataConverter().FromPayload(value, &values); err != nil {
			return ctx, err
		}
		ctx = workflow.WithValue(ctx, PropagateKey, values)
	}

	return ctx, nil
}

func (s *propagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	value := ctx.Value(PropagateKey)
	if value == nil {
		return nil
	}

	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	if err != nil {
		return err
	}

	writer.Set(HeaderKey, payload)

	return nil
}

func (s *propagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	value := ctx.Value(PropagateKey)
	if value == nil {
		return nil
	}

	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	if err != nil {
		return err
	}

	writer.Set(HeaderKey, payload)

	return nil
}

func NewContextPropagator() workflow.ContextPropagator {
	return &propagator{}
}

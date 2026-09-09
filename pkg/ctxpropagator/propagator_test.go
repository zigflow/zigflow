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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// testHeader is an in-memory stand-in for the Temporal header. The SDK only
// exposes the reader/writer interfaces, so tests supply their own.
type testHeader struct {
	fields map[string]*commonpb.Payload
}

func newTestHeader() *testHeader {
	return &testHeader{fields: map[string]*commonpb.Payload{}}
}

func (h *testHeader) Set(key string, value *commonpb.Payload) {
	h.fields[key] = value
}

func (h *testHeader) Get(key string) (*commonpb.Payload, bool) {
	value, ok := h.fields[key]

	return value, ok
}

// ForEachKey walks the keys in sorted order so tests never depend on Go's
// randomised map iteration.
func (h *testHeader) ForEachKey(handler func(string, *commonpb.Payload) error) error {
	keys := make([]string, 0, len(h.fields))
	for key := range h.fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if err := handler(key, h.fields[key]); err != nil {
			return err
		}
	}

	return nil
}

var (
	_ workflow.HeaderReader = (*testHeader)(nil)
	_ workflow.HeaderWriter = (*testHeader)(nil)
)

// malformedPayload claims to be JSON but is not decodable.
func malformedPayload() *commonpb.Payload {
	return &commonpb.Payload{
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
		Data:     []byte("{ this is not json"),
	}
}

func TestPropagator_InjectExtractRoundTrip(t *testing.T) {
	values := map[string]any{
		CorrelationID: "corr-123",
		"tenant":      "acme",
	}

	p := NewContextPropagator()
	header := newTestHeader()

	require.NoError(t, p.Inject(context.WithValue(context.Background(), PropagateKey, values), header))

	_, ok := header.Get(HeaderKey)
	require.True(t, ok, "Inject should write the propagated values to the header")

	ctx, err := p.Extract(context.Background(), header)
	require.NoError(t, err)

	assert.Equal(t, values, ctx.Value(PropagateKey))
}

func TestPropagator_InjectWithoutValues(t *testing.T) {
	p := NewContextPropagator()
	header := newTestHeader()

	require.NoError(t, p.Inject(context.Background(), header))

	_, ok := header.Get(HeaderKey)
	assert.False(t, ok, "nothing should be written when there is nothing to propagate")
}

func TestPropagator_ExtractWithoutHeader(t *testing.T) {
	p := NewContextPropagator()

	ctx, err := p.Extract(context.Background(), newTestHeader())
	require.NoError(t, err)

	assert.Nil(t, ctx.Value(PropagateKey), "no header means no propagated values")
}

func TestPropagator_ExtractIgnoresUnrelatedHeaderKeys(t *testing.T) {
	p := NewContextPropagator()
	header := newTestHeader()
	header.Set("some.other.key", malformedPayload())

	ctx, err := p.Extract(context.Background(), header)
	require.NoError(t, err)

	assert.Nil(t, ctx.Value(PropagateKey))
}

func TestPropagator_ExtractMalformedPayload(t *testing.T) {
	p := NewContextPropagator()
	header := newTestHeader()
	header.Set(HeaderKey, malformedPayload())

	ctx, err := p.Extract(context.Background(), header)

	require.Error(t, err, "an undecodable payload must be reported, not silently dropped")
	assert.Nil(t, ctx.Value(PropagateKey))
}

func TestPropagator_InjectFromWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]any
		setValues bool
		wantKey   bool
	}{
		{
			name:      "writes propagated values",
			values:    map[string]any{CorrelationID: "corr-456"},
			setValues: true,
			wantKey:   true,
		},
		{
			name:      "writes nothing when there is nothing to propagate",
			setValues: false,
			wantKey:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewContextPropagator()
			header := newTestHeader()

			workflowFn := func(ctx workflow.Context) error {
				if tc.setValues {
					ctx = workflow.WithValue(ctx, PropagateKey, tc.values)
				}

				return p.InjectFromWorkflow(ctx, header)
			}

			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.ExecuteWorkflow(workflowFn)

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())

			_, ok := header.Get(HeaderKey)
			require.Equal(t, tc.wantKey, ok)

			if !tc.wantKey {
				return
			}

			// The injected payload must be readable by the non-workflow side.
			ctx, err := p.Extract(context.Background(), header)
			require.NoError(t, err)
			assert.Equal(t, tc.values, ctx.Value(PropagateKey))
		})
	}
}

func TestPropagator_ExtractToWorkflow(t *testing.T) {
	values := map[string]any{
		CorrelationID: "corr-789",
		"tenant":      "acme",
	}

	populated := newTestHeader()
	require.NoError(t, NewContextPropagator().Inject(
		context.WithValue(context.Background(), PropagateKey, values), populated,
	))

	malformed := newTestHeader()
	malformed.Set(HeaderKey, malformedPayload())

	tests := []struct {
		name    string
		header  *testHeader
		want    map[string]any
		wantErr bool
	}{
		{name: "propagated values are readable from the workflow context", header: populated, want: values},
		{name: "no header leaves the workflow context untouched", header: newTestHeader()},
		{name: "malformed payload is reported", header: malformed, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewContextPropagator()

			workflowFn := func(ctx workflow.Context) (map[string]any, error) {
				ctx, err := p.ExtractToWorkflow(ctx, tc.header)
				if err != nil {
					return nil, err
				}

				values, _ := ctx.Value(PropagateKey).(map[string]any)

				return values, nil
			}

			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.ExecuteWorkflow(workflowFn)

			require.True(t, env.IsWorkflowCompleted())

			if tc.wantErr {
				assert.Error(t, env.GetWorkflowError())

				return
			}

			require.NoError(t, env.GetWorkflowError())

			var got map[string]any
			require.NoError(t, env.GetWorkflowResult(&got))
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestPropagator_WorkflowContextIsPopulatedByTemporal exercises the propagator
// through the SDK itself rather than by calling it directly.
func TestPropagator_WorkflowContextIsPopulatedByTemporal(t *testing.T) {
	values := map[string]any{CorrelationID: "corr-sdk"}

	written := newTestHeader()
	require.NoError(t, NewContextPropagator().Inject(
		context.WithValue(context.Background(), PropagateKey, values), written,
	))

	workflowFn := func(ctx workflow.Context) (map[string]any, error) {
		propagated, _ := ctx.Value(PropagateKey).(map[string]any)

		return propagated, nil
	}

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetContextPropagators([]workflow.ContextPropagator{NewContextPropagator()})
	env.SetHeader(&commonpb.Header{Fields: written.fields})
	env.ExecuteWorkflow(workflowFn)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var got map[string]any
	require.NoError(t, env.GetWorkflowResult(&got))
	assert.Equal(t, values, got)
}

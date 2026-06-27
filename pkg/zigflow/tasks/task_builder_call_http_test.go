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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/activity"
)

func TestParseHTTPArguments(t *testing.T) {
	state := utils.NewState()
	state.Env["token"] = "abc-123"
	state.Data["flag"] = true

	task := &model.CallHTTP{
		Call: "http",
		With: model.HTTPArguments{
			Method:   "GET",
			Endpoint: model.NewEndpoint("https://example.com"),
			Headers: model.NewObjectOrRuntimeExpr(map[string]any{
				// #nosec G101 -- DSL expression, not a hardcoded credential. Value is a JQ expression resolved at runtime from environment variables.
				"X-Token": "${ $env.token }",
			}),
			Query: model.NewObjectOrRuntimeExpr(map[string]any{
				"debug": testConstDataFlag,
			}),
		},
	}

	got, err := activities.ParseHTTPArguments(task, state)
	assert.NoError(t, err)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, "https://example.com", got.Endpoint.String())

	headers, ok := got.Headers.AsStringOrMap().(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "abc-123", headers["X-Token"])

	query, ok := got.Query.AsStringOrMap().(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, true, query["debug"])
}

func TestParseOutput(t *testing.T) {
	httpResp := activities.HTTPResponse{
		StatusCode: 200,
		Content: map[string]any{
			testConstMessage: "ok",
		},
	}
	raw := []byte("payload")

	tests := []struct {
		name       string
		outputType string
		expect     any
	}{
		{
			name:       "raw response returns base64 string",
			outputType: "raw",
			expect:     base64.StdEncoding.EncodeToString(raw),
		},
		{
			name:       "response returns metadata structure",
			outputType: "response",
			expect:     httpResp,
		},
		{
			name:       "default returns parsed content",
			outputType: "",
			expect:     httpResp.Content,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := activities.ParseOutput(tc.outputType, httpResp, raw)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func newTestHTTPTask() *model.CallHTTP {
	return &model.CallHTTP{
		Call: "http",
		With: model.HTTPArguments{
			Method:   "GET",
			Endpoint: model.NewEndpoint("https://example.com"),
		},
	}
}

func TestCallHTTPTaskBuilderRegistersPerTaskActivityName(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-http-metrics"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-http-metrics.fetchData",
		}).
		Once()

	b, err := NewCallHTTPTaskBuilder(w, newTestHTTPTask(), "fetchData", doc, testEvents, nil)
	assert.NoError(t, err)

	_, err = b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

func TestCallHTTPTaskBuilderRegistersOncePerWorker(t *testing.T) {
	assertRegistersOncePerWorker(t, "wf-http-dedup", "authenticate",
		func(w *WorkflowRegistryMock, doc *model.Workflow, taskName string) (TaskBuilder, error) {
			return NewCallHTTPTaskBuilder(w, newTestHTTPTask(), taskName, doc, testEvents, nil)
		})
}

func TestCallHTTPTaskBuilderBuildWithoutWorker(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-http-nil-worker"}}

	b, err := NewCallHTTPTaskBuilder(nil, newTestHTTPTask(), "step", doc, testEvents, nil)
	assert.NoError(t, err)

	fn, err := b.Build()
	assert.NoError(t, err)
	assert.NotNil(t, fn)
}

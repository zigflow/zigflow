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
	"encoding/base64"
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestCallHTTPTaskBuilderEvaluatesWithBeforeActivity(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	var captured *model.CallHTTP
	env.RegisterActivityWithOptions(func(_ context.Context, task *model.CallHTTP, _ any, _ *utils.State) (any, error) {
		captured = task
		return map[string]any{testConstOK: true}, nil
	}, activity.RegisterOptions{Name: "CallHTTPActivity"})

	task := &model.CallHTTP{
		Call: "http",
		With: model.HTTPArguments{
			Method:   "GET",
			Endpoint: model.NewEndpoint("https://example.com"),
			Headers: map[string]string{
				// #nosec G101 -- DSL expression, not a hardcoded credential.
				"X-Token": "${ $env.token }",
			},
			Query: map[string]any{
				"debug": testConstDataFlag,
			},
		},
	}

	b, err := NewCallHTTPTaskBuilder(nil, task, "httpTask", nil, testEvents, nil)
	assert.NoError(t, err)

	fn, err := b.Build()
	assert.NoError(t, err)

	workflowFunc := func(ctx workflow.Context) (any, error) {
		state := utils.NewState()
		state.Env["token"] = "abc-123"
		state.Data["flag"] = true
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return fn(ctx, nil, state)
	}

	env.ExecuteWorkflow(workflowFunc)

	assert.NoError(t, env.GetWorkflowError())
	require.NotNil(t, captured)
	assert.Equal(t, "abc-123", captured.With.Headers["X-Token"])
	assert.Equal(t, true, captured.With.Query["debug"])
	assert.NotContains(t, captured.With.Headers["X-Token"], "${")
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

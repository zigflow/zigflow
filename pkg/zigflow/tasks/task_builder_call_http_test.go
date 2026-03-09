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
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
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
			Headers: map[string]string{
				// #nosec G101 -- DSL expression, not a hardcoded credential. Value is a JQ expression resolved at runtime from environment variables.
				"X-Token": "${ $env.token }",
			},
			Query: map[string]any{
				"debug": "${ $data.flag }",
			},
		},
	}

	got, err := activities.ParseHTTPArguments(task, state)
	assert.NoError(t, err)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, "https://example.com", got.Endpoint.String())
	assert.Equal(t, "abc-123", got.Headers["X-Token"])
	assert.Equal(t, true, got.Query["debug"])
}

func TestParseOutput(t *testing.T) {
	httpResp := activities.HTTPResponse{
		StatusCode: 200,
		Content: map[string]any{
			"message": "ok",
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

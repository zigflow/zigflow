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

package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

// --- output format validation ---

func TestGetSchema_EmptyOutputDefaultsToJSON(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.GetSchema(context.Background(), nil, GetSchemaInput{})
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.NotEmpty(t, out.Schema)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out.Schema), &parsed), "schema must be valid JSON")

	id, _ := parsed["$id"].(string)
	assert.Contains(t, id, ".json", "schema ID must use resolved json format")
}

func TestGetSchema_JSONOutput(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.GetSchema(context.Background(), nil, GetSchemaInput{Output: "json"})
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.NotEmpty(t, out.Schema)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out.Schema), &parsed), "schema must be valid JSON")

	id, _ := parsed["$id"].(string)
	assert.Contains(t, id, ".json", "schema ID must use json format")
}

func TestGetSchema_YAMLOutput(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.GetSchema(context.Background(), nil, GetSchemaInput{Output: "yaml"})
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.NotEmpty(t, out.Schema)

	var parsed map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(out.Schema), &parsed), "schema must be valid YAML")

	// $id key is present in the raw schema string since yaml.Unmarshal does not
	// map $ prefixed keys into a plain Go map reliably.
	assert.Contains(t, out.Schema, "schema.yaml", "schema ID must use yaml format")
}

func TestGetSchema_InvalidOutput_InputError(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{"yml shorthand", "yml"},
		{"xml", "xml"},
		{"with surrounding spaces", " foo "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestMCP()
			_, out, err := m.GetSchema(context.Background(), nil, GetSchemaInput{Output: tc.output})
			require.NoError(t, err)
			require.Len(t, out.Errors, 1)
			assert.Equal(t, "input", out.Errors[0].Stage)
			assert.NotEmpty(t, out.Errors[0].Message)
			assert.Empty(t, out.Schema, "schema must be empty on input error")
		})
	}
}

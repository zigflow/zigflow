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
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Declared as a const because goconst trips at three occurrences.
const getExampleToolName = "get_example"

func toolSession(t *testing.T) *mcp.ClientSession {
	t.Helper()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	_, err := testServer(t).Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: testVersion}, nil).
		Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, session.Close())
	})

	return session
}

// The schemas as actually advertised to a client must not use the array form
// of "type" anywhere, at any nesting depth.
func TestOutputSchemas_HaveNoNullableTypeArrays(t *testing.T) {
	tools, err := toolSession(t).ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, tools.Tools)

	for _, tool := range tools.Tools {
		schema, err := json.Marshal(tool.OutputSchema)
		require.NoError(t, err)

		assert.NotContains(t, string(schema), `["null","array"]`, "tool %q", tool.Name)
	}
}

// list_examples nests a slice inside another slice's element schema, which
// only a recursive rewrite reaches.
func TestOutputSchemaFor_RewritesNestedSlices(t *testing.T) {
	examples := outputSchemaFor[ListExamplesOutput]().Properties["examples"]

	require.Len(t, examples.AnyOf, 2)
	assert.Equal(t, "null", examples.AnyOf[0].Type)
	assert.Equal(t, "array", examples.AnyOf[1].Type)
	assert.Empty(t, examples.Types)
	assert.Nil(t, examples.Items)

	tags := examples.AnyOf[1].Items.Properties["tags"]
	require.Len(t, tags.AnyOf, 2)
	assert.Equal(t, "null", tags.AnyOf[0].Type)
	assert.Equal(t, "array", tags.AnyOf[1].Type)
}

// Non-slice properties and the required list must survive the rewrite.
func TestOutputSchemaFor_LeavesNonSlicesAlone(t *testing.T) {
	schema := outputSchemaFor[ValidateWorkflowOutput]()

	assert.Equal(t, "boolean", schema.Properties["valid"].Type)
	assert.Empty(t, schema.Properties["valid"].AnyOf)
	assert.Equal(t, []string{"valid"}, schema.Required)
}

// The rewritten schema must still accept every real output shape. The SDK
// validates the handler's result against OutputSchema server-side, so a
// violation comes back as an error result rather than a Go error.
func TestGetExample_OutputValidatesAgainstSchema(t *testing.T) {
	session := toolSession(t)

	tests := []struct {
		name        string
		exampleName string
		wantTags    bool
		wantErrors  bool
	}{
		{name: "nil tags and nil errors", exampleName: "basic"},
		{name: "populated tags", exampleName: "signal", wantTags: true},
		{name: "populated errors", exampleName: "no-such-example", wantErrors: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
				Name:      getExampleToolName,
				Arguments: map[string]any{"name": test.exampleName},
			})
			require.NoError(t, err)
			require.False(t, res.IsError, "%v", res.Content)

			raw, err := json.Marshal(res.StructuredContent)
			require.NoError(t, err)

			var out GetExampleOutput
			require.NoError(t, json.Unmarshal(raw, &out))

			// Assert the case really exercised what it claims, so the
			// coverage can't silently lapse if the catalog changes.
			assert.Equal(t, test.wantTags, len(out.Tags) > 0)
			assert.Equal(t, test.wantErrors, len(out.Errors) > 0)
		})
	}
}

// Every tool's real output must still validate against its rewritten schema.
// The SDK validates the handler's result server-side, so a violation comes
// back as an error result rather than a Go error.
func TestOutputSchemas_ToolCallsValidate(t *testing.T) {
	session := toolSession(t)

	tests := []struct {
		name string
		tool string
		args map[string]any
	}{
		{name: "schema", tool: "get_schema", args: map[string]any{}},
		{name: "examples", tool: "list_examples", args: map[string]any{}},
		{name: "task docs", tool: "get_task_docs", args: map[string]any{"task_type": "call"}},
		{name: "task docs errors", tool: "get_task_docs", args: map[string]any{"task_type": "nope"}},
		{name: "workflow errors", tool: "validate_workflow", args: map[string]any{"yaml": "bad: ["}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
				Name:      test.tool,
				Arguments: test.args,
			})
			require.NoError(t, err)
			assert.False(t, res.IsError, "%v", res.Content)
		})
	}
}

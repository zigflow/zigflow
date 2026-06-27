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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zigflowexamples "github.com/zigflow/zigflow/examples"
)

func explain(t *testing.T, msg string) ExplainErrorOutput {
	t.Helper()

	out, err := explainError(zigflowexamples.EmbeddedFS, msg)
	require.NoError(t, err)

	return out
}

// rawSchemaMessage mirrors the message validate_workflow returns for a schema
// failure, so the matchers are tested against text Zigflow really produces.
const rawTaskQueueMessage = `workflow failed schema validation: validating https://zigflow.dev/schema.json: ` +
	`validating /properties/document: validating /properties/document/properties/taskQueue: ` +
	`pattern: "my_queue.v2" does not match regular expression "^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$"`

const rawTaskOneOfMessage = `workflow failed schema validation: validating https://zigflow.dev/schema.json: ` +
	`validating /properties/do: validating /$defs/taskList: validating /$defs/task: ` +
	`oneOf: did not validate against any of [<anonymous schema>]`

// --- input validation ---

func TestExplainError_EmptyInput(t *testing.T) {
	out := explain(t, "")
	assert.False(t, out.Matched)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, stageInput, out.Errors[0].Stage)
	assert.Equal(t, "error_message is required", out.Errors[0].Message)
}

func TestExplainError_WhitespaceTreatedAsAbsent(t *testing.T) {
	out := explain(t, "   \n\t ")
	require.Len(t, out.Errors, 1)
	assert.Equal(t, stageInput, out.Errors[0].Stage)
}

// --- recognised errors: explanation, suggested fix, documentation ---

func TestExplainError_RFC1123_RawSchemaMessage(t *testing.T) {
	out := explain(t, rawTaskQueueMessage)
	assert.True(t, out.Matched)
	assert.Equal(t, rawTaskQueueMessage, out.Error)
	assert.Contains(t, out.Explanation, "RFC 1123")
	assert.NotEmpty(t, out.SuggestedFix)
	assert.Contains(t, out.Documentation, "https://zigflow.dev/docs/concepts/common-mistakes")
}

func TestExplainError_RFC1123_NormalisedMessage(t *testing.T) {
	// The form used in the issue description; matchers must be tolerant of it.
	out := explain(t, "schema: /document/taskQueue must match RFC1123 DNS label")
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "RFC 1123")
	assert.NotEmpty(t, out.SuggestedFix)
}

func TestExplainError_NonDeterministicExpression(t *testing.T) {
	msg := `non-deterministic expression:
  - do[0].w.seconds: expression "${ timestamp }" uses non-deterministic value "timestamp"`
	out := explain(t, msg)
	assert.True(t, out.Matched)
	assert.Contains(t, strings.ToLower(out.Explanation), "set task")
	assert.NotEmpty(t, out.SuggestedFix)
	assert.Contains(t, out.RelatedTaskTypes, "set")
}

func TestExplainError_InvalidRuntimeVariable(t *testing.T) {
	msg := `invalid runtime expression:
  - do[0].s.set.a: failed to compile expression "$workflow.id": variable not defined: $workflow`
	out := explain(t, msg)
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "$context")
	assert.NotEmpty(t, out.SuggestedFix)
}

func TestExplainError_InvalidExpressionSyntax(t *testing.T) {
	msg := `invalid runtime expression:
  - do[0].s.set.a: failed to parse expression "@@@": unexpected token "@"`
	out := explain(t, msg)
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "${ ... }")
	assert.NotEmpty(t, out.SuggestedFix)
}

func TestExplainError_UnsupportedCallType(t *testing.T) {
	out := explain(t, "unsupported call type 'soap' for task 'fetch'")
	assert.True(t, out.Matched)
	assert.Contains(t, out.RelatedTaskTypes, "call")
	require.NotEmpty(t, out.Examples)
	assert.Equal(t, "activity-call", out.Examples[0].Name)
}

func TestExplainError_UnsupportedTaskType(t *testing.T) {
	out := explain(t, "unsupported task type '*model.HTTPTask' for task 'fetch'")
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "eleven task types")
}

func TestExplainError_InvalidDuration(t *testing.T) {
	out := explain(t, "duration field minute has unsupported type string")
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "plural")
	assert.Contains(t, out.RelatedTaskTypes, "wait")
}

func TestExplainError_UnknownDocumentField(t *testing.T) {
	msg := `workflow failed schema validation: validating /properties/document: ` +
		`validating /properties/document/unevaluatedProperties: not: validated against <anonymous schema>`
	out := explain(t, msg)
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "closed")
}

func TestExplainError_EndpointNotURL(t *testing.T) {
	out := explain(t, "with.url is not a recognised field; use endpoint")
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "endpoint")
	assert.Contains(t, out.RelatedTaskTypes, "call")
}

func TestExplainError_ExportVsOutput(t *testing.T) {
	out := explain(t, "the export block does not merge into $context")
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "$context")
}

func TestExplainError_GenericTaskOneOf(t *testing.T) {
	// The raw schema message for many distinct task mistakes; must resolve to
	// the catch-all rather than a wrong specific rule or the fallback.
	out := explain(t, rawTaskOneOfMessage)
	assert.True(t, out.Matched)
	assert.Contains(t, out.Explanation, "did not match any known task type")
	assert.NotEmpty(t, out.SuggestedFix)
}

func TestExplainError_UnsupportedDSL(t *testing.T) {
	out := explain(t, "unsupported dsl version: 9.9.9")
	assert.True(t, out.Matched)
	assert.Contains(t, out.SuggestedFix, "get_schema")
}

// --- graceful fallback ---

func TestExplainError_UnknownError(t *testing.T) {
	out := explain(t, "something completely unrelated to any rule xyzzy")
	assert.False(t, out.Matched)
	assert.Empty(t, out.Errors)
	assert.Contains(t, out.Explanation, "No curated explanation is available")
	// The fallback still echoes the input and offers a starting point.
	assert.Equal(t, "something completely unrelated to any rule xyzzy", out.Error)
	assert.Contains(t, out.Documentation, "https://zigflow.dev/docs/concepts/common-mistakes")
	assert.Empty(t, out.SuggestedFix, "fallback must not invent a fix")
}

// --- drift guards: curated data must reference real things ---

// TestExplainError_DocumentationLinksAreCanonical guards every documentation URL
// in the table and the fallback against the canonical docs base, so a malformed
// or relative link is caught.
func TestExplainError_DocumentationLinksAreCanonical(t *testing.T) {
	check := func(links []string) {
		for _, link := range links {
			assert.Truef(t, strings.HasPrefix(link, docsBaseURL),
				"documentation link %q must start with %q", link, docsBaseURL)
			assert.NotEqual(t, docsBaseURL, link, "documentation link must reference a page")
		}
	}

	for i := range explanationRules {
		rule := &explanationRules[i]
		assert.NotEmptyf(t, rule.documentation, "rule %q must provide documentation", rule.id)
		check(rule.documentation)
	}

	fallback := explain(t, "totally unknown error message")
	check(fallback.Documentation)
}

// TestExplainError_RelatedTaskTypesAreReal guards that every related task type a
// rule names is a real Zigflow task type from the schema.
func TestExplainError_RelatedTaskTypesAreReal(t *testing.T) {
	known := make(map[string]struct{})
	for _, k := range schemaTaskTypes(t) {
		known[k] = struct{}{}
	}

	for i := range explanationRules {
		rule := &explanationRules[i]
		for _, tt := range rule.relatedTaskTypes {
			_, ok := known[tt]
			assert.Truef(t, ok, "rule %q references unknown task type %q", rule.id, tt)
		}
	}
}

// TestExplainError_ExampleNamesResolve guards that every curated example name
// resolves to a bundled, validated example.
func TestExplainError_ExampleNamesResolve(t *testing.T) {
	for i := range explanationRules {
		rule := &explanationRules[i]
		if len(rule.exampleNames) == 0 {
			continue
		}
		refs, err := resolveExampleRefs(zigflowexamples.EmbeddedFS, rule.exampleNames)
		require.NoErrorf(t, err, "rule %q references a missing example", rule.id)
		require.Len(t, refs, len(rule.exampleNames))
		for _, ref := range refs {
			assert.NotEmptyf(t, ref.Title, "example %q must carry a title", ref.Name)
		}
	}
}

// TestExplainError_HandlerWiring exercises the exported handler end to end.
func TestExplainError_HandlerWiring(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ExplainError(context.Background(), nil, ExplainErrorInput{
		ErrorMessage: rawTaskQueueMessage,
	})
	require.NoError(t, err)
	assert.True(t, out.Matched)
	assert.NotEmpty(t, out.Explanation)
}

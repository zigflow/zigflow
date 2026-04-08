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

package schema

import (
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -- helpers ------------------------------------------------------------------

// schemaRefs collects all $ref strings from a slice of schemas.
func schemaRefs(schemas []*jsonschema.Schema) []string {
	refs := make([]string, 0, len(schemas))
	for _, s := range schemas {
		refs = append(refs, s.Ref)
	}
	return refs
}

// callConst returns the value of the "call" const from a callTask OneOf entry.
// Each entry is an object with AllOf[1].Properties["call"].Const.
func callConst(s *jsonschema.Schema) string {
	if len(s.AllOf) < 2 {
		return ""
	}
	props := s.AllOf[1].Properties
	if props == nil {
		return ""
	}
	call, ok := props["call"]
	if !ok || call.Const == nil {
		return ""
	}
	v, ok := (*call.Const).(string)
	if !ok {
		return ""
	}
	return v
}

// runVariantTitles returns the Title fields from the OneOf list inside
// runTaskDefinition's "run" property.
func runVariantTitles() []string {
	run := runTaskDefinition.AllOf[1].Properties["run"]
	titles := make([]string, 0, len(run.OneOf))
	for _, v := range run.OneOf {
		titles = append(titles, v.Title)
	}
	return titles
}

// catchProperty returns the named property from tryTask's catch sub-schema,
// navigating AllOf[1].Properties["catch"].Properties[name].
func catchProperty(name string) *jsonschema.Schema {
	catch := tryTaskDefinition.AllOf[1].Properties["catch"]
	if catch == nil || catch.Properties == nil {
		return nil
	}
	return catch.Properties[name]
}

// -- tests --------------------------------------------------------------------

// TestBuildSchema verifies the schema builds successfully and resolves without
// errors. This is the top-level smoke test.
func TestBuildSchema(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)
	require.NotNil(t, s)
}

// TestBuildDefinitionsKeys verifies that buildDefinitions returns exactly the
// expected set of supported definitions and that intentionally unsupported
// definitions are absent.
func TestBuildDefinitionsKeys(t *testing.T) {
	defs := buildDefinitions()

	expected := []string{
		"callTask",
		"commonMetadata",
		"containerLifetime",
		"doTask",
		"documentMetadata",
		"duration",
		"endpoint",
		"error",
		"eventConsumptionStrategy",
		"eventFilter",
		"eventProperties",
		"export",
		"externalResource",
		"flowDirective",
		"forkTask",
		"forTask",
		"input",
		"listenTask",
		"output",
		"raiseTask",
		"runTask",
		"runtimeExpression",
		"schema",
		"setTask",
		"subscriptionIterator",
		"switchTask",
		"task",
		"taskBase",
		"taskList",
		"taskMetadata",
		"timeout",
		"tryTask",
		"uriTemplate",
		"waitTask",
	}

	for _, key := range expected {
		assert.Contains(t, defs, key, "buildDefinitions() should contain %q", key)
	}

	// emitTask is intentionally unsupported.
	assert.NotContains(t, defs, "emitTask", "emitTask must not be present")
}

// TestTaskDefinitionOneOf verifies that taskDefinition references exactly the
// supported task types and excludes unsupported ones.
func TestTaskDefinitionOneOf(t *testing.T) {
	refs := schemaRefs(taskDefinition.OneOf)

	supported := []string{
		SchemaRef("callTask"),
		SchemaRef("doTask"),
		SchemaRef("forTask"),
		SchemaRef("forkTask"),
		SchemaRef("listenTask"),
		SchemaRef("raiseTask"),
		SchemaRef("runTask"),
		SchemaRef("setTask"),
		SchemaRef("switchTask"),
		SchemaRef("tryTask"),
		SchemaRef("waitTask"),
	}

	for _, ref := range supported {
		assert.Contains(t, refs, ref, "taskDefinition.OneOf should reference %q", ref)
	}

	assert.NotContains(t, refs, SchemaRef("emitTask"), "emitTask must not appear in taskDefinition.OneOf")
}

// TestDurationDefinitionShape verifies that duration supports only the object
// form and does not include the runtimeExpression or ISO 8601 string branches
// that exist in the upstream Serverless Workflow schema.
func TestDurationDefinitionShape(t *testing.T) {
	d := durationDefinition

	assert.Len(t, d.OneOf, 1, "duration must have exactly one OneOf branch (object form only)")

	branch := d.OneOf[0]
	assert.Equal(t, "object", branch.Type, "the single duration branch must be an object")

	for _, prop := range []string{"days", "hours", "minutes", "seconds", "milliseconds"} {
		assert.Contains(t, branch.Properties, prop, "duration object must have property %q", prop)
	}

	// Confirm the two unsupported branches are absent.
	for _, s := range d.OneOf {
		assert.Empty(t, s.Ref, "duration must not contain a $ref branch (no runtimeExpression)")
		assert.Empty(t, s.Pattern, "duration must not contain a pattern branch (no ISO 8601 literal)")
	}
}

// TestTryTaskDefinitionCatch verifies that tryTask requires both "try" and
// "catch", and that catch requires "do" referencing taskList.
func TestTryTaskDefinitionCatch(t *testing.T) {
	assert.Contains(t, tryTaskDefinition.Required, "try")
	assert.Contains(t, tryTaskDefinition.Required, "catch")

	catch := tryTaskDefinition.AllOf[1].Properties["catch"]
	require.NotNil(t, catch, "catch property must be present")

	assert.Equal(t, "object", catch.Type)
	assert.Contains(t, catch.Required, "do")

	doField := catchProperty("do")
	require.NotNil(t, doField, "catch.do must be present")
	assert.Equal(t, SchemaRef("taskList"), doField.Ref, "catch.do must reference taskList")
}

// TestCallTaskDefinitionVariants verifies that callTaskDefinition contains
// exactly the supported call variants and not the unsupported ones.
func TestCallTaskDefinitionVariants(t *testing.T) {
	consts := make([]string, 0, len(callTaskDefinition.OneOf))
	for _, s := range callTaskDefinition.OneOf {
		consts = append(consts, callConst(s))
	}

	for _, want := range []string{"activity", "grpc", "http"} {
		assert.True(t, slices.Contains(consts, want), "callTask should support %q", want)
	}

	for _, absent := range []string{"asyncapi", "openapi", "mcp", "a2a"} {
		assert.False(t, slices.Contains(consts, absent), "callTask must not support %q", absent)
	}
}

// TestRunTaskDefinitionVariants verifies that runTaskDefinition includes all
// four supported run modes.
func TestRunTaskDefinitionVariants(t *testing.T) {
	titles := runVariantTitles()

	for _, want := range []string{"RunContainer", "RunScript", "RunShell", "RunWorkflow"} {
		assert.True(t, slices.Contains(titles, want), "runTask should have variant %q", want)
	}
}

// TestSchemaDefinitionInlineOnly verifies that schemaDefinition supports only
// the inline document form and not the external resource form.
func TestSchemaDefinitionInlineOnly(t *testing.T) {
	s := schemaDefinition

	assert.Len(t, s.OneOf, 1, "schema must have exactly one OneOf branch (inline only)")

	branch := s.OneOf[0]
	assert.Contains(t, branch.Required, "document", "inline schema branch must require 'document'")

	// The external branch would require "resource"; verify it is absent.
	for _, b := range s.OneOf {
		assert.NotContains(t, b.Required, "resource", "schema must not have an external resource branch")
	}
}

// TestRuntimeExpressionPattern verifies that runtimeExpressionDefinition uses
// the expected pattern constant.
func TestRuntimeExpressionPattern(t *testing.T) {
	assert.Equal(t, runtimeExpressionString, runtimeExpressionDefinition.Pattern)
}

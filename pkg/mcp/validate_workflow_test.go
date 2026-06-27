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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validWorkflowYAML = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world
`

// structFailureYAML passes JSON schema validation but fails struct validation:
// ForTaskConfiguration.In has validate:"required", which rejects an empty
// string even though the schema places no minLength constraint on for.in.
const structFailureYAML = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - loop:
      for:
        in: ""
        each: item
      do:
        - step:
            set:
              x: "1"
`

func newTestMCP() *MCP {
	return &MCP{version: "development"}
}

// --- input validation ---

func TestValidateWorkflow_EmptyYAML(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "input", out.Errors[0].Stage)
	assert.Equal(t, "yaml is required", out.Errors[0].Message)
}

func TestValidateWorkflow_WhitespaceYAMLTreatedAsAbsent(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: "   \n\t  ",
	})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "input", out.Errors[0].Stage)
}

// --- valid input ---

func TestValidateWorkflow_ValidYAML(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: validWorkflowYAML,
	})
	require.NoError(t, err)
	assert.True(t, out.Valid)
	assert.Empty(t, out.Errors)
}

// --- parse stage ---

func TestValidateWorkflow_InvalidYAMLSyntax_ParseStage(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: "invalid content: [",
	})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "parse", out.Errors[0].Stage)
	assert.NotEmpty(t, out.Errors[0].Message)
}

// --- schema stage ---

func TestValidateWorkflow_SchemaMissingRequiredFields(t *testing.T) {
	m := newTestMCP()
	// Missing required fields: workflowType, taskQueue
	const badYAML = `document:
  dsl: 1.0.0
  version: 0.0.1
do:
  - step:
      set:
        hello: world
`
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{YAML: badYAML})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "schema", out.Errors[0].Stage)
	assert.NotEmpty(t, out.Errors[0].Message)
}

// invalidTaskQueueYAML fails JSON schema validation because taskQueue does not
// match the required pattern. This is a recognised validation error.
const invalidTaskQueueYAML = `document:
  dsl: 1.0.0
  taskQueue: "Not A Valid Queue"
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world
`

func TestValidateWorkflow_RecognisedSchemaError_ExposesCodeAndDocumentation(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: invalidTaskQueueYAML,
	})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)

	got := out.Errors[0]
	assert.Equal(t, "schema", got.Stage)
	assert.Equal(t, "$.document.taskQueue", got.Path)
	assert.Equal(t, "ERR_INVALID_TASK_QUEUE", got.Code)
	assert.Equal(t, "https://zigflow.dev/errors/invalid-task-queue", got.Documentation)
	// The underlying validation message must remain unchanged: it is the raw
	// schema failure, not a rewritten or enriched string.
	assert.Contains(t, got.Message, "does not match")
	assert.NotContains(t, got.Message, "ERR_INVALID_TASK_QUEUE")
	assert.NotContains(t, got.Message, "zigflow.dev/errors")
}

func TestValidateWorkflow_UnrecognisedSchemaError_OmitsDocumentation(t *testing.T) {
	m := newTestMCP()
	// Missing required fields fail at the parent ($.document) location, which is
	// not a recognised field-level error, so no code or documentation applies.
	const badYAML = `document:
  dsl: 1.0.0
  version: 0.0.1
do:
  - step:
      set:
        hello: world
`
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{YAML: badYAML})
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)

	got := out.Errors[0]
	assert.Equal(t, "schema", got.Stage)
	assert.Empty(t, got.Code)
	assert.Empty(t, got.Documentation)
	assert.NotEmpty(t, got.Message)
}

// --- expression stage ---

// nonDeterministicYAML passes schema validation but uses a non-deterministic
// expression (timestamp) outside a Set body, which is rejected by the
// determinism pass.
const nonDeterministicYAML = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - waitForCooldown:
      wait:
        seconds: ${ timestamp }
`

// invalidExpressionYAML passes schema validation but contains jq that cannot be
// parsed, inside a Set body.
const invalidExpressionYAML = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - capture:
      set:
        id: ${ @@@ }
`

func TestValidateWorkflow_NonDeterministicExpression_ExpressionStage(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: nonDeterministicYAML,
	})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "expression", out.Errors[0].Stage,
		"non-deterministic expression must not be reported as a parse failure")
	assert.NotEmpty(t, out.Errors[0].Message)
}

func TestValidateWorkflow_InvalidRuntimeExpression_ExpressionStage(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: invalidExpressionYAML,
	})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "expression", out.Errors[0].Stage,
		"invalid runtime expression must not be reported as a parse failure")
	assert.NotEmpty(t, out.Errors[0].Message)
}

// --- struct stage ---

func TestValidateWorkflow_StructValidationFailure_StructStage(t *testing.T) {
	m := newTestMCP()
	_, out, err := m.ValidateWorkflow(context.Background(), nil, ValidateWorkflowInput{
		YAML: structFailureYAML,
	})
	require.NoError(t, err)
	assert.False(t, out.Valid)
	require.NotEmpty(t, out.Errors)
	assert.Equal(t, "struct", out.Errors[0].Stage)
	assert.NotEmpty(t, out.Errors[0].Message, "translated validator message must be present")
}

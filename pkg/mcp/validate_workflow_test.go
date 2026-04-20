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

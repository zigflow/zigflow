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

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
)

const validWorkflowYAML = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowUnsupportedDSL = `document:
  dsl: 0.9.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowMissingWorkflowName = `document:
  dsl: 1.0.0
  taskQueue: default
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowLegacyName = `document:
  dsl: 1.0.0
  taskQueue: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowLegacyNamespace = `document:
  dsl: 1.0.0
  namespace: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowSingleNonDeterministic = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      if: ${ uuid }
      set:
        hello: world`

const workflowMultipleNonDeterministic = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - first:
      if: ${ uuid }
      set:
        hello: world
  - second:
      if: ${ timestamp }
      set:
        hello: world`

// writeTempWorkflow writes content to a temporary workflow file and returns its
// path, registering cleanup with the test.
func writeTempWorkflow(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// runValidate writes content to a temp file, runs the validate command against
// it with the given extra args, and returns the captured stdout and the
// command error.
func runValidate(t *testing.T, content string, extraArgs ...string) (string, error) {
	t.Helper()
	path := writeTempWorkflow(t, content)

	var buf bytes.Buffer
	cmd := newValidateCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(&buf)
	cmd.SetArgs(append([]string{path}, extraArgs...))

	err := cmd.Execute()
	return buf.String(), err
}

// runValidateJSON runs the validate command with --output-json against content,
// asserts the command failed and emitted an invalid result, and returns the
// decoded result for the caller to make kind-specific assertions on.
func runValidateJSON(t *testing.T, content string) utils.ValidationResult {
	t.Helper()
	out, err := runValidate(t, content, "--output-json")
	require.Error(t, err)

	var result utils.ValidationResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.False(t, result.Valid)
	return result
}

// TestValidateCmdDeterminismOutput verifies that determinism failures are
// rendered as concise, workflow-level validation failures focused on the
// location and offending expression — not as schema validation failures.
func TestValidateCmdDeterminismOutput(t *testing.T) {
	t.Run("single non-deterministic expression", func(t *testing.T) {
		out, err := runValidate(t, workflowSingleNonDeterministic)
		require.Error(t, err)

		assert.Contains(t, out, "is invalid")
		assert.Contains(t, out, "Non-deterministic expression")
		assert.Contains(t, out, "Location:")
		assert.Contains(t, out, "Expression:")
		assert.Contains(t, out, "${ uuid }")
		// The old, confusing wording must be gone.
		assert.NotContains(t, out, "Schema validation failed")
	})

	t.Run("multiple non-deterministic expressions", func(t *testing.T) {
		out, err := runValidate(t, workflowMultipleNonDeterministic)
		require.Error(t, err)

		assert.Contains(t, out, "Non-deterministic expressions found:")
		assert.Contains(t, out, "${ uuid }")
		assert.Contains(t, out, "${ timestamp }")
		// Each location is reported exactly once.
		assert.Equal(t, 1, strings.Count(out, "${ uuid }"))
		assert.Equal(t, 1, strings.Count(out, "${ timestamp }"))
	})

	t.Run("JSON output stays machine-readable", func(t *testing.T) {
		result := runValidateJSON(t, workflowSingleNonDeterministic)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "non_deterministic_expression", result.Errors[0].Key)
		assert.Contains(t, result.Errors[0].Message, "${ uuid }")
		assert.Contains(t, result.Errors[0].Path, "do[0]")
	})
}

// workflowInvalidExpression fails expression syntax validation: ${ @@@ } is not
// valid jq. It sits inside a set body, which is exempt from the determinism
// rule but not from parse validation.
const workflowInvalidExpression = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - capture:
      set:
        id: ${ @@@ }`

// workflowCompileInvalidExpression parses as valid jq but cannot compile: the
// bare identifier is an unregistered 0-arg function call. It sits in a set body
// to prove Set still enforces compile validation.
const workflowCompileInvalidExpression = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - capture:
      set:
        id: ${ definitely_not_registered }`

// TestValidateCmdInvalidExpressionOutput verifies that invalid runtime
// expressions are rendered as their own "Invalid runtime expression" failure
// (location, expression, error) and never mislabelled as non-deterministic.
func TestValidateCmdInvalidExpressionOutput(t *testing.T) {
	t.Run("human-readable by default", func(t *testing.T) {
		out, err := runValidate(t, workflowInvalidExpression)
		require.Error(t, err)

		assert.Contains(t, out, "is invalid")
		assert.Contains(t, out, "Invalid runtime expression")
		assert.Contains(t, out, "Location:")
		assert.Contains(t, out, "Expression:")
		assert.Contains(t, out, "Error:")
		assert.Contains(t, out, "${ @@@ }")
		// A parse failure must not be rendered as non-determinism.
		assert.NotContains(t, out, "Non-deterministic")
	})

	t.Run("compile-invalid expression in set body", func(t *testing.T) {
		out, err := runValidate(t, workflowCompileInvalidExpression)
		require.Error(t, err)

		assert.Contains(t, out, "Invalid runtime expression")
		assert.Contains(t, out, "${ definitely_not_registered }")
		// A compile failure inside a set body must not be rendered as
		// non-determinism, nor silently accepted.
		assert.NotContains(t, out, "Non-deterministic")
	})

	t.Run("JSON output stays machine-readable", func(t *testing.T) {
		result := runValidateJSON(t, workflowInvalidExpression)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "invalid_runtime_expression", result.Errors[0].Key)
		assert.Contains(t, result.Errors[0].Message, "${ @@@ }")
		assert.Contains(t, result.Errors[0].Path, "set.id")
	})
}

// workflowSchemaMissingDo fails JSON Schema validation: the required top-level
// "do" property is missing (e.g. the user wrote "do2").
const workflowSchemaMissingDo = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do2:
  - step:
      set:
        hello: world`

// workflowSchemaLegacyName fails JSON Schema validation deeper in the document:
// the required document.workflowType is missing (legacy document.name used).
const workflowSchemaLegacyName = `document:
  dsl: 1.0.0
  taskQueue: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

// TestValidateCmdSchemaOutput verifies that schema validation failures are
// rendered for humans by default (location + message under an "is invalid"
// heading) rather than as JSON/log-style output.
func TestValidateCmdSchemaOutput(t *testing.T) {
	t.Run("human-readable by default", func(t *testing.T) {
		out, err := runValidate(t, workflowSchemaMissingDo)
		require.Error(t, err)

		assert.Contains(t, out, "is invalid")
		assert.Contains(t, out, "Schema validation failed")
		assert.Contains(t, out, "Location:")
		assert.Contains(t, out, "Message:")
		// Must not leak the raw, log/JSON-style error envelope.
		assert.NotContains(t, out, "\"level\"")
		assert.NotContains(t, out, "validating https://")
	})

	t.Run("location points at the offending document path", func(t *testing.T) {
		out, err := runValidate(t, workflowSchemaLegacyName)
		require.Error(t, err)

		assert.Contains(t, out, "$.document")
	})

	t.Run("JSON output stays machine-readable", func(t *testing.T) {
		result := runValidateJSON(t, workflowSchemaMissingDo)
		require.NotEmpty(t, result.Errors)
		assert.Equal(t, "schema_validation", result.Errors[0].Key)
		assert.NotEmpty(t, result.Errors[0].Path)
		assert.NotEmpty(t, result.Errors[0].Message)
	})
}

func TestNewValidateCmd(t *testing.T) {
	tests := []struct {
		Name        string
		Content     string
		FilePath    string
		ExtraArgs   []string
		ExpectError bool
	}{
		{
			Name:    "valid workflow",
			Content: validWorkflowYAML,
		},
		{
			Name:      "valid workflow with JSON output",
			Content:   validWorkflowYAML,
			ExtraArgs: []string{"--output-json"},
		},
		{
			Name:        "non-existent file",
			FilePath:    "/nonexistent/path/workflow.yaml",
			ExpectError: true,
		},
		{
			Name:        "invalid YAML",
			Content:     "invalid content: [",
			ExpectError: true,
		},
		{
			Name:        "unsupported DSL version",
			Content:     workflowUnsupportedDSL,
			ExpectError: true,
		},
		{
			Name:        "workflow missing required workflowType field",
			Content:     workflowMissingWorkflowName,
			ExpectError: true,
		},
		{
			Name:        "legacy document.name field is rejected",
			Content:     workflowLegacyName,
			ExpectError: true,
		},
		{
			Name:        "legacy document.namespace field is rejected",
			Content:     workflowLegacyNamespace,
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			filePath := test.FilePath
			if filePath == "" {
				tmpDir, err := os.MkdirTemp("", "validate_test")
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, os.RemoveAll(tmpDir))
				}()

				filePath = filepath.Join(tmpDir, "workflow.yaml")
				err = os.WriteFile(filePath, []byte(test.Content), 0o600)
				assert.NoError(t, err)
			}

			cmd := newValidateCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(append([]string{filePath}, test.ExtraArgs...))

			err := cmd.Execute()

			if test.ExpectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

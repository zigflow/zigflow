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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workflowSchemaInvalid is structurally invalid: the document block is present
// but the required "name" field is missing.
const workflowSchemaInvalid = `document:
  dsl: 1.0.0
  namespace: default
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

// workflowSchemaUnknownField contains a field rejected by the schema:
// schedule.after is not supported by the runtime.
const workflowSchemaUnknownField = `document:
  dsl: 1.0.0
  namespace: default
  name: test
  version: 0.0.1
schedule:
  after: PT5M
do:
  - step:
      set:
        hello: world`

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestNewValidateSchemaCmd(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		filePath     string
		extraArgs    []string
		expectError  bool
		expectOutput string
	}{
		{
			name:         "valid workflow",
			content:      validWorkflowYAML,
			expectOutput: "is valid",
		},
		{
			name:         "valid workflow with JSON output",
			content:      validWorkflowYAML,
			extraArgs:    []string{"--output-json"},
			expectOutput: `"valid": true`,
		},
		{
			name:         "invalid workflow - missing required field",
			content:      workflowSchemaInvalid,
			expectError:  true,
			expectOutput: "Validation failed",
		},
		{
			name:         "invalid workflow with JSON output",
			content:      workflowSchemaInvalid,
			extraArgs:    []string{"--output-json"},
			expectError:  true,
			expectOutput: `"valid": false`,
		},
		{
			name:        "invalid workflow - unsupported field",
			content:     workflowSchemaUnknownField,
			expectError: true,
		},
		{
			name:        "invalid YAML",
			content:     "invalid: [unclosed bracket",
			expectError: true,
		},
		{
			name:        "missing file",
			filePath:    "/nonexistent/path/workflow.yaml",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filePath := test.filePath

			if filePath == "" {
				tmpDir, err := os.MkdirTemp("", "validate_schema_test")
				require.NoError(t, err)
				t.Cleanup(func() { assert.NoError(t, os.RemoveAll(tmpDir)) })

				filePath = writeTemp(t, tmpDir, "workflow.yaml", test.content)
			}

			var out bytes.Buffer

			cmd := newValidateSchemaCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetOut(&out)
			cmd.SetArgs(append([]string{filePath}, test.extraArgs...))

			err := cmd.Execute()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.expectOutput != "" {
				assert.Contains(t, out.String(), test.expectOutput)
			}
		})
	}
}

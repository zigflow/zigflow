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

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validWorkflowYAML = `document:
  dsl: 1.0.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowUnsupportedDSL = `document:
  dsl: 0.9.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

const workflowMissingName = `document:
  dsl: 1.0.0
  namespace: default
  version: 0.0.1
do:
  - step:
      set:
        hello: world`

// workflowScheduleAfterRejected has a field rejected by the JSON Schema
// (schedule.after is not supported) but accepted by the runtime validator,
// which ignores unknown YAML fields during struct parsing.
const workflowScheduleAfterRejected = `document:
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

// subCmdTestCase is the shared test case type used by runSubCmdTests.
type subCmdTestCase struct {
	name         string
	content      string
	filePath     string
	extraArgs    []string
	expectError  bool
	expectOutput string
}

// runSubCmdTests exercises a validate subcommand against a slice of test
// cases. It creates a temporary workflow file for cases that supply content,
// or uses the provided filePath directly.
func runSubCmdTests(t *testing.T, newCmd func() *cobra.Command, tests []subCmdTestCase) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filePath := test.filePath
			if filePath == "" {
				filePath = tempValidateFile(t, test.content)
			}

			var out bytes.Buffer

			cmd := newCmd()
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

// tempValidateFile writes content to a temporary workflow file and returns the
// path. The file and its directory are removed when the test finishes.
func tempValidateFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "validate_test")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(tmpDir)) })

	path := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestNewValidateCmd(t *testing.T) {
	tests := []subCmdTestCase{
		{
			name:         "both validations pass",
			content:      validWorkflowYAML,
			expectOutput: "is valid",
		},
		{
			name:         "JSON output includes schema and runtime sections",
			content:      validWorkflowYAML,
			extraArgs:    []string{"--output-json"},
			expectOutput: `"schema"`,
		},
		{
			name:        "non-existent file",
			filePath:    "/nonexistent/path/workflow.yaml",
			expectError: true,
		},
		{
			name:        "invalid YAML",
			content:     "invalid content: [",
			expectError: true,
		},
		{
			name:        "schema passes but runtime fails - unsupported DSL version",
			content:     workflowUnsupportedDSL,
			expectError: true,
		},
		{
			name:        "both fail - missing required name field",
			content:     workflowMissingName,
			expectError: true,
		},
		{
			name:        "schema fails, runtime passes - unsupported schedule field",
			content:     workflowScheduleAfterRejected,
			expectError: true,
		},
	}

	runSubCmdTests(t, newValidateCmd, tests)
}

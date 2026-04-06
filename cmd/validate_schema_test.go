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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

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

func TestNewValidateSchemaCmd(t *testing.T) {
	tests := []subCmdTestCase{
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
			name:        "invalid workflow - unsupported schedule field",
			content:     workflowScheduleAfterRejected,
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

	runSubCmdTests(t, newValidateSchemaCmd, tests)
}

type multiSchemaCase struct {
	name         string
	contents     []string // one entry per file
	expectError  bool
	expectOutput string
}

func runMultiSchemaTest(t *testing.T, tc multiSchemaCase) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "validate_schema_multi")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(tmpDir)) })

	args := make([]string, len(tc.contents))
	for i, content := range tc.contents {
		args[i] = writeTempFile(t, tmpDir, fmt.Sprintf("f%d.yaml", i), content)
	}

	var out bytes.Buffer
	cmd := newValidateSchemaCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(&out)
	cmd.SetArgs(args)

	err = cmd.Execute()
	if tc.expectError {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}

	if tc.expectOutput != "" {
		assert.Contains(t, out.String(), tc.expectOutput)
	}
}

func TestNewValidateSchemaCmdMultiFile(t *testing.T) {
	cases := []multiSchemaCase{
		{
			name:         "all files valid",
			contents:     []string{validWorkflowYAML, validWorkflowYAML},
			expectOutput: "All 2 file(s) passed schema validation.",
		},
		{
			name:         "one file fails, one passes",
			contents:     []string{validWorkflowYAML, workflowSchemaInvalid},
			expectError:  true,
			expectOutput: "1 of 2 file(s) failed schema validation.",
		},
		{
			name:         "all files fail",
			contents:     []string{workflowSchemaInvalid, workflowSchemaInvalid},
			expectError:  true,
			expectOutput: "2 of 2 file(s) failed schema validation.",
		},
		{
			// Both file names must appear in output, confirming no early exit.
			name:         "validation does not stop on first failure",
			contents:     []string{workflowSchemaInvalid, validWorkflowYAML},
			expectError:  true,
			expectOutput: "f0.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runMultiSchemaTest(t, tc)
		})
	}
}

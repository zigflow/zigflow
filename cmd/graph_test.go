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

func TestNewGraphCmd(t *testing.T) {
	tests := []struct {
		Name           string
		Content        string
		FilePath       string
		ExtraArgs      []string
		ExpectError    bool
		OutputContains []string
	}{
		{
			Name:           "valid workflow produces mermaid output",
			Content:        validWorkflowYAML,
			OutputContains: []string{"flowchart TD", "step"},
		},
		{
			Name:           "explicit --output mermaid flag",
			Content:        validWorkflowYAML,
			ExtraArgs:      []string{"--output", "mermaid"},
			OutputContains: []string{"flowchart TD"},
		},
		{
			Name:           "explicit -o shorthand flag",
			Content:        validWorkflowYAML,
			ExtraArgs:      []string{"-o", "mermaid"},
			OutputContains: []string{"flowchart TD"},
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
			Name:        "unsupported output format",
			Content:     validWorkflowYAML,
			ExtraArgs:   []string{"--output", "unknown"},
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			filePath := test.FilePath
			if filePath == "" {
				tmpDir, err := os.MkdirTemp("", "graph_test")
				require.NoError(t, err)
				defer func() {
					assert.NoError(t, os.RemoveAll(tmpDir))
				}()

				filePath = filepath.Join(tmpDir, "workflow.yaml")
				err = os.WriteFile(filePath, []byte(test.Content), 0o600)
				require.NoError(t, err)
			}

			// Capture stdout so we can assert on the generated output.
			r, w, err := os.Pipe()
			require.NoError(t, err)
			origStdout := os.Stdout
			os.Stdout = w

			cmd := newGraphCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(append([]string{filePath}, test.ExtraArgs...))
			execErr := cmd.Execute()

			assert.NoError(t, w.Close())
			os.Stdout = origStdout

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if test.ExpectError {
				assert.Error(t, execErr)
			} else {
				assert.NoError(t, execErr)
				for _, want := range test.OutputContains {
					assert.Contains(t, output, want)
				}
			}
		})
	}
}

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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const targetWithMarkers = `# Workflow

<!-- ZIGFLOW_GRAPH_START -->
<!-- ZIGFLOW_GRAPH_END -->

End of doc.
`

const targetWithEmbeddedPath = `# Workflow

<!-- ZIGFLOW_GRAPH_START ./workflow.yaml -->
<!-- ZIGFLOW_GRAPH_END -->

End of doc.
`

const targetWithCustomMarkers = `# Workflow

{{GRAPH_START}}
{{GRAPH_END}}

End of doc.
`

const targetWithoutMarkers = `# Workflow

No markers here.
`

func TestNewGraphInjectCmd(t *testing.T) {
	tests := []struct {
		Name                 string
		WorkflowYAML         string
		TargetContent        string
		ExtraArgs            []string
		UseNonExistentTarget bool // use /nonexistent/target.md instead of tmpDir/target.md
		ExpectError          bool
		OutputContains       []string
	}{
		{
			// Explicit workflow via --workflow flag.
			Name:          "explicit workflow injects mermaid graph",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithMarkers,
			ExtraArgs:     []string{"--workflow", "PLACEHOLDER_WORKFLOW"},
			OutputContains: []string{
				"<!-- ZIGFLOW_GRAPH_START -->",
				"<!-- ZIGFLOW_GRAPH_END -->",
				"```mermaid",
				"flowchart TD",
			},
		},
		{
			// Custom markers require --workflow so the start marker is known.
			Name:          "custom markers with explicit workflow",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithCustomMarkers,
			ExtraArgs: []string{
				"--workflow", "PLACEHOLDER_WORKFLOW",
				"--start-marker", "{{GRAPH_START}}",
				"--end-marker", "{{GRAPH_END}}",
			},
			OutputContains: []string{
				"flowchart TD",
				"{{GRAPH_START}}",
				"{{GRAPH_END}}",
			},
		},
		{
			Name:          "non-existent workflow file",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithMarkers,
			ExtraArgs:     []string{"--workflow", "/nonexistent/workflow.yaml"},
			ExpectError:   true,
		},
		{
			Name:                 "non-existent target file",
			WorkflowYAML:         validWorkflowYAML,
			ExtraArgs:            []string{"--workflow", "PLACEHOLDER_WORKFLOW"},
			UseNonExistentTarget: true,
			ExpectError:          true,
		},
		{
			// In explicit mode (--workflow set), missing markers is an error.
			Name:          "markers not found in target",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithoutMarkers,
			ExtraArgs:     []string{"--workflow", "PLACEHOLDER_WORKFLOW"},
			ExpectError:   true,
		},
		{
			Name:          "unsupported output format",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithMarkers,
			ExtraArgs:     []string{"--workflow", "PLACEHOLDER_WORKFLOW", "--output", "unknown"},
			ExpectError:   true,
		},
		{
			// Auto-detect: workflow path is embedded in the start marker.
			Name:          "auto-detect embedded workflow path",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithEmbeddedPath,
			OutputContains: []string{
				"<!-- ZIGFLOW_GRAPH_START ./workflow.yaml -->",
				"<!-- ZIGFLOW_GRAPH_END -->",
				"```mermaid",
				"flowchart TD",
			},
		},
		{
			// Auto-detect: file has no markers — skip silently, no error.
			Name:          "auto-detect skips file without markers",
			WorkflowYAML:  validWorkflowYAML,
			TargetContent: targetWithoutMarkers,
			OutputContains: []string{
				// Original content must be preserved.
				"No markers here.",
			},
		},
		{
			// Auto-detect: non-existent target file returns an error.
			Name:                 "auto-detect non-existent target file",
			WorkflowYAML:         validWorkflowYAML,
			UseNonExistentTarget: true,
			ExpectError:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "inject_test")
			require.NoError(t, err)
			defer func() { assert.NoError(t, os.RemoveAll(tmpDir)) }()

			workflowPath := filepath.Join(tmpDir, "workflow.yaml")
			if test.WorkflowYAML != "" {
				require.NoError(t, os.WriteFile(workflowPath, []byte(test.WorkflowYAML), 0o600))
			}

			targetPath := filepath.Join(tmpDir, "target.md")
			if test.UseNonExistentTarget {
				targetPath = "/nonexistent/target.md"
			} else if test.TargetContent != "" {
				require.NoError(t, os.WriteFile(targetPath, []byte(test.TargetContent), 0o600))
			}

			// Resolve PLACEHOLDER_WORKFLOW in extra args.
			resolvedArgs := make([]string, len(test.ExtraArgs))
			for i, a := range test.ExtraArgs {
				if a == "PLACEHOLDER_WORKFLOW" {
					resolvedArgs[i] = workflowPath
				} else {
					resolvedArgs[i] = a
				}
			}

			cmd := newGraphInjectCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(append([]string{targetPath}, resolvedArgs...))
			execErr := cmd.Execute()

			if test.ExpectError {
				assert.Error(t, execErr)
				return
			}

			require.NoError(t, execErr)

			// Read back the written file and assert on its content.
			written, err := os.ReadFile(targetPath)
			require.NoError(t, err)
			content := string(written)
			for _, want := range test.OutputContains {
				assert.Contains(t, content, want)
			}
		})
	}
}

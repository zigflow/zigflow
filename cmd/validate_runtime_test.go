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
	"testing"
)

func TestNewValidateRuntimeCmd(t *testing.T) {
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
			name:        "unsupported DSL version",
			content:     workflowUnsupportedDSL,
			expectError: true,
		},
		{
			name:        "missing required name field",
			content:     workflowMissingName,
			expectError: true,
		},
	}

	runSubCmdTests(t, newValidateRuntimeCmd, tests)
}

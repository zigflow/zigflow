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

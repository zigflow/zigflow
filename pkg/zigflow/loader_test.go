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

package zigflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

func TestLoadWorkflowFile(t *testing.T) {
	tests := []struct {
		Name        string
		Content     string
		Error       error
		ExpectError bool
	}{
		{
			Name: "Load valid workflow file",
			Content: `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
		{
			Name: "Invalid DSL version",
			Content: `document:
  dsl: 0.9.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
			Error:       zigflow.ErrUnsupportedDSL,
			ExpectError: true,
		},
		{
			Name:        "Invalid YAML",
			Content:     `invalid content: [`,
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "workflow_test")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, os.RemoveAll(tmpDir))
			}()

			filePath := filepath.Join(tmpDir, "zigflow.yaml")
			err = os.WriteFile(filePath, []byte(test.Content), 0o600)
			assert.NoError(t, err)

			workflow, err := zigflow.LoadFromFile(filePath)
			if test.ExpectError {
				assert.Error(t, err)
				assert.Nil(t, workflow)

				if test.Error != nil {
					assert.ErrorIs(t, err, test.Error)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workflow)
			}
		})
	}
}

// TestLoadWorkflowFile_FieldMapping verifies that document.workflowType and
// document.taskQueue are correctly mapped onto the SDK's Document.Name and
// Document.Namespace fields so that all downstream code continues to work.
func TestLoadWorkflowFile_FieldMapping(t *testing.T) {
	const content = `document:
  dsl: 1.0.0
  taskQueue: my-queue
  workflowType: my-workflow
  version: 1.0.0
do:
  - step:
      set:
        hello: world`

	tmpDir, err := os.MkdirTemp("", "workflow_test")
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	filePath := filepath.Join(tmpDir, "zigflow.yaml")
	err = os.WriteFile(filePath, []byte(content), 0o600)
	assert.NoError(t, err)

	workflow, err := zigflow.LoadFromFile(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "my-workflow", workflow.Document.Name, "workflowType must be mapped to Document.Name")
	assert.Equal(t, "my-queue", workflow.Document.Namespace, "taskQueue must be mapped to Document.Namespace")
}

// TestLoadFromFile_AcceptsLegacyFields verifies that the loader does not
// enforce schema rules. Legacy field names (document.name, document.namespace)
// are accepted without error. Schema enforcement is the responsibility of CLI
// callers via zigflow.ValidateFile.
func TestLoadFromFile_AcceptsLegacyFields(t *testing.T) {
	tests := []struct {
		Name    string
		Content string
	}{
		{
			Name: "legacy document.name loads without error",
			Content: `document:
  dsl: 1.0.0
  taskQueue: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
		{
			Name: "legacy document.namespace loads without error",
			Content: `document:
  dsl: 1.0.0
  namespace: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "workflow_test")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, os.RemoveAll(tmpDir))
			}()

			filePath := filepath.Join(tmpDir, "zigflow.yaml")
			err = os.WriteFile(filePath, []byte(test.Content), 0o600)
			assert.NoError(t, err)

			workflow, err := zigflow.LoadFromFile(filePath)
			assert.NoError(t, err, "loader must not enforce schema rules")
			assert.NotNil(t, workflow)
		})
	}
}

// TestValidateFile_RejectsLegacyFields verifies that ValidateFile rejects
// the legacy Serverless Workflow field names with ErrSchemaValidation.
func TestValidateFile_RejectsLegacyFields(t *testing.T) {
	tests := []struct {
		Name    string
		Content string
	}{
		{
			Name: "legacy document.name is rejected",
			Content: `document:
  dsl: 1.0.0
  taskQueue: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
		{
			Name: "legacy document.namespace is rejected",
			Content: `document:
  dsl: 1.0.0
  namespace: default
  workflowType: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
		{
			Name: "both legacy fields are rejected",
			Content: `document:
  dsl: 1.0.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "workflow_test")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, os.RemoveAll(tmpDir))
			}()

			filePath := filepath.Join(tmpDir, "zigflow.yaml")
			err = os.WriteFile(filePath, []byte(test.Content), 0o600)
			assert.NoError(t, err)

			err = zigflow.ValidateFile(filePath)
			assert.Error(t, err, "legacy fields must be rejected")
			assert.ErrorIs(t, err, zigflow.ErrSchemaValidation)
		})
	}
}

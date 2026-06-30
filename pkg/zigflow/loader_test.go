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

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
)

// writeWorkflow is a small helper that writes the given workflow YAML to a
// temp file and returns its path. The temp directory is cleaned up via
// t.TempDir, so callers do not need to manage cleanup explicitly.
func writeWorkflow(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "zigflow.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

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
// the legacy Open Workflow Specification field names with ErrSchemaValidation.
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

// TestLoadFromFile_VanillaWaitTaskTypeIsSDK verifies that a literal-numeric
// wait task is parsed by the SDK as its native *model.WaitTask. The Zigflow
// extension must not intercept vanilla waits, so the existing builder path
// stays unchanged.
func TestLoadFromFile_VanillaWaitTaskTypeIsSDK(t *testing.T) {
	const content = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: vanilla-wait
  version: 0.0.1
do:
  - pause:
      wait:
        seconds: 5`

	workflow, err := zigflow.LoadFromFile(writeWorkflow(t, content))
	require.NoError(t, err)
	require.NotNil(t, workflow.Do)
	tasks := *workflow.Do
	require.Len(t, tasks, 1)

	_, isSDKWait := tasks[0].Task.(*model.WaitTask)
	assert.True(t, isSDKWait, "vanilla wait must be parsed as the SDK's *model.WaitTask")
}

// TestLoadFromFile_WaitUntilTaskTypeIsZigflowExt verifies that a wait task
// using the absolute-time until form is renamed during normalisation and
// constructed by the SDK as a *models.WaitExtTask, ready for the dynamic
// builder.
func TestLoadFromFile_WaitUntilTaskTypeIsZigflowExt(t *testing.T) {
	const content = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: wait-until
  version: 0.0.1
do:
  - pause:
      wait:
        until: 2026-12-31T23:59:59Z`

	workflow, err := zigflow.LoadFromFile(writeWorkflow(t, content))
	require.NoError(t, err)
	require.NotNil(t, workflow.Do)
	tasks := *workflow.Do
	require.Len(t, tasks, 1)

	task, isExt := tasks[0].Task.(*models.WaitExtTask)
	require.True(t, isExt, "wait with until must be parsed as *models.WaitExtTask")
	require.NotNil(t, task.Wait)
	assert.Equal(t, "2026-12-31T23:59:59Z", task.Wait.Until)
}

// TestLoadFromFile_WaitExpressionDurationTypeIsZigflowExt verifies that a
// wait task with a runtime expression in a duration field is renamed and
// parsed as a *models.WaitExtTask.
func TestLoadFromFile_WaitExpressionDurationTypeIsZigflowExt(t *testing.T) {
	const content = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: wait-expression
  version: 0.0.1
do:
  - pause:
      wait:
        seconds: ${ $data.cooldownSeconds }`

	workflow, err := zigflow.LoadFromFile(writeWorkflow(t, content))
	require.NoError(t, err)
	require.NotNil(t, workflow.Do)
	tasks := *workflow.Do
	require.Len(t, tasks, 1)

	task, isExt := tasks[0].Task.(*models.WaitExtTask)
	require.True(t, isExt, "wait with expression duration must be parsed as *models.WaitExtTask")
	require.NotNil(t, task.Wait)
	assert.Equal(t, "${ $data.cooldownSeconds }", task.Wait.Seconds)
}

// TestLoadFromBytes_RunsTaskValidate confirms that LoadFromBytes
// exercises the per-task Validate() hook. A run script with `await:
// false` passes JSON schema and parses cleanly, but is rejected by
// RunTaskBuilder.Validate() (scripts must run with await).
//
// Every CLI/MCP validate entry point calls LoadFromBytes (directly or
// via LoadFromFile) after the schema check, so the hook reaches them
// all through this single integration point.
func TestLoadFromBytes_RunsTaskValidate(t *testing.T) {
	content := `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
do:
  - bad:
      run:
        await: false
        script:
          language: python
          code: "print(1)"`

	_, err := zigflow.LoadFromBytes([]byte(content))
	assert.Error(t, err, "await:false on run script must be rejected by Validate()")
	assert.Contains(t, err.Error(), "run scripts must be run with await")
}

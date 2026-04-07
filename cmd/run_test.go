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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/codec"
	"github.com/zigflow/zigflow/pkg/utils"
)

func TestPanicMessage(t *testing.T) {
	tests := []struct {
		Name     string
		Input    any
		Expected string
	}{
		{
			Name:     "error value",
			Input:    errors.New("something went wrong"),
			Expected: "something went wrong",
		},
		{
			Name:     "string value",
			Input:    "a plain string",
			Expected: "a plain string",
		},
		{
			Name:     "other value",
			Input:    42,
			Expected: fmt.Sprintf("%+v", 42),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, panicMessage(test.Input))
		})
	}
}

func TestBuildDataConverter(t *testing.T) {
	tests := []struct {
		Name         string
		ConvertData  string
		Endpoint     string
		KeyPath      string
		CodecHeaders map[string]string
		ExpectNil    bool
		ExpectError  bool
	}{
		{
			Name:        "disabled returns nil without reading key file",
			ConvertData: "",
			KeyPath:     "",
			ExpectNil:   true,
		},
		{
			Name:        "aes with missing key file returns error",
			ConvertData: "aes",
			KeyPath:     "/nonexistent/path/keys.yaml",
			ExpectNil:   true,
			ExpectError: true,
		},
		{
			Name:        "remote returns converter without error",
			ConvertData: "remote",
			Endpoint:    "http://localhost:8080",
			ExpectNil:   false,
		},
		{
			Name:         "remote with headers returns converter without error",
			ConvertData:  "remote",
			Endpoint:     "http://localhost:8080",
			CodecHeaders: map[string]string{"Authorization": "Bearer token"},
			ExpectNil:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			codecType, _ := codec.ParseCodecType(test.ConvertData)
			dc, err := codec.NewDataConverter(codecType, test.Endpoint, test.KeyPath, test.CodecHeaders)

			if test.ExpectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.ExpectNil {
				assert.Nil(t, dc)
			} else {
				assert.NotNil(t, dc)
			}
		})
	}
}

func TestNewRunCmd_Flags(t *testing.T) {
	cmd := newRunCmd()

	assert.NotNil(t, cmd.Flags().Lookup("file"))
	assert.NotNil(t, cmd.Flags().Lookup("validate"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-address"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-namespace"))
	assert.NotNil(t, cmd.Flags().Lookup("codec-endpoint"))
	assert.NotNil(t, cmd.Flags().Lookup("codec-headers"))
	assert.NotNil(t, cmd.Flags().Lookup("convert-data"))
	assert.NotNil(t, cmd.Flags().Lookup("converter-key-path"))
	assert.NotNil(t, cmd.Flags().Lookup("cloudevents-config"))
	assert.NotNil(t, cmd.Flags().Lookup("env-prefix"))
	assert.NotNil(t, cmd.Flags().Lookup("health-listen-address"))
	assert.NotNil(t, cmd.Flags().Lookup("metrics-listen-address"))
	assert.NotNil(t, cmd.Flags().Lookup("dir"))
	assert.NotNil(t, cmd.Flags().Lookup("glob"))
}

// minimalWorkflowYAML returns the smallest valid workflow YAML for the given
// taskQueue and workflowType. The document version and DSL are fixed to keep
// fixtures short.
func minimalWorkflowYAML(namespace, name string) string {
	return `document:
  dsl: 1.0.0
  taskQueue: ` + namespace + `
  workflowType: ` + name + `
  version: 0.0.1
do:
  - noop:
      set:
        set: {}
`
}

// writeTempWorkflow writes a minimal workflow YAML file into dir and returns
// the absolute path. It fails the test immediately on any write error.
func writeTempWorkflow(t *testing.T, dir, namespace, name string) string {
	t.Helper()
	p := filepath.Join(dir, namespace+"."+name+".yaml")
	require.NoError(t, os.WriteFile(p, []byte(minimalWorkflowYAML(namespace, name)), 0o600))
	return p
}

// ---- discoverWorkflowFiles ----

func TestDiscoverWorkflowFiles_NoFilesError(t *testing.T) {
	_, err := discoverWorkflowFiles(&runOptions{
		DirectoryGlob: "*.yaml",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No workflow files found")
}

func TestDiscoverWorkflowFiles_ExplicitFiles(t *testing.T) {
	dir := t.TempDir()
	p := writeTempWorkflow(t, dir, "ns", "wf")

	files, err := discoverWorkflowFiles(&runOptions{
		Files:         []string{p},
		DirectoryGlob: "*.yaml",
	})
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, p, files[0])
}

func TestDiscoverWorkflowFiles_DirectoryGlob(t *testing.T) {
	dir := t.TempDir()
	writeTempWorkflow(t, dir, "ns", "wf1")
	writeTempWorkflow(t, dir, "ns", "wf2")

	files, err := discoverWorkflowFiles(&runOptions{
		DirectoryPath: dir,
		DirectoryGlob: "*.yaml",
	})
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestDiscoverWorkflowFiles_MergesFilesAndDirectory(t *testing.T) {
	dir := t.TempDir()
	p1 := writeTempWorkflow(t, dir, "ns", "wf1")
	p2 := writeTempWorkflow(t, dir, "ns", "wf2")

	files, err := discoverWorkflowFiles(&runOptions{
		Files:         []string{p1},
		DirectoryPath: dir,
		DirectoryGlob: "*.yaml",
	})
	require.NoError(t, err)
	// p1 discovered by both sources; must appear exactly once.
	assert.Len(t, files, 2)
	assert.Contains(t, files, p1)
	assert.Contains(t, files, p2)
}

func TestDiscoverWorkflowFiles_DeduplicatesRelativeAndAbsolute(t *testing.T) {
	dir := t.TempDir()
	p := writeTempWorkflow(t, dir, "ns", "wf")

	// Pass both the absolute path and a path that resolves to the same file.
	files, err := discoverWorkflowFiles(&runOptions{
		Files:         []string{p, p},
		DirectoryGlob: "*.yaml",
	})
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestDiscoverWorkflowFiles_InvalidGlobError(t *testing.T) {
	// An invalid directory causes the glob to fail.
	_, err := discoverWorkflowFiles(&runOptions{
		DirectoryPath: string([]byte{0}), // NUL in path is rejected by the OS
		DirectoryGlob: "*.yaml",
	})
	assert.Error(t, err)
}

// ---- loadWorkflows ----

func newTestValidator(t *testing.T) *utils.Validator {
	t.Helper()
	v, err := utils.NewValidator()
	require.NoError(t, err)
	return v
}

func TestLoadWorkflows_SingleValidFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTempWorkflow(t, dir, "myns", "mywf")

	validator := newTestValidator(t)
	regs, err := loadWorkflows([]string{p}, "", validator, false)
	require.NoError(t, err)
	require.Len(t, regs, 1)
	assert.Equal(t, "myns", regs[0].TaskQueue)
	assert.Equal(t, "mywf", regs[0].WorkflowType)
	assert.Equal(t, p, regs[0].SourceFile)
}

func TestLoadWorkflows_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	p1 := writeTempWorkflow(t, dir, "ns", "wf1")
	p2 := writeTempWorkflow(t, dir, "ns", "wf2")

	validator := newTestValidator(t)
	regs, err := loadWorkflows([]string{p1, p2}, "", validator, false)
	require.NoError(t, err)
	assert.Len(t, regs, 2)
}

func TestLoadWorkflows_RejectsEmptyName(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty-name.yaml")
	require.NoError(t, os.WriteFile(p, []byte(`document:
  dsl: 1.0.0
  taskQueue: ns
  workflowType: ""
  version: 0.0.1
do:
  - noop:
      set:
        set: {}
`), 0o600))

	validator := newTestValidator(t)
	_, err := loadWorkflows([]string{p}, "", validator, false)
	assert.Error(t, err, "empty workflowType must be rejected")
}

func TestLoadWorkflows_RejectsEmptyNamespace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty-ns.yaml")
	require.NoError(t, os.WriteFile(p, []byte(`document:
  dsl: 1.0.0
  taskQueue: ""
  workflowType: wf
  version: 0.0.1
do:
  - noop:
      set:
        set: {}
`), 0o600))

	validator := newTestValidator(t)
	_, err := loadWorkflows([]string{p}, "", validator, false)
	assert.Error(t, err, "empty taskQueue must be rejected")
}

// ---- validateWorkflowConflicts ----

func TestValidateWorkflowConflicts_DuplicateNameSameQueue(t *testing.T) {
	regs := []*workflowRegistration{
		{SourceFile: "a.yaml", TaskQueue: "q", WorkflowType: "wf"},
		{SourceFile: "b.yaml", TaskQueue: "q", WorkflowType: "wf"},
	}
	err := validateWorkflowConflicts(regs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Duplicate workflow name on the same task queue")
}

func TestValidateWorkflowConflicts_SameNameDifferentQueues(t *testing.T) {
	regs := []*workflowRegistration{
		{SourceFile: "a.yaml", TaskQueue: "q1", WorkflowType: "wf"},
		{SourceFile: "b.yaml", TaskQueue: "q2", WorkflowType: "wf"},
	}
	assert.NoError(t, validateWorkflowConflicts(regs))
}

func TestValidateWorkflowConflicts_DifferentNamesSameQueue(t *testing.T) {
	regs := []*workflowRegistration{
		{SourceFile: "a.yaml", TaskQueue: "q", WorkflowType: "wf1"},
		{SourceFile: "b.yaml", TaskQueue: "q", WorkflowType: "wf2"},
	}
	assert.NoError(t, validateWorkflowConflicts(regs))
}

// ---- prepareRegistrations ----

func TestPrepareRegistrations_HappyPath(t *testing.T) {
	dir := t.TempDir()
	writeTempWorkflow(t, dir, "ns1", "wf1")
	writeTempWorkflow(t, dir, "ns2", "wf2")

	opts := &runOptions{
		DirectoryPath: dir,
		DirectoryGlob: "*.yaml",
		Validate:      false,
	}

	regs, err := prepareRegistrations(opts)
	require.NoError(t, err)
	assert.Len(t, regs, 2)
}

// TestLoadWorkflows_ValidateFlagControlsSchemaValidation verifies that the
// validate flag is threaded through to schema validation in the loader.
func TestLoadWorkflows_ValidateFlagControlsSchemaValidation(t *testing.T) {
	// A workflow using the legacy document.name field, which the Zigflow schema
	// rejects but the raw SDK unmarshal accepts.
	const legacyWorkflow = `document:
  dsl: 1.0.0
  taskQueue: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world
`

	dir := t.TempDir()
	p := filepath.Join(dir, "legacy.yaml")
	require.NoError(t, os.WriteFile(p, []byte(legacyWorkflow), 0o600))

	validator := newTestValidator(t)

	t.Run("validate=true rejects legacy fields", func(t *testing.T) {
		_, err := loadWorkflows([]string{p}, "", validator, true)
		assert.Error(t, err, "schema validation must reject legacy fields when validate=true")
	})

	t.Run("validate=false allows legacy fields", func(t *testing.T) {
		regs, err := loadWorkflows([]string{p}, "", validator, false)
		assert.NoError(t, err, "legacy fields must be accepted when validate=false")
		assert.Len(t, regs, 1)
	})
}

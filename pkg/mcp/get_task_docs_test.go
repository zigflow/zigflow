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

package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/schema"
	zigflowdocs "github.com/zigflow/zigflow/docs"
	zigflowexamples "github.com/zigflow/zigflow/examples"
)

func taskDocs(t *testing.T, taskType string) GetTaskDocsOutput {
	t.Helper()

	out, err := getTaskDocs(testVersion, zigflowdocs.TaskDocsFS, zigflowexamples.EmbeddedFS, taskType)
	require.NoError(t, err)

	return out
}

// schemaTaskTypes returns the authoritative list of task keys, derived from the
// schema's $defs/task OneOf, mirroring TestLLMSTxtDocumentsEveryTaskType.
func schemaTaskTypes(t *testing.T) []string {
	t.Helper()

	s, err := schema.BuildSchema(testVersion, outputFormatJSON)
	require.NoError(t, err)

	taskDef, ok := s.Defs["task"]
	require.True(t, ok, "schema is missing the $defs/task definition")
	require.NotEmpty(t, taskDef.OneOf, "$defs/task has no OneOf task references")

	var keys []string
	for _, ref := range taskDef.OneOf {
		name := strings.TrimPrefix(ref.Ref, "#/$defs/")
		key := strings.TrimSuffix(name, "Task")
		require.NotEmpty(t, key)
		require.NotEqual(t, key, name, "unexpected task ref %q", ref.Ref)
		keys = append(keys, key)
	}

	return keys
}

func TestGetTaskDocs_Call(t *testing.T) {
	out := taskDocs(t, "call")

	assert.Empty(t, out.Errors)
	assert.Equal(t, "call", out.TaskType)
	assert.NotEmpty(t, out.Description, "description sourced from schema def")
	assert.Contains(t, out.Schema, "properties", "schema definition returned")
	assert.Contains(t, out.Documentation, "# Call", "full reference page returned")
	assert.Equal(t, []string{"https://zigflow.dev/docs/dsl/tasks/call"}, out.RelatedLinks)
}

func TestGetTaskDocs_CallSubTypesFromSchema(t *testing.T) {
	out := taskDocs(t, "call")
	assert.Equal(t, []string{"activity", "grpc", "http"}, out.SubTypes)
}

func TestGetTaskDocs_CallExamplesFromCatalog(t *testing.T) {
	out := taskDocs(t, "call")
	require.NotEmpty(t, out.Examples, "call should reference example workflows")

	names := make([]string, len(out.Examples))
	for i, ex := range out.Examples {
		names[i] = ex.Name
		assert.NotEmpty(t, ex.Title, "example refs carry a title")
	}
	assert.Contains(t, names, "activity-call")
}

func TestGetTaskDocs_SetHasNoSubTypesOrExamples(t *testing.T) {
	out := taskDocs(t, "set")

	assert.Empty(t, out.Errors)
	assert.Equal(t, "set", out.TaskType)
	assert.Empty(t, out.SubTypes, "set has no discriminated variants")
	assert.Empty(t, out.Examples, "set is ubiquitous and intentionally unmapped")
	assert.Contains(t, out.Documentation, "# Set")
}

func TestGetTaskDocs_CaseInsensitive(t *testing.T) {
	out := taskDocs(t, "  CALL  ")
	assert.Empty(t, out.Errors)
	assert.Equal(t, "call", out.TaskType)
}

func TestGetTaskDocs_EmptyInput(t *testing.T) {
	out, err := getTaskDocs(testVersion, zigflowdocs.TaskDocsFS, zigflowexamples.EmbeddedFS, "")
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, stageInput, out.Errors[0].Stage)
	assert.Contains(t, out.Errors[0].Message, "required")
	// The supported list is surfaced to guide the caller.
	assert.Contains(t, out.Errors[0].Message, "call")
}

func TestGetTaskDocs_UnknownTask(t *testing.T) {
	out, err := getTaskDocs(testVersion, zigflowdocs.TaskDocsFS, zigflowexamples.EmbeddedFS, "emit")
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, stageInput, out.Errors[0].Stage)
	assert.Contains(t, out.Errors[0].Message, "emit")
	assert.Contains(t, out.Errors[0].Message, "wait", "supported list is included")
}

// TestGetTaskDocs_EverySchemaTaskTypeIsServable is the drift guard: every task
// type the schema recognises must resolve to a complete response (schema def +
// embedded reference page). A task added to or renamed in the schema fails this
// test until its documentation is added.
func TestGetTaskDocs_EverySchemaTaskTypeIsServable(t *testing.T) {
	for _, taskType := range schemaTaskTypes(t) {
		t.Run(taskType, func(t *testing.T) {
			out := taskDocs(t, taskType)
			assert.Empty(t, out.Errors, "task %q must be servable", taskType)
			assert.Equal(t, taskType, out.TaskType)
			assert.NotEmpty(t, out.Description, "task %q missing schema description", taskType)
			assert.NotEmpty(t, out.Schema, "task %q missing schema", taskType)
			assert.NotEmpty(t, out.Documentation, "task %q missing reference page", taskType)
		})
	}
}

// TestGetTaskDocs_NoOrphanDocPages guards the reverse direction: every embedded
// task reference page must correspond to a real schema task type, so a stray or
// misnamed page is caught.
func TestGetTaskDocs_NoOrphanDocPages(t *testing.T) {
	schemaTypes := make(map[string]struct{})
	for _, k := range schemaTaskTypes(t) {
		schemaTypes[k] = struct{}{}
	}

	supported, err := supportedTaskTypes(zigflowdocs.TaskDocsFS)
	require.NoError(t, err)
	require.NotEmpty(t, supported)

	for _, taskType := range supported {
		_, ok := schemaTypes[taskType]
		assert.Truef(t, ok, "doc page %q.md has no matching schema task type", taskType)
	}
}

// TestGetTaskDocs_ExampleTagsResolve guards the curated task->tag map: every tag
// it references must match at least one bundled example, so a renamed or removed
// tag is caught rather than silently returning no examples.
func TestGetTaskDocs_ExampleTagsResolve(t *testing.T) {
	catalog, err := zigflowexamples.LoadCatalog(zigflowexamples.EmbeddedFS, ".")
	require.NoError(t, err)

	tagged := make(map[string]struct{})
	for _, ex := range catalog {
		for _, tag := range ex.Tags {
			tagged[tag] = struct{}{}
		}
	}

	for taskType, tags := range taskExampleTags {
		for _, tag := range tags {
			_, ok := tagged[tag]
			assert.Truef(t, ok, "task %q maps to tag %q which no example uses", taskType, tag)
		}
	}
}

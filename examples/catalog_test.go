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

package examples_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/examples"
)

func workflowYAML(title, summary string) []byte {
	return []byte("document:\n  title: " + title + "\n  summary: " + summary + "\n")
}

func TestLoadCatalog_ReturnsSortedExamples(t *testing.T) {
	fsys := fstest.MapFS{
		"zebra-workflow/workflow.yaml":  {Data: workflowYAML("Zebra Workflow", "Last alphabetically")},
		"alpha-workflow/workflow.yaml":  {Data: workflowYAML("Alpha Workflow", "First alphabetically")},
		"middle-workflow/workflow.yaml": {Data: workflowYAML("Middle Workflow", "Somewhere in the middle")},
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	require.NoError(t, err)
	require.Len(t, catalog, 3)

	assert.Equal(t, "alpha-workflow", catalog[0].Name)
	assert.Equal(t, "middle-workflow", catalog[1].Name)
	assert.Equal(t, "zebra-workflow", catalog[2].Name)
}

func TestLoadCatalog_ExtractsTitleAndDescription(t *testing.T) {
	fsys := fstest.MapFS{
		"signal/workflow.yaml": {Data: []byte(`document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: signal
  version: 0.0.1
  title: Signal Listeners
  summary: Listen for Temporal signal events
`)},
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	require.NoError(t, err)
	require.Len(t, catalog, 1)

	ex := catalog[0]
	assert.Equal(t, "signal", ex.Name)
	assert.Equal(t, "Signal Listeners", ex.Title)
	assert.Equal(t, "Listen for Temporal signal events", ex.Description)
}

func TestLoadCatalog_InfoYAMLFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"multi-file/info.yaml": {Data: workflowYAML("Multiple Workflow Files", "Run multiple workflow definitions from separate YAML files")},
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	require.NoError(t, err)
	require.Len(t, catalog, 1)

	ex := catalog[0]
	assert.Equal(t, "multi-file", ex.Name)
	assert.Equal(t, "Multiple Workflow Files", ex.Title)
	assert.Equal(t, "Run multiple workflow definitions from separate YAML files", ex.Description)
}

func TestLoadCatalog_SkipsNonDirectoryEntries(t *testing.T) {
	fsys := fstest.MapFS{
		"k8s-values.yaml":           {Data: []byte("somekey: somevalue\n")},
		"hello-world/workflow.yaml": {Data: workflowYAML("Hello World", "Hello world with Zigflow")},
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	require.NoError(t, err)
	require.Len(t, catalog, 1)
	assert.Equal(t, "hello-world", catalog[0].Name)
}

func TestLoadCatalog_ErrorOnMissingMetadataFile(t *testing.T) {
	fsys := fstest.MapFS{
		"broken-example/other.yaml": {Data: []byte("somekey: somevalue\n")},
	}

	_, err := examples.LoadCatalog(fsys, ".")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "broken-example")
}

func TestLoadCatalog_ErrorOnUnreadableDirectory(t *testing.T) {
	_, err := examples.LoadCatalog(fstest.MapFS{}, "nonexistent")
	assert.Error(t, err)
}

func TestLoadCatalog_KnownTagsAttached(t *testing.T) {
	fsys := fstest.MapFS{
		"signal/workflow.yaml": {Data: workflowYAML("Signal Listeners", "Listen for Temporal signal events")},
		"query/workflow.yaml":  {Data: workflowYAML("Query Listeners", "Listen for Temporal query events")},
		"basic/workflow.yaml":  {Data: workflowYAML("Basic", "A basic example")},
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	require.NoError(t, err)
	require.Len(t, catalog, 3)

	byName := make(map[string]examples.Example)
	for _, ex := range catalog {
		byName[ex.Name] = ex
	}

	assert.Equal(t, []string{"signal"}, byName["signal"].Tags)
	assert.Equal(t, []string{"query"}, byName["query"].Tags)
	assert.Empty(t, byName["basic"].Tags)
}

func TestLoadCatalog_DirIsRelativeToFS(t *testing.T) {
	fsys := fstest.MapFS{
		"hello-world/workflow.yaml": {Data: workflowYAML("Hello World", "Hello world with Zigflow")},
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	require.NoError(t, err)
	require.Len(t, catalog, 1)

	assert.Equal(t, "hello-world", catalog[0].Dir)
}

func TestLoadCatalog_EmbeddedFS(t *testing.T) {
	catalog, err := examples.LoadCatalog(examples.EmbeddedFS, ".")
	require.NoError(t, err)
	assert.NotEmpty(t, catalog, "embedded FS must contain at least one example")

	for _, ex := range catalog {
		assert.NotEmpty(t, ex.Name)
		assert.NotEmpty(t, ex.Title)
	}
}

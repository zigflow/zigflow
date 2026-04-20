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
	"context"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exampleFS(entries map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for path, content := range entries {
		fsys[path] = &fstest.MapFile{Data: []byte(content)}
	}

	return fsys
}

func workflowFile(title, summary string) string {
	return "document:\n  title: " + title + "\n  summary: " + summary + "\n"
}

func TestListExamples_ReturnsExamples(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"signal/workflow.yaml": workflowFile("Signal Listeners", "Listen for Temporal signal events"),
		"query/workflow.yaml":  workflowFile("Query Listeners", "Listen for Temporal query events"),
	})

	m := &MCP{examplesFS: fsys}
	_, out, err := m.ListExamples(context.Background(), nil, ListExamplesInput{})
	require.NoError(t, err)
	assert.Len(t, out.Examples, 2)
}

func TestListExamples_StableOrder(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"zebra/workflow.yaml":  workflowFile("Zebra", "Last"),
		"alpha/workflow.yaml":  workflowFile("Alpha", "First"),
		"middle/workflow.yaml": workflowFile("Middle", "Middle"),
	})

	m := &MCP{examplesFS: fsys}
	_, out, err := m.ListExamples(context.Background(), nil, ListExamplesInput{})
	require.NoError(t, err)
	require.Len(t, out.Examples, 3)

	names := []string{out.Examples[0].Name, out.Examples[1].Name, out.Examples[2].Name}
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)
	assert.Equal(t, sorted, names, "examples must be in alphabetical order")
}

func TestListExamples_NamesNonEmpty(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"hello-world/workflow.yaml": workflowFile("Hello World", "Hello world with Zigflow"),
		"signal/workflow.yaml":      workflowFile("Signal Listeners", "Listen for Temporal signal events"),
	})

	m := &MCP{examplesFS: fsys}
	_, out, err := m.ListExamples(context.Background(), nil, ListExamplesInput{})
	require.NoError(t, err)

	for _, ex := range out.Examples {
		assert.NotEmpty(t, ex.Name, "example Name must be non-empty for use with get_example")
	}
}

func TestListExamples_NoYAMLContents(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"hello-world/workflow.yaml": workflowFile("Hello World", "Hello world with Zigflow"),
	})

	m := &MCP{examplesFS: fsys}
	_, out, err := m.ListExamples(context.Background(), nil, ListExamplesInput{})
	require.NoError(t, err)
	require.Len(t, out.Examples, 1)

	ex := out.Examples[0]
	assert.Equal(t, "hello-world", ex.Name)
	assert.Equal(t, "Hello World", ex.Title)
	assert.Equal(t, "Hello world with Zigflow", ex.Description)
	assert.IsType(t, ExampleSummary{}, ex)
}

func TestListExamples_LoaderFailureSurfaced(t *testing.T) {
	m := &MCP{examplesFS: exampleFS(map[string]string{
		"broken/other.yaml": "somekey: somevalue\n",
	})}
	_, _, err := m.ListExamples(context.Background(), nil, ListExamplesInput{})
	assert.Error(t, err)
}

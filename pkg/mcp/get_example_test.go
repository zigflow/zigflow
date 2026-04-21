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
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zigflowexamples "github.com/zigflow/zigflow/examples"
)

// signalFS builds a minimal embedded FS containing the "signal" example.
func signalFS() fstest.MapFS {
	return exampleFS(map[string]string{
		"signal/workflow.yaml": "document:\n  title: Signal\n  summary: Signal example\ndo: []\n",
	})
}

func TestGetExample_ValidName(t *testing.T) {
	out, err := getExampleFromFS(signalFS(), "signal")
	require.NoError(t, err)
	assert.Empty(t, out.Errors)
	assert.Equal(t, "signal", out.Name)
}

func TestGetExample_ContentNonEmptyAndIsYAML(t *testing.T) {
	out, err := getExampleFromFS(signalFS(), "signal")
	require.NoError(t, err)
	assert.NotEmpty(t, out.Content)
	assert.Contains(t, out.Content, "document:")
}

func TestGetExample_MetadataMatchesCatalog(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"query/workflow.yaml": "document:\n  title: Query Listeners\n  metadata:\n    tags: [query]\n  " +
			"summary: Listen for Temporal query events\ndo: []\n",
	})

	out, err := getExampleFromFS(fsys, "query")
	require.NoError(t, err)
	assert.Equal(t, "query", out.Name)
	assert.Equal(t, "Query Listeners", out.Title)
	assert.Equal(t, "Listen for Temporal query events", out.Description)
	assert.Equal(t, []string{"query"}, out.Tags)
}

func TestGetExample_EmptyName(t *testing.T) {
	out, err := getExampleFromFS(signalFS(), "")
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "input", out.Errors[0].Stage)
	assert.NotEmpty(t, out.Errors[0].Message)
}

func TestGetExample_WhitespaceName(t *testing.T) {
	out, err := getExampleFromFS(signalFS(), "   ")
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "input", out.Errors[0].Stage)
}

func TestGetExample_UnknownName(t *testing.T) {
	out, err := getExampleFromFS(signalFS(), "does-not-exist")
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)
	assert.Equal(t, "input", out.Errors[0].Stage)
	assert.Contains(t, out.Errors[0].Message, "does-not-exist")
	assert.Contains(t, out.Errors[0].Message, "signal")
}

func TestGetExample_UnknownNameListsAvailable(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"alpha/workflow.yaml": "document:\n  title: Alpha\n  summary: First\n",
		"beta/workflow.yaml":  "document:\n  title: Beta\n  summary: Second\n",
	})

	out, err := getExampleFromFS(fsys, "gamma")
	require.NoError(t, err)
	require.Len(t, out.Errors, 1)
	assert.Contains(t, out.Errors[0].Message, "alpha")
	assert.Contains(t, out.Errors[0].Message, "beta")
}

func TestGetExample_ReadFailureSurfacedAsGoError(t *testing.T) {
	_, err := getExampleFromFS(&alwaysErrFS{}, "anything")
	assert.Error(t, err)
}

func TestGetExample_InfoYAMLFallback(t *testing.T) {
	fsys := exampleFS(map[string]string{
		"multi/info.yaml": "document:\n  title: Multi\n  summary: Uses info.yaml\n",
	})

	out, err := getExampleFromFS(fsys, "multi")
	require.NoError(t, err)
	assert.Empty(t, out.Errors)
	assert.Contains(t, out.Content, "document:")
}

func TestGetExample_EmbeddedFS(t *testing.T) {
	out, err := getExampleFromFS(zigflowexamples.EmbeddedFS, "signal")
	require.NoError(t, err)
	assert.Empty(t, out.Errors)
	assert.Equal(t, "signal", out.Name)
	assert.NotEmpty(t, out.Content)
}

// alwaysErrFS is an fs.FS whose Open always returns an error.
type alwaysErrFS struct{}

func (a *alwaysErrFS) Open(_ string) (fs.File, error) {
	return nil, fs.ErrInvalid
}

func TestReadExampleContent_NoYAMLFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"mydir/README.md": &fstest.MapFile{Data: []byte("# Example")},
	}
	_, err := readExampleContent(fsys, "mydir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow.yaml or info.yaml found")
}

func TestReadExampleContent_ReadError(t *testing.T) {
	_, err := readExampleContent(&alwaysErrFS{}, "mydir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow.yaml")
}

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

package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/graph"
)

const (
	defaultStart = "<!-- ZIGFLOW_GRAPH_START -->"
	defaultEnd   = "<!-- ZIGFLOW_GRAPH_END -->"
)

func TestInjectGraph_Basic(t *testing.T) {
	src := "before\n" + defaultStart + defaultEnd + "\nafter"
	got, err := graph.InjectGraph(src, defaultStart, defaultEnd, "content")
	require.NoError(t, err)
	assert.Equal(t, "before\n"+defaultStart+"\ncontent\n"+defaultEnd+"\nafter", got)
}

func TestInjectGraph_Idempotent(t *testing.T) {
	src := "# Doc\n" + defaultStart + "\nold content\n" + defaultEnd + "\nmore text"
	first, err := graph.InjectGraph(src, defaultStart, defaultEnd, "new content")
	require.NoError(t, err)
	second, err := graph.InjectGraph(first, defaultStart, defaultEnd, "new content")
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestInjectGraph_ReplacesExistingContent(t *testing.T) {
	src := defaultStart + "\nold graph\n" + defaultEnd
	got, err := graph.InjectGraph(src, defaultStart, defaultEnd, "new graph")
	require.NoError(t, err)
	assert.Contains(t, got, "new graph")
	assert.NotContains(t, got, "old graph")
}

func TestInjectGraph_PreservesTextOutsideMarkers(t *testing.T) {
	src := "header\n" + defaultStart + "\nstale\n" + defaultEnd + "\nfooter"
	got, err := graph.InjectGraph(src, defaultStart, defaultEnd, "fresh")
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "header\n")
	assert.Contains(t, got, "\nfooter")
}

func TestInjectGraph_MissingStartMarker(t *testing.T) {
	src := "no markers here\n" + defaultEnd
	_, err := graph.InjectGraph(src, defaultStart, defaultEnd, "content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start marker")
}

func TestInjectGraph_MissingEndMarker(t *testing.T) {
	src := defaultStart + "\nsome content"
	_, err := graph.InjectGraph(src, defaultStart, defaultEnd, "content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end marker")
}

func TestInjectGraph_EndBeforeStart(t *testing.T) {
	// End marker appears before start marker — treated as end not found.
	src := defaultEnd + "\n" + defaultStart
	_, err := graph.InjectGraph(src, defaultStart, defaultEnd, "content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end marker")
}

func TestInjectGraph_CustomMarkers(t *testing.T) {
	start := "{{GRAPH_START}}"
	end := "{{GRAPH_END}}"
	src := "intro\n" + start + end + "\noutro"
	got, err := graph.InjectGraph(src, start, end, "diagram")
	require.NoError(t, err)
	assert.Contains(t, got, "diagram")
	assert.Contains(t, got, "intro")
	assert.Contains(t, got, "outro")
}

func TestInjectGraph_MultipleMarkerOccurrences(t *testing.T) {
	// Only the first start + first end-after-start are used.
	src := defaultStart + "\nA\n" + defaultEnd + "\n" + defaultStart + "\nB\n" + defaultEnd
	got, err := graph.InjectGraph(src, defaultStart, defaultEnd, "X")
	require.NoError(t, err)
	// First block updated; second block untouched.
	assert.Contains(t, got, "\nX\n")
	assert.Contains(t, got, "\nB\n")
}

func TestParseEmbeddedPath_WithPath(t *testing.T) {
	src := "# Doc\n<!-- ZIGFLOW_GRAPH_START ./my-workflow.yaml -->\n<!-- ZIGFLOW_GRAPH_END -->\n"
	wfPath, fullMarker, found := graph.ParseEmbeddedPath(src, graph.DefaultWorkflowFile)
	require.True(t, found)
	assert.Equal(t, "./my-workflow.yaml", wfPath)
	assert.Equal(t, "<!-- ZIGFLOW_GRAPH_START ./my-workflow.yaml -->", fullMarker)
}

func TestParseEmbeddedPath_WithoutPath(t *testing.T) {
	src := "# Doc\n" + defaultStart + "\n" + defaultEnd + "\n"
	wfPath, fullMarker, found := graph.ParseEmbeddedPath(src, graph.DefaultWorkflowFile)
	require.True(t, found)
	assert.Equal(t, graph.DefaultWorkflowFile, wfPath)
	assert.Equal(t, defaultStart, fullMarker)
}

func TestParseEmbeddedPath_NotFound(t *testing.T) {
	src := "# Doc\nNo markers here.\n"
	wfPath, fullMarker, found := graph.ParseEmbeddedPath(src, graph.DefaultWorkflowFile)
	assert.False(t, found)
	assert.Empty(t, wfPath)
	assert.Empty(t, fullMarker)
}

func TestParseEmbeddedPath_ExtraWhitespace(t *testing.T) {
	src := "<!-- ZIGFLOW_GRAPH_START  ./spaced.yaml  -->"
	wfPath, _, found := graph.ParseEmbeddedPath(src, graph.DefaultWorkflowFile)
	require.True(t, found)
	assert.Equal(t, "./spaced.yaml", wfPath)
}

func TestParseEmbeddedPath_FullMarkerMatchesInjectGraph(t *testing.T) {
	// Verify that the fullStartMarker returned by ParseEmbeddedPath can be
	// passed directly to InjectGraph and correctly locates the block.
	marker := "<!-- ZIGFLOW_GRAPH_START ./wf.yaml -->"
	src := "before\n" + marker + "\nold\n" + defaultEnd + "\nafter"
	wfPath, fullMarker, found := graph.ParseEmbeddedPath(src, graph.DefaultWorkflowFile)
	require.True(t, found)
	assert.Equal(t, "./wf.yaml", wfPath)

	got, err := graph.InjectGraph(src, fullMarker, defaultEnd, "new")
	require.NoError(t, err)
	assert.Contains(t, got, "new")
	assert.NotContains(t, got, "old")
	assert.Contains(t, got, "before")
	assert.Contains(t, got, "after")
}

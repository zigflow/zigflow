/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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

package graph

import (
	"fmt"
	"strings"
)

const (
	// DefaultStartMarkerPrefix is the opening text of the ZIGFLOW_GRAPH_START
	// comment. The full marker may optionally embed a workflow file path between
	// this prefix and the closing "-->", for example:
	//
	//	<!-- ZIGFLOW_GRAPH_START ./workflow.yaml -->
	//
	// When no path is embedded the comment is simply:
	//
	//	<!-- ZIGFLOW_GRAPH_START -->
	DefaultStartMarkerPrefix = "<!-- ZIGFLOW_GRAPH_START"

	// DefaultEndMarker is the closing marker for the injection block.
	DefaultEndMarker = "<!-- ZIGFLOW_GRAPH_END -->"

	// DefaultWorkflowFile is used by ParseEmbeddedPath when the start marker
	// carries no embedded path.
	DefaultWorkflowFile = "workflow.yaml"
)

// ParseEmbeddedPath reads the optional workflow file path embedded in a
// ZIGFLOW_GRAPH_START comment in src.
//
// Parameters:
//   - src: full contents of the target document
//   - defaultPath: returned as workflowPath when no path is embedded
//
// Returns:
//   - workflowPath: the embedded path, or defaultPath when none is present
//   - fullStartMarker: the exact marker string as it appears in src (e.g.
//     "<!-- ZIGFLOW_GRAPH_START ./wf.yaml -->"), ready to pass to InjectGraph
//   - found: false when no DefaultStartMarkerPrefix exists in src, meaning the
//     caller should skip the file without error
func ParseEmbeddedPath(src, defaultPath string) (workflowPath, fullStartMarker string, found bool) {
	prefixIdx := strings.Index(src, DefaultStartMarkerPrefix)
	if prefixIdx < 0 {
		return "", "", false
	}
	afterPrefix := prefixIdx + len(DefaultStartMarkerPrefix)

	// Find the closing "-->" of this HTML comment.
	closeIdx := strings.Index(src[afterPrefix:], "-->")
	if closeIdx < 0 {
		return "", "", false
	}

	fullStartMarker = src[prefixIdx : afterPrefix+closeIdx+len("-->")]
	embedded := strings.TrimSpace(src[afterPrefix : afterPrefix+closeIdx])

	if embedded == "" {
		workflowPath = defaultPath
	} else {
		workflowPath = embedded
	}

	return workflowPath, fullStartMarker, true
}

// InjectGraph replaces the content between startMarker and endMarker in src
// with a newline-padded content block. Both markers must be present and start
// must appear before end. The markers themselves are preserved intact.
func InjectGraph(src, startMarker, endMarker, content string) (string, error) {
	startIdx := strings.Index(src, startMarker)
	if startIdx < 0 {
		return "", fmt.Errorf("start marker %q not found", startMarker)
	}
	afterStart := startIdx + len(startMarker)

	// Search for the end marker only in the text after the start marker so
	// that an end-before-start ordering is correctly reported as an error.
	relEnd := strings.Index(src[afterStart:], endMarker)
	if relEnd < 0 {
		return "", fmt.Errorf("end marker %q not found after start marker", endMarker)
	}
	endIdx := afterStart + relEnd

	return src[:afterStart] + "\n" + content + "\n" + src[endIdx:], nil
}

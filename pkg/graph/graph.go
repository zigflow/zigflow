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

	"github.com/serverlessworkflow/sdk-go/v3/model"
)

// Format represents a supported graph output format.
type Format string

const (
	// FormatMermaid renders the workflow as a Mermaid flowchart.
	FormatMermaid Format = "mermaid"
)

// Generator renders a workflow definition as a graph.
type Generator interface {
	Generate(wf *model.Workflow) (string, error)
}

// New returns a Generator for the given format.
func New(format Format) (Generator, error) {
	switch format {
	case FormatMermaid:
		return &mermaidGenerator{}, nil
	default:
		return nil, fmt.Errorf("unsupported graph format: %q", format)
	}
}

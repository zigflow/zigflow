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
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/examples"
)

type ListExamplesInput struct{}

type ExampleSummary struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type ListExamplesOutput struct {
	Examples []ExampleSummary `json:"examples"`
}

func (m *MCP) ListExamples(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListExamplesInput,
) (*mcp.CallToolResult, ListExamplesOutput, error) {
	catalog, err := examples.LoadCatalog(m.examplesFS, ".")
	if err != nil {
		return nil, ListExamplesOutput{}, fmt.Errorf("loading examples: %w", err)
	}

	summaries := make([]ExampleSummary, len(catalog))
	for i, ex := range catalog {
		summaries[i] = ExampleSummary{
			Name:        ex.Name,
			Title:       ex.Title,
			Description: ex.Description,
			Tags:        ex.Tags,
		}
	}

	return nil, ListExamplesOutput{Examples: summaries}, nil
}

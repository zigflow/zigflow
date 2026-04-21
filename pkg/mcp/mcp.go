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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCP struct {
	version string
}

func New(server *mcp.Server, version string) *MCP {
	m := &MCP{
		version: version,
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:  "get_schema",
		Title: "Get Schema",
		Description: "Returns the Zigflow workflow JSON schema for the current version. Use this to understand valid " +
			"workflow structure before generating or validating YAML.",
	}, m.GetSchema)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_example",
		Title:       "Get Example",
		Description: "Returns a Zigflow example by name, including its YAML content and metadata.",
	}, m.GetExample)

	mcp.AddTool(server, &mcp.Tool{
		Name:  "list_examples",
		Title: "List Examples",
		Description: "Lists the bundled Zigflow workflow examples with short descriptions and tags. " +
			"Use this to discover available examples before calling get_example.",
	}, m.ListExamples)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_workflow",
		Title:       "Validate Workflow",
		Description: "Validates a Zigflow workflow YAML string and returns structured errors by stage.",
	}, m.ValidateWorkflow)

	return m
}

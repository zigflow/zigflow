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
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/pkg/schema"
	"go.yaml.in/yaml/v2"
)

const (
	outputFormatJSON = "json"
	outputFormatYAML = "yaml"
)

type GetSchemaInput struct {
	Output string `json:"output" jsonschema:"Output format (json or yaml),enum=json,enum=yaml,default=json"`
}

type GetSchemaError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type GetSchemaOutput struct {
	Schema string           `json:"schema,omitempty"`
	Errors []GetSchemaError `json:"errors,omitempty"`
}

func (m *MCP) GetSchema(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetSchemaInput,
) (*mcp.CallToolResult, GetSchemaOutput, error) {
	format := strings.TrimSpace(input.Output)
	if format == "" {
		format = outputFormatJSON
	}

	if format != outputFormatJSON && format != outputFormatYAML {
		return nil, GetSchemaOutput{Errors: []GetSchemaError{{
			Stage:   "input",
			Message: `output must be "json" or "yaml"`,
		}}}, nil
	}

	s, err := schema.BuildSchema(m.version, format)
	if err != nil {
		return nil, GetSchemaOutput{}, err
	}

	var res []byte
	switch format {
	case outputFormatYAML:
		res, err = yaml.Marshal(s)
	default:
		res, err = json.MarshalIndent(s, "", "  ")
	}
	if err != nil {
		return nil, GetSchemaOutput{}, err
	}

	return nil, GetSchemaOutput{Schema: string(res)}, nil
}

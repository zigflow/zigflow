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
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/examples"
)

type GetExampleInput struct {
	Name string `json:"name" jsonschema:"Example name"`
}

type GetExampleError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type GetExampleOutput struct {
	Name        string            `json:"name,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Content     string            `json:"content,omitempty"`
	Errors      []GetExampleError `json:"errors,omitempty"`
}

func getExampleFromFS(fsys fs.FS, name string) (GetExampleOutput, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return GetExampleOutput{
			Errors: []GetExampleError{
				{
					Stage:   "input",
					Message: "name is required",
				},
			},
		}, nil
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	if err != nil {
		return GetExampleOutput{}, fmt.Errorf("loading examples: %w", err)
	}

	var found *examples.Example
	for i := range catalog {
		if catalog[i].Name == name {
			found = &catalog[i]
			break
		}
	}

	if found == nil {
		available := make([]string, len(catalog))
		for i, ex := range catalog {
			available[i] = ex.Name
		}

		return GetExampleOutput{Errors: []GetExampleError{{
			Stage:   "input",
			Message: fmt.Sprintf("example %q not found; available: %s", name, strings.Join(available, ", ")),
		}}}, nil
	}

	data, err := readExampleContent(fsys, found.Dir)
	if err != nil {
		return GetExampleOutput{}, fmt.Errorf("reading example %q: %w", name, err)
	}

	return GetExampleOutput{
		Name:        found.Name,
		Title:       found.Title,
		Description: found.Description,
		Tags:        found.Tags,
		Content:     string(data),
	}, nil
}

func (m *MCP) GetExample(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetExampleInput,
) (*mcp.CallToolResult, GetExampleOutput, error) {
	out, err := getExampleFromFS(examples.EmbeddedFS, input.Name)
	return nil, out, err
}

// readExampleContent reads the primary YAML file from the example directory,
// trying workflow.yaml first then info.yaml as a fallback.
func readExampleContent(fsys fs.FS, dir string) ([]byte, error) {
	for _, filename := range []string{"workflow.yaml", "info.yaml"} {
		data, err := fs.ReadFile(fsys, dir+"/"+filename)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filename, err)
		}

		return data, nil
	}

	return nil, fmt.Errorf("no workflow.yaml or info.yaml found in %q", dir)
}

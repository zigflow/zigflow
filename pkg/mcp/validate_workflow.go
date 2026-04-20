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
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

type ValidateWorkflowInput struct {
	YAML string `json:"yaml" jsonschema:"Workflow definition as a YAML string"`
}

type ValidateWorkflowError struct {
	Stage   string `json:"stage"`
	Path    string `json:"path,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

type ValidateWorkflowOutput struct {
	Valid  bool                    `json:"valid"`
	Errors []ValidateWorkflowError `json:"errors,omitempty"`
}

func (m *MCP) ValidateWorkflow(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ValidateWorkflowInput,
) (*mcp.CallToolResult, ValidateWorkflowOutput, error) {
	if strings.TrimSpace(input.YAML) == "" {
		return nil, ValidateWorkflowOutput{
			Errors: []ValidateWorkflowError{{
				Stage:   "input",
				Message: "yaml is required",
			}},
		}, nil
	}

	data := []byte(input.YAML)

	// Schema validation: non-schema errors from ValidateBytes are parse failures.
	if err := zigflow.ValidateBytes(data); err != nil {
		stage := "parse"
		if errors.Is(err, zigflow.ErrSchemaValidation) {
			stage = "schema"
		}

		return nil, ValidateWorkflowOutput{
			Errors: []ValidateWorkflowError{{Stage: stage, Message: err.Error()}},
		}, nil
	}

	// Load into model.
	wf, err := zigflow.LoadFromBytes(data)
	if err != nil {
		return nil, ValidateWorkflowOutput{
			Errors: []ValidateWorkflowError{{Stage: "load", Message: err.Error()}},
		}, nil
	}

	// Struct validation.
	validator, err := utils.NewValidator()
	if err != nil {
		return nil, ValidateWorkflowOutput{}, fmt.Errorf("creating validator: %w", err)
	}

	res, err := validator.ValidateStruct(wf)
	if err != nil {
		return nil, ValidateWorkflowOutput{}, fmt.Errorf("validating workflow: %w", err)
	}

	if len(res) > 0 {
		errs := make([]ValidateWorkflowError, len(res))
		for i, ve := range res {
			errs[i] = ValidateWorkflowError{
				Stage:   "struct",
				Rule:    ve.Key,
				Path:    ve.Path,
				Param:   ve.Param,
				Message: ve.Message,
			}
		}

		return nil, ValidateWorkflowOutput{Errors: errs}, nil
	}

	return nil, ValidateWorkflowOutput{Valid: true}, nil
}

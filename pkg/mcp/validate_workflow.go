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
	Stage         string `json:"stage"`
	Path          string `json:"path,omitempty"`
	Rule          string `json:"rule,omitempty"`
	Param         string `json:"param,omitempty"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message"`
	Documentation string `json:"documentation,omitempty"`
}

type ValidateWorkflowOutput struct {
	Valid  bool                    `json:"valid"`
	Errors []ValidateWorkflowError `json:"errors,omitempty"`
}

// validateBytesStage maps an error returned by zigflow.ValidateBytes onto the
// MCP validation stage that best describes it. Expression failures (both
// non-determinism and invalid syntax) are reported as "expression"; schema
// failures as "schema"; anything else is treated as a genuine parse failure.
func validateBytesStage(err error) string {
	switch {
	case errors.Is(err, zigflow.ErrSchemaValidation):
		return "schema"
	case errors.Is(err, zigflow.ErrNonDeterministicExpression),
		errors.Is(err, zigflow.ErrInvalidRuntimeExpression):
		return "expression"
	default:
		return "parse"
	}
}

// validateBytesErrors converts an error from zigflow.ValidateBytes into one or
// more structured MCP errors. Schema failures are expanded into per-field
// errors so each carries its own path and, where the field is recognised, a
// stable code and derived documentation URL. All other failures keep their
// single-error form, classified by stage.
func validateBytesErrors(err error) []ValidateWorkflowError {
	var schemaErr *zigflow.SchemaValidationError
	if errors.As(err, &schemaErr) && len(schemaErr.Errors) > 0 {
		out := make([]ValidateWorkflowError, 0, len(schemaErr.Errors))
		for _, se := range schemaErr.Errors {
			code := utils.CodeForPath(se.Location)
			out = append(out, ValidateWorkflowError{
				Stage:         "schema",
				Path:          se.Location,
				Code:          code,
				Message:       se.Message,
				Documentation: utils.DocumentationURL(code),
			})
		}
		return out
	}

	return []ValidateWorkflowError{{Stage: validateBytesStage(err), Message: err.Error()}}
}

func (m *MCP) ValidateWorkflow(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ValidateWorkflowInput,
) (*mcp.CallToolResult, ValidateWorkflowOutput, error) {
	if strings.TrimSpace(input.YAML) == "" {
		return nil, ValidateWorkflowOutput{
			Errors: []ValidateWorkflowError{{
				Stage:   stageInput,
				Message: "yaml is required",
			}},
		}, nil
	}

	data := []byte(input.YAML)

	// ValidateBytes can fail for several distinct reasons; classify them so the
	// reported stage is accurate rather than always "parse".
	if err := zigflow.ValidateBytes(data); err != nil {
		return nil, ValidateWorkflowOutput{Errors: validateBytesErrors(err)}, nil
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
				Stage:         "struct",
				Rule:          ve.Key,
				Path:          ve.Path,
				Param:         ve.Param,
				Code:          ve.Code,
				Message:       ve.Message,
				Documentation: utils.DocumentationURL(ve.Code),
			}
		}

		return nil, ValidateWorkflowOutput{Errors: errs}, nil
	}

	return nil, ValidateWorkflowOutput{Valid: true}, nil
}

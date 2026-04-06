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

package cmd

import (
	"errors"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog/log"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	zigschema "github.com/zigflow/zigflow/pkg/schema"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

func newValidateCmd() *cobra.Command {
	var opts struct {
		OutputJSON bool
	}

	cmd := &cobra.Command{
		Use:   "validate <workflow-file>",
		Short: "Validate a Zigflow workflow file",
		Long: `Validate a Zigflow workflow definition written in the Zigflow DSL.

This command runs both JSON Schema validation and Go/runtime validation against
the provided workflow file. Both must pass for the command to succeed.

Use the subcommands for isolated validation:
  validate schema <file>    JSON Schema validation only
  validate runtime <file>   Go/runtime validation only

Validation checks:
  - JSON Schema: field names, required properties, value formats,
    mutual-exclusion constraints and unsupported constructs
  - Runtime: DSL version support, structural requirements and
    reference checks according to the Zigflow specification

The command exits with a non-zero status code if either validation fails,
making it suitable for use in scripts, CI pipelines and automated tooling.

Arguments:
  workflow-file   Path to the Zigflow workflow file to validate`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			schemaResult, err := buildSchemaResult(filePath)
			if err != nil {
				return err
			}

			runtimeResult, err := buildRuntimeResult(filePath)
			if err != nil {
				return err
			}

			result := utils.CombinedValidationResult{
				File:    filePath,
				Valid:   schemaResult.Valid && runtimeResult.Valid,
				Schema:  schemaResult,
				Runtime: runtimeResult,
			}

			if opts.OutputJSON {
				_ = utils.RenderJSONCombined(cmd.OutOrStdout(), &result)
			} else {
				utils.RenderHumanCombined(cmd.OutOrStdout(), &result)
			}

			if !result.Valid {
				return gh.FatalError{
					Msg:    "Validation failed",
					Logger: log.Trace,
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(
		&opts.OutputJSON, "output-json",
		viper.GetBool("output_json"), "Output as JSON",
	)

	cmd.AddCommand(
		newValidateSchemaCmd(),
		newValidateRuntimeCmd(),
	)

	return cmd
}

// runSingleValidation executes one validation step, renders the result, and
// returns a FatalError when the workflow is invalid. Used by both the
// validate schema and validate runtime subcommands to avoid repetition.
func runSingleValidation(
	cmd *cobra.Command,
	filePath string,
	outputJSON bool,
	build func(string) (utils.ValidationResult, error),
	failMsg string,
) error {
	result, err := build(filePath)
	if err != nil {
		return err
	}

	if outputJSON {
		_ = utils.RenderJSON(cmd.OutOrStdout(), result)
	} else {
		utils.RenderHuman(cmd.OutOrStdout(), result)
	}

	if !result.Valid {
		return gh.FatalError{
			Msg:    failMsg,
			Logger: log.Trace,
		}
	}

	return nil
}

// buildSchemaResult compiles the Zigflow JSON Schema and validates the given
// file against it. A non-nil error indicates a fatal condition (file unreadable,
// malformed YAML, schema compilation failure) rather than a validation failure.
// Structural validation errors are captured in the returned ValidationResult.
func buildSchemaResult(filePath string) (utils.ValidationResult, error) {
	sch, err := zigschema.CompileSchema(Version)
	if err != nil {
		return utils.ValidationResult{File: filePath}, gh.FatalError{
			Cause: err,
			Msg:   "Failed to build schema",
		}
	}

	return buildSchemaResultFromSchema(sch, filePath), nil
}

// buildSchemaResultFromSchema validates filePath against a pre-compiled schema.
// This avoids re-compiling the schema on every call when validating multiple
// files in a single invocation.
func buildSchemaResultFromSchema(sch *jsonschema.Schema, filePath string) utils.ValidationResult {
	result := utils.ValidationResult{File: filePath}

	if err := zigschema.ValidateFile(sch, filePath); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			for _, e := range ve.BasicOutput().Errors {
				loc := e.InstanceLocation
				if loc == "" {
					loc = "(root)"
				}

				result.Errors = append(result.Errors, utils.ValidationErrors{
					Path:    loc,
					Message: e.Error,
				})
			}
		} else {
			result.Errors = append(result.Errors, utils.ValidationErrors{
				Path:    "(root)",
				Message: err.Error(),
			})
		}
	} else {
		result.Valid = true
	}

	return result
}

// buildRuntimeResult loads the workflow file and validates it using the
// Zigflow runtime validator. A non-nil error indicates a fatal condition
// (file unreadable, unsupported format, validator initialisation failure).
// Structural validation errors are captured in the returned ValidationResult.
func buildRuntimeResult(filePath string) (utils.ValidationResult, error) {
	result := utils.ValidationResult{File: filePath}

	workflowDefinition, err := zigflow.LoadFromFile(filePath)
	if err != nil {
		return result, gh.FatalError{
			Cause: err,
			Msg:   "Unable to load workflow file",
		}
	}

	validator, err := utils.NewValidator()
	if err != nil {
		return result, gh.FatalError{
			Cause: err,
			Msg:   "Error creating validator",
		}
	}

	if res, err := validator.ValidateStruct(workflowDefinition); err != nil {
		return result, gh.FatalError{
			Cause: err,
			Msg:   "Error creating validation stack",
		}
	} else if len(res) > 0 {
		result.Errors = res
	} else {
		result.Valid = true
	}

	return result, nil
}

type Error struct {
	Path    string `json:"path"`
	Rule    string `json:"rule"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

type Result struct {
	Valid  bool    `json:"valid"`
	File   string  `json:"file"`
	Errors []Error `json:"errors,omitempty"`
}

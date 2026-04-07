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
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

This command parses the provided workflow file and verifies that it is
syntactically valid and structurally correct according to the Zigflow
specification. It does not execute the workflow or connect to Temporal.

Validation includes:
  - DSL syntax checks
  - Schema and structural validation
  - Reference and dependency checks where applicable

The command exits with a non-zero status code if validation fails,
making it suitable for use in scripts, CI pipelines and automated tooling.

Arguments:
  workflow-file   Path to the Zigflow workflow file to validate`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			result := utils.ValidationResult{
				File: filePath,
			}

			if err := zigflow.ValidateFile(filePath); err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Schema validation failed",
				}
			}

			workflowDefinition, err := zigflow.LoadFromFile(filePath)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Unable to load workflow file",
				}
			}

			validator, err := utils.NewValidator()
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error creating validator",
				}
			}

			if res, err := validator.ValidateStruct(workflowDefinition); err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error creating validation stack",
				}
			} else if res != nil {
				result.Errors = res
			} else {
				result.Valid = true
			}

			if opts.OutputJSON {
				_ = utils.RenderJSON(os.Stdout, result)
			} else {
				utils.RenderHuman(os.Stdout, result)
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

	return cmd
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

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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newValidateRuntimeCmd() *cobra.Command {
	var opts struct {
		OutputJSON bool
	}

	cmd := &cobra.Command{
		Use:   "runtime <workflow-file>",
		Short: "Validate a workflow file using the Zigflow runtime validator",
		Long: `Validate a workflow file using the Zigflow Go/runtime validator.

The runtime validator checks DSL version support, structural requirements and
reference correctness according to the Zigflow specification. It does not
connect to Temporal or execute the workflow.

This command does not run JSON Schema validation. Use this subcommand to
isolate runtime issues from structural schema failures.
To run both validations together, use the parent validate command:

  zigflow validate <workflow-file>

The command exits with a non-zero status code if validation fails, making it
suitable for use in scripts, CI pipelines and automated tooling.

Arguments:
  workflow-file   Path to the Zigflow workflow file to validate`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSingleValidation(cmd, args[0], opts.OutputJSON, buildRuntimeResult, "Runtime validation failed")
		},
	}

	cmd.Flags().BoolVar(
		&opts.OutputJSON, "output-json",
		viper.GetBool("output_json"), "Output as JSON",
	)

	return cmd
}

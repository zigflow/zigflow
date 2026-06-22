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
	"fmt"
	"io"

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
			return runValidateCmd(cmd, args[0], opts.OutputJSON)
		},
	}

	cmd.Flags().BoolVar(
		&opts.OutputJSON, "output-json",
		viper.GetBool("output_json"), "Output as JSON",
	)

	return cmd
}

// runValidateCmd validates the workflow file at filePath, rendering the result
// to the command's output (human-readable by default, JSON when outputJSON is
// set) and returning a fatal error when validation fails.
func runValidateCmd(cmd *cobra.Command, filePath string, outputJSON bool) error {
	result := utils.ValidationResult{
		File: filePath,
	}

	out := cmd.OutOrStdout()

	if err := zigflow.ValidateFile(filePath); err != nil {
		// Render known validation failures (schema, determinism) for
		// humans by default — concise location + message — and suppress
		// the noisy nested error/log output by returning a trace-level
		// FatalError without a Cause. --output-json keeps the structured,
		// machine-readable form.
		var (
			schemaErr  *zigflow.SchemaValidationError
			invalidErr *zigflow.InvalidRuntimeExpressionError
			ndErr      *zigflow.NonDeterministicExpressionError
		)
		switch {
		case errors.As(err, &schemaErr):
			if outputJSON {
				result.Errors = schemaValidationErrors(schemaErr.Errors)
				_ = utils.RenderJSON(out, result)
			} else {
				renderSchemaFailure(out, filePath, schemaErr.Errors)
			}

		case errors.As(err, &invalidErr):
			if outputJSON {
				result.Errors = invalidExpressionValidationErrors(invalidErr.Expressions)
				_ = utils.RenderJSON(out, result)
			} else {
				renderInvalidExpressionFailure(out, filePath, invalidErr.Expressions)
			}

		case errors.As(err, &ndErr):
			if outputJSON {
				result.Errors = determinismValidationErrors(ndErr.Expressions)
				_ = utils.RenderJSON(out, result)
			} else {
				renderDeterminismFailure(out, filePath, ndErr.Expressions)
			}

		default:
			return gh.FatalError{
				Cause: err,
				Msg:   "Workflow validation failed",
			}
		}

		return gh.FatalError{
			Msg:    "Workflow validation failed",
			Logger: log.Trace,
		}
	}

	workflowDefinition, _, err := zigflow.LoadFromFile(filePath)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   errMsgUnableToLoadWorkflowFile,
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

	if outputJSON {
		_ = utils.RenderJSON(out, result)
	} else {
		utils.RenderHuman(out, result)
	}

	if !result.Valid {
		return gh.FatalError{
			Msg:    "Validation failed",
			Logger: log.Trace,
		}
	}

	return nil
}

// Field labels shared by the human-readable validation renderers.
const (
	labelLocation   = "Location"
	labelExpression = "Expression"
)

// validationField is one labelled line of a validation problem (e.g.
// "Location" / "$.do[0]"). An entry is the ordered set of fields describing a
// single problem.
type validationField struct {
	label string
	value string
}

// renderValidationFailure prints a concise, human-readable validation failure
// for any validation kind. A single entry is shown as labelled blocks; multiple
// entries are bulleted, with the first field as the bullet and the rest
// indented beneath it. Labels are only shown in the single-entry form.
func renderValidationFailure(w io.Writer, file, singular, plural string, entries [][]validationField) {
	_, _ = fmt.Fprintf(w, "❌ %s is invalid\n\n", file)

	if len(entries) == 1 {
		_, _ = fmt.Fprintf(w, "%s\n\n", singular)
		for i, f := range entries[0] {
			if i > 0 {
				_, _ = fmt.Fprintln(w)
			}
			_, _ = fmt.Fprintf(w, "%s:\n  %s\n", f.label, f.value)
		}
		return
	}

	_, _ = fmt.Fprintf(w, "%s\n\n", plural)
	for i, entry := range entries {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintf(w, "• %s\n", entry[0].value)
		for _, f := range entry[1:] {
			_, _ = fmt.Fprintf(w, "  %s\n", f.value)
		}
	}
}

// renderSchemaFailure prints a concise, human-readable schema validation error
// focused on where the problem is and what it is.
func renderSchemaFailure(w io.Writer, file string, errs []zigflow.SchemaError) {
	entries := make([][]validationField, len(errs))
	for i, e := range errs {
		entries[i] = []validationField{
			{label: labelLocation, value: e.Location},
			{label: "Message", value: e.Message},
		}
	}

	renderValidationFailure(
		w, file,
		"Schema validation failed",
		"Schema validation errors found:",
		entries,
	)
}

// schemaValidationErrors maps the structured schema failures onto the shared
// ValidationErrors shape so JSON output stays machine-readable and consistent
// with determinism/structural validation errors.
func schemaValidationErrors(errs []zigflow.SchemaError) []utils.ValidationErrors {
	out := make([]utils.ValidationErrors, 0, len(errs))
	for _, e := range errs {
		out = append(out, utils.ValidationErrors{
			Key:     "schema_validation",
			Message: e.Message,
			Path:    e.Location,
		})
	}
	return out
}

// renderDeterminismFailure prints a concise, human-readable determinism error
// focused on where the offending expression is and what it is.
func renderDeterminismFailure(w io.Writer, file string, exprs []zigflow.NonDeterministicExpression) {
	entries := make([][]validationField, len(exprs))
	for i, e := range exprs {
		entries[i] = []validationField{
			{label: labelLocation, value: e.Path},
			{label: labelExpression, value: e.Expression},
		}
	}

	renderValidationFailure(
		w, file,
		"Non-deterministic expression",
		"Non-deterministic expressions found:",
		entries,
	)
}

// determinismValidationErrors maps the structured determinism failures onto the
// shared ValidationErrors shape so JSON output stays machine-readable and
// consistent with schema/structural validation errors.
func determinismValidationErrors(exprs []zigflow.NonDeterministicExpression) []utils.ValidationErrors {
	errs := make([]utils.ValidationErrors, 0, len(exprs))
	for _, e := range exprs {
		errs = append(errs, utils.ValidationErrors{
			Key:     "non_deterministic_expression",
			Message: fmt.Sprintf("non-deterministic expression %q", e.Expression),
			Path:    e.Path,
		})
	}
	return errs
}

// renderInvalidExpressionFailure prints a concise, human-readable invalid
// runtime expression error focused on the location, the offending expression,
// and the underlying parse error.
func renderInvalidExpressionFailure(w io.Writer, file string, exprs []zigflow.InvalidRuntimeExpression) {
	entries := make([][]validationField, len(exprs))
	for i, e := range exprs {
		entries[i] = []validationField{
			{label: labelLocation, value: e.Path},
			{label: labelExpression, value: e.Expression},
			{label: "Error", value: e.Err.Error()},
		}
	}

	renderValidationFailure(
		w, file,
		"Invalid runtime expression",
		"Invalid runtime expressions found:",
		entries,
	)
}

// invalidExpressionValidationErrors maps the structured invalid-expression
// failures onto the shared ValidationErrors shape so JSON output stays
// machine-readable and consistent with the other validation errors.
func invalidExpressionValidationErrors(exprs []zigflow.InvalidRuntimeExpression) []utils.ValidationErrors {
	errs := make([]utils.ValidationErrors, 0, len(exprs))
	for _, e := range exprs {
		errs = append(errs, utils.ValidationErrors{
			Key:     "invalid_runtime_expression",
			Message: fmt.Sprintf("invalid runtime expression %q: %s", e.Expression, e.Err),
			Path:    e.Path,
		})
	}
	return errs
}

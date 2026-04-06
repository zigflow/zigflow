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

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

var ErrUnknownValidationError = fmt.Errorf("unknown validation error")

type ValidationResult struct {
	Valid  bool               `json:"valid"`
	File   string             `json:"file"`
	Errors []ValidationErrors `json:"errors,omitempty"`
}

type ValidationErrors struct {
	Key     string               `json:"key"`
	Message string               `json:"message"`
	Path    string               `json:"path"`
	Param   string               `json:"param,omitempty"`
	Error   validator.FieldError `json:"-"`
}

type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

func (v *Validator) ValidateStruct(data any) ([]ValidationErrors, error) {
	// Store validation errors
	var vErrs []ValidationErrors

	// Check the data
	if err := v.validate.Struct(data); err != nil {
		if validationError, ok := err.(validator.ValidationErrors); !ok {
			return nil, fmt.Errorf("%s: %w", ErrUnknownValidationError, err)
		} else {
			for _, e := range validationError {
				vErrs = append(vErrs, ValidationErrors{
					Key:     e.Tag(),
					Message: e.Translate(v.trans),
					Path:    e.StructNamespace(),
					Param:   e.Param(),
					Error:   e,
				})
			}
		}
	}

	return vErrs, nil
}

func NewValidator() (*Validator, error) {
	enTrans := en.New()
	uni := ut.New(enTrans)
	trans, _ := uni.GetTranslator(enTrans.Locale())

	validate := model.GetValidator()

	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, fmt.Errorf("error registering validator translations: %w", err)
	}

	return &Validator{
		validate: validate,
	}, nil
}

// CombinedValidationResult holds the outcome of running both schema and
// runtime validation against a single workflow file.
type CombinedValidationResult struct {
	Valid   bool             `json:"valid"`
	File    string           `json:"file"`
	Schema  ValidationResult `json:"schema"`
	Runtime ValidationResult `json:"runtime"`
}

func RenderHuman(w io.Writer, result ValidationResult) {
	if result.Valid {
		_, _ = fmt.Fprintf(w, "✅ %s is valid\n", result.File)
		return
	}

	_, _ = fmt.Fprintf(w, "❌ Validation failed for %s\n\n", result.File)
	_, _ = fmt.Fprintf(w, "%d validation error(s):\n\n", len(result.Errors))

	for i, err := range result.Errors {
		msg := err.Message
		if err.Error != nil {
			msg = humanMessage(err.Error)
		}

		_, _ = fmt.Fprintf(w, "%d. %s: %s\n", i+1, err.Path, msg)
	}
}

// maxHumanErrors is the maximum number of validation errors shown per failing
// section in human-readable combined output. The full list is always available
// via --output-json.
const maxHumanErrors = 20

// RenderHumanCombined prints a combined schema and runtime validation result
// in human-readable form.
//
// Layout:
//
//	Validation failed for <file>
//
//	Overall result: failed
//
//	* Schema validation: failed (N errors)
//	* Runtime validation: passed
//
//	First 20 schema errors:
//	1. ...
//	... and X more errors. Use --output-json for full output.
func RenderHumanCombined(w io.Writer, result *CombinedValidationResult) {
	if result.Valid {
		_, _ = fmt.Fprintf(w, "✅ %s is valid\n", result.File)
		return
	}

	_, _ = fmt.Fprintf(w, "❌ Validation failed for %s\n\n", result.File)
	_, _ = fmt.Fprintf(w, "Overall result: failed\n\n")

	_, _ = fmt.Fprintf(w, "* Schema validation: %s\n", combinedSectionSummary(result.Schema))
	_, _ = fmt.Fprintf(w, "* Runtime validation: %s\n", combinedSectionSummary(result.Runtime))

	if !result.Schema.Valid {
		_, _ = fmt.Fprintln(w)
		renderCappedErrors(w, "schema", result.Schema.Errors)
	}

	if !result.Runtime.Valid {
		_, _ = fmt.Fprintln(w)
		renderCappedErrors(w, "runtime", result.Runtime.Errors)
	}
}

// combinedSectionSummary returns a one-line status string for a validation
// section, e.g. "passed" or "failed (3 errors)".
func combinedSectionSummary(result ValidationResult) string {
	if result.Valid {
		return "passed"
	}

	return fmt.Sprintf("failed (%d error(s))", len(result.Errors))
}

// renderCappedErrors prints up to maxHumanErrors errors from a section.
// label should be lowercase (e.g. "schema", "runtime").
// When the list is truncated a hint to use --output-json is appended.
func renderCappedErrors(w io.Writer, label string, errs []ValidationErrors) {
	n := len(errs)
	shown := n

	if shown > maxHumanErrors {
		shown = maxHumanErrors
		_, _ = fmt.Fprintf(w, "First %d %s errors:\n\n", shown, label)
	} else {
		_, _ = fmt.Fprintf(w, "%s errors:\n\n", strings.ToUpper(label[:1])+label[1:])
	}

	for i, e := range errs[:shown] {
		msg := e.Message
		if e.Error != nil {
			msg = humanMessage(e.Error)
		}

		_, _ = fmt.Fprintf(w, "%d. %s: %s\n", i+1, e.Path, msg)
	}

	if n > maxHumanErrors {
		_, _ = fmt.Fprintf(w, "\n... and %d more errors. Use --output-json for full output.\n", n-shown)
	}
}

func RenderJSON(w io.Writer, result ValidationResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// RenderJSONCombined writes a combined validation result as indented JSON.
func RenderJSONCombined(w io.Writer, result *CombinedValidationResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func humanMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"

	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())

	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())

	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())

	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())

	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())

	case "min":
		return fmt.Sprintf("must have minimum length of %s", fe.Param())

	case "max":
		return fmt.Sprintf("must have maximum length of %s", fe.Param())

	default:
		return fmt.Sprintf("failed validation (%s)", fe.Tag())
	}
}

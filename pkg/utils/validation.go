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

func RenderHuman(w io.Writer, result ValidationResult) {
	if result.Valid {
		_, _ = fmt.Fprintf(w, "✅ %s is valid\n", result.File)
		return
	}

	_, _ = fmt.Fprintf(w, "❌ Validation failed for %s\n\n", result.File)
	_, _ = fmt.Fprintf(w, "%d validation error(s):\n\n", len(result.Errors))

	for i, err := range result.Errors {
		_, _ = fmt.Fprintf(w, "%d. %s: %s\n", i+1, err.Path, humanMessage(err.Error))
	}
}

func RenderJSON(w io.Writer, result ValidationResult) error {
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

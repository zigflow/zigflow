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

package zigflow

import (
	"fmt"
	"strings"
)

// SchemaError identifies a single JSON Schema violation: where it is in the
// document (a best-effort instance-style path) and what the problem is. It is
// the machine-readable unit behind a schema validation failure.
type SchemaError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

// SchemaValidationError wraps the underlying JSON Schema validation failure. It
// preserves the original error message (Error() is unchanged from the previous
// fmt.Errorf wrapping) and matches ErrSchemaValidation via errors.Is, while
// exposing the failures in structured form for callers that want to render or
// process them without parsing the message.
type SchemaValidationError struct {
	Errors []SchemaError
	cause  error
}

func (e *SchemaValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ErrSchemaValidation, e.cause)
}

func (e *SchemaValidationError) Unwrap() error { return ErrSchemaValidation }

// newSchemaValidationError builds a SchemaValidationError from the error
// returned by schema.ValidateDocument, parsing it into structured entries for
// human-readable rendering. The original error is retained verbatim so the
// message and wrapping behaviour are unchanged.
func newSchemaValidationError(cause error) *SchemaValidationError {
	return &SchemaValidationError{
		Errors: parseSchemaErrors(cause),
		cause:  cause,
	}
}

// parseSchemaErrors turns the (single, colon-nested) error string produced by
// google/jsonschema-go into one or more structured entries. The library does
// not expose instance locations, so the location is derived best-effort from
// the deepest schema keyword-location in the message. errors.Join'd failures
// are split on newlines.
func parseSchemaErrors(err error) []SchemaError {
	if err == nil {
		return nil
	}

	var out []SchemaError
	for _, line := range strings.Split(err.Error(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// The library nests context as repeated "validating <schema>: "
		// prefixes, where <schema> is the root id or a schema pointer such as
		// "/properties/document". Strip those, keeping the deepest pointer as
		// the location, and treat the remainder as the message.
		location := ""
		rest := line
		for strings.HasPrefix(rest, "validating ") {
			after := rest[len("validating "):]
			idx := strings.Index(after, ": ")
			if idx < 0 {
				break
			}
			token := after[:idx]
			rest = after[idx+2:]
			if strings.HasPrefix(token, "/") {
				location = token
			}
		}

		out = append(out, SchemaError{
			Location: schemaPointerToInstancePath(location),
			Message:  rest,
		})
	}

	return out
}

// schemaPointerToInstancePath converts a JSON Schema keyword-location pointer
// (e.g. "/properties/document/properties/workflowType") into a best-effort
// instance path (e.g. "$.document.workflowType"). It is intentionally lenient:
// applicator keywords are skipped and unrecognised segments are ignored so a
// useful, stable location is produced even as the schema evolves.
func schemaPointerToInstancePath(pointer string) string {
	if pointer == "" {
		return "$"
	}

	parts := strings.Split(strings.Trim(pointer, "/"), "/")
	path := "$"
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "properties", "patternProperties":
			if i+1 < len(parts) {
				path += "." + parts[i+1]
				i++
			}
		case "items", "prefixItems", "additionalItems", "contains":
			path += "[]"
		case "$defs", "definitions", "allOf", "anyOf", "oneOf", "not", "if", "then", "else":
			// Applicator / definition keywords carry a following index or name
			// that is not part of the instance path; skip it.
			if i+1 < len(parts) {
				i++
			}
		default:
			// Unknown keyword (e.g. unevaluatedProperties): ignore it rather
			// than guess at an instance segment.
		}
	}

	return path
}

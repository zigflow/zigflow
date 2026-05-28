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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaRoot is the prefix google/jsonschema-go puts on every error: the root
// schema id. Shared by the test cases to keep the fixtures readable.
const schemaRoot = "validating https://zigflow.dev/schema.json: "

func TestSchemaValidationError_PreservesWrapping(t *testing.T) {
	cause := errors.New(`validating https://zigflow.dev/schema.json: required: missing properties: ["do"]`)
	err := newSchemaValidationError(cause)

	// errors.Is must still match the sentinel so existing callers (CLI, MCP)
	// keep working.
	assert.ErrorIs(t, err, ErrSchemaValidation)

	// Error() must match the previous fmt.Errorf("%w: %s", ...) wrapping.
	want := fmt.Sprintf("%s: %s", ErrSchemaValidation, cause)
	assert.Equal(t, want, err.Error())
}

func TestParseSchemaErrors(t *testing.T) {
	tests := []struct {
		Name         string
		Err          error
		WantLen      int
		WantLocation string
		WantMessage  string
	}{
		{
			Name:         "root-level required property",
			Err:          errors.New(schemaRoot + `required: missing properties: ["do"]`),
			WantLen:      1,
			WantLocation: "$",
			WantMessage:  `required: missing properties: ["do"]`,
		},
		{
			Name:         "nested document property",
			Err:          errors.New(schemaRoot + `validating /properties/document: required: missing properties: ["workflowType"]`),
			WantLen:      1,
			WantLocation: "$.document",
			WantMessage:  `required: missing properties: ["workflowType"]`,
		},
		{
			Name: "unknown keyword keeps parent location",
			Err: errors.New(schemaRoot +
				`validating /properties/document: ` +
				`validating /properties/document/unevaluatedProperties: ` +
				`not: validated against <anonymous schema>`),
			WantLen:      1,
			WantLocation: "$.document",
			WantMessage:  "not: validated against <anonymous schema>",
		},
		{
			Name:         "items become an index segment",
			Err:          errors.New(schemaRoot + `validating /properties/do/items/properties/type: type: 1 has type "number", want "string"`),
			WantLen:      1,
			WantLocation: "$.do[].type",
			WantMessage:  `type: 1 has type "number", want "string"`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got := parseSchemaErrors(test.Err)
			require.Len(t, got, test.WantLen)
			assert.Equal(t, test.WantLocation, got[0].Location)
			assert.Equal(t, test.WantMessage, got[0].Message)
		})
	}
}

func TestParseSchemaErrors_MultipleJoined(t *testing.T) {
	// errors.Join renders sub-errors on separate lines.
	err := errors.New(
		schemaRoot + `validating /properties/document: required: missing properties: ["workflowType"]` +
			"\n" +
			schemaRoot + `validating /properties/do: type: 1 has type "number", want "array"`,
	)

	got := parseSchemaErrors(err)
	require.Len(t, got, 2)
	assert.Equal(t, "$.document", got[0].Location)
	assert.Equal(t, "$.do", got[1].Location)
}

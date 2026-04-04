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

// Package schema provides a programmatic JSON Schema builder for Zigflow workflow
// definitions. It generates a JSON Schema that reflects only the features Zigflow
// actually supports, rather than the full Serverless Workflow specification.
package schema

import (
	"bytes"
	"encoding/json"
)

// Schema is a JSON Schema node. Known schema keywords are typed as struct
// fields; struct field declaration order determines JSON key order, so
// $schema and $id are always marshalled first. Nested schemas are stored as
// pointers to avoid copying large structs through the builder API. Any
// additional keywords not covered by the declared fields can be placed in
// Extra, which is merged into the JSON output after all declared fields.
type Schema struct {
	// Identity – marshalled first.
	SchemaURI string `json:"$schema,omitempty"`
	ID        string `json:"$id,omitempty"`

	// Reference.
	Ref string `json:"$ref,omitempty"`

	// Annotations.
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// Type assertion.
	Type  string `json:"type,omitempty"`
	Const any    `json:"const,omitempty"`
	Enum  []any  `json:"enum,omitempty"`

	// Object keywords.
	Properties            map[string]*Schema `json:"properties,omitempty"`
	Required              []string           `json:"required,omitempty"`
	AdditionalProperties  any                `json:"additionalProperties,omitempty"`
	UnevaluatedProperties *bool              `json:"unevaluatedProperties,omitempty"`
	MinProperties         *int               `json:"minProperties,omitempty"`
	MaxProperties         *int               `json:"maxProperties,omitempty"`

	// Array keywords.
	Items *Schema `json:"items,omitempty"`

	// String keywords.
	Pattern string `json:"pattern,omitempty"`

	// Applicator keywords.
	OneOf []*Schema `json:"oneOf,omitempty"`
	AnyOf []*Schema `json:"anyOf,omitempty"`
	AllOf []*Schema `json:"allOf,omitempty"`

	// Schema definitions.
	Defs map[string]*Schema `json:"$defs,omitempty"`

	// Extra holds additional schema keywords not covered by the fields above.
	// These are merged into the JSON output after all declared fields.
	Extra map[string]any `json:"-"`
}

// MarshalJSON serialises the Schema to JSON. Declared fields are emitted in
// their struct declaration order (so $schema and $id always come first).
// Fields in Extra are appended after all declared fields.
func (s *Schema) MarshalJSON() ([]byte, error) {
	// schemaAlias does not inherit *Schema's methods, so delegating to it
	// for the base encoding avoids infinite recursion back into MarshalJSON.
	type schemaAlias Schema

	base, err := json.Marshal((*schemaAlias)(s))
	if err != nil {
		return nil, err
	}
	if len(s.Extra) == 0 {
		return base, nil
	}

	extra, err := json.Marshal(s.Extra)
	if err != nil {
		return nil, err
	}

	// Splice extra fields into the object: remove the trailing } from base,
	// remove the leading { from extra, and join with a comma.
	if bytes.Equal(base, []byte("{}")) {
		return extra, nil
	}
	out := make([]byte, 0, len(base)+1+len(extra)-1)
	out = append(out, base[:len(base)-1]...)
	out = append(out, ',')
	out = append(out, extra[1:]...)
	return out, nil
}

// boolPtr returns a pointer to b. Used for optional boolean schema fields
// such as UnevaluatedProperties where the zero value must be distinguishable
// from "not set".
func boolPtr(b bool) *bool { return &b }

// intPtr returns a pointer to i. Used for optional integer schema fields
// such as MinProperties where zero must be distinguishable from "not set".
func intPtr(i int) *int { return &i }

// Object builds an object schema with the given properties. Pass required
// field names as variadic arguments after the properties map.
func Object(props map[string]*Schema, required ...string) *Schema {
	s := &Schema{Type: "object", Properties: props}
	if len(required) > 0 {
		s.Required = required
	}
	return s
}

// String returns a string schema.
func String() *Schema { return &Schema{Type: "string"} }

// StringEnum returns a string schema restricted to a fixed set of values.
func StringEnum(values ...string) *Schema {
	enum := make([]any, len(values))
	for i, v := range values {
		enum[i] = v
	}
	return &Schema{Type: "string", Enum: enum}
}

// StringConst returns a string schema restricted to a single constant value.
func StringConst(value string) *Schema {
	return &Schema{Type: "string", Const: value}
}

// Integer returns an integer schema.
func Integer() *Schema { return &Schema{Type: "integer"} }

// Number returns a number schema (integer or float).
func Number() *Schema { return &Schema{Type: "number"} }

// Boolean returns a boolean schema.
func Boolean() *Schema { return &Schema{Type: "boolean"} }

// Any returns an empty schema, which matches any value.
func Any() *Schema { return &Schema{} }

// Array returns an array schema where each item must match item.
func Array(item *Schema) *Schema {
	return &Schema{Type: "array", Items: item}
}

// OneOf returns a schema that must match exactly one of the provided schemas.
func OneOf(schemas ...*Schema) *Schema {
	items := make([]*Schema, len(schemas))
	copy(items, schemas)
	return &Schema{OneOf: items}
}

// AnyOf returns a schema that must match at least one of the provided schemas.
func AnyOf(schemas ...*Schema) *Schema {
	items := make([]*Schema, len(schemas))
	copy(items, schemas)
	return &Schema{AnyOf: items}
}

// AllOf returns a schema that must match all of the provided schemas.
func AllOf(schemas ...*Schema) *Schema {
	items := make([]*Schema, len(schemas))
	copy(items, schemas)
	return &Schema{AllOf: items}
}

// Ref returns a $ref schema pointing to the given JSON Pointer location.
func Ref(ref string) *Schema { return &Schema{Ref: ref} }

// WithDescription returns a copy of s with the given description set.
func WithDescription(s *Schema, description string) *Schema {
	out := *s
	out.Description = description
	return &out
}

// WithTitle returns a copy of s with the given title set.
func WithTitle(s *Schema, title string) *Schema {
	out := *s
	out.Title = title
	return &out
}

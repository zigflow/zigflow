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
	"fmt"
	"reflect"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
)

// outputSchemaFor builds the schema mcp.AddTool would otherwise infer for T,
// with every nullable-array property rewritten into the equivalent `anyOf`
// form.
//
// AddTool only reflects over the handler's output type when Tool.OutputSchema
// is nil, so assigning this on a tool literal makes the SDK use it verbatim.
func outputSchemaFor[T any]() *jsonschema.Schema {
	rt := reflect.TypeFor[T]()

	schema, err := jsonschema.ForType(rt, &jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Errorf("inferring output schema for %s: %w", rt, err))
	}

	nullableArraysToAnyOf(schema)

	return schema
}

// nullableArraysToAnyOf rewrites `"type": ["null", "array"]` into
// `anyOf: [{"type": "null"}, {"type": "array", ...}]` throughout schema.
//
// jsonschema-go infers every Go slice as that nullable pair, because a nil
// slice marshals to JSON null (infer.go:219-223). Both forms accept exactly
// the same documents, but several MCP clients assume "type" is a single
// string and reject or mishandle the array form.
func nullableArraysToAnyOf(schema *jsonschema.Schema) {
	if schema == nil {
		return
	}

	// Recurse first, so a slice nested inside another slice's element schema
	// is rewritten before the parent moves that schema under anyOf.
	for _, prop := range schema.Properties {
		nullableArraysToAnyOf(prop)
	}

	for _, sub := range slices.Concat(schema.AnyOf, schema.OneOf, schema.AllOf) {
		nullableArraysToAnyOf(sub)
	}

	nullableArraysToAnyOf(schema.Items)
	nullableArraysToAnyOf(schema.AdditionalProperties)

	if len(schema.Types) != 2 || schema.Types[0] != "null" || schema.Types[1] != "array" {
		return
	}

	array := &jsonschema.Schema{
		Type:     "array",
		Items:    schema.Items,
		MinItems: schema.MinItems,
		MaxItems: schema.MaxItems,
	}

	schema.Types = nil
	schema.Items = nil
	schema.MinItems = nil
	schema.MaxItems = nil
	schema.AnyOf = []*jsonschema.Schema{{Type: "null"}, array}
}

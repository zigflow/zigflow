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
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AssertionType is the name of a supported structural assertion. Each value
// validates the shape or type of a workflow output value rather than its exact
// contents, which lets examples that produce variable data (UUIDs, timestamps,
// generated IDs) still be tested.
type AssertionType string

const (
	// AssertExists passes as long as the value is present, whatever its type.
	AssertExists AssertionType = "exists"
	// AssertString requires a string value.
	AssertString AssertionType = "string"
	// AssertNonEmptyString requires a string value with a length greater than
	// zero.
	AssertNonEmptyString AssertionType = "non-empty-string"
	// AssertNumber requires a numeric value. JSON numbers decode to float64, so
	// that is the canonical type, but native integer types are also accepted.
	AssertNumber AssertionType = "number"
	// AssertBoolean requires a boolean value.
	AssertBoolean AssertionType = "boolean"
	// AssertObject requires a JSON object (map with string keys).
	AssertObject AssertionType = "object"
	// AssertArray requires a JSON array.
	AssertArray AssertionType = "array"
	// AssertTimestamp requires a string parseable as an RFC 3339 / ISO 8601
	// timestamp.
	AssertTimestamp AssertionType = "timestamp"
	// AssertUUID requires a string parseable as a UUID.
	AssertUUID AssertionType = "uuid"
)

// timestampLayouts are the layouts accepted by the timestamp assertion. They
// cover the forms produced by Zigflow's runtime expressions (now,
// timestamp_iso8601) as well as common date-only and space-separated forms.
var timestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// typeCheckers maps each assertion type to the predicate that validates it.
var typeCheckers = map[AssertionType]func(any) bool{
	AssertExists:         func(any) bool { return true },
	AssertString:         func(v any) bool { _, ok := v.(string); return ok },
	AssertNonEmptyString: func(v any) bool { s, ok := v.(string); return ok && s != "" },
	AssertNumber:         isNumber,
	AssertBoolean:        func(v any) bool { _, ok := v.(bool); return ok },
	AssertObject:         func(v any) bool { _, ok := v.(map[string]any); return ok },
	AssertArray:          func(v any) bool { _, ok := v.([]any); return ok },
	AssertTimestamp:      isTimestamp,
	AssertUUID:           isUUID,
}

func isNumber(v any) bool {
	switch v.(type) {
	case float64, float32, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isTimestamp(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for _, layout := range timestampLayouts {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

func isUUID(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// CheckStructure validates actual against a structural assertion spec. The spec
// mirrors the shape of the expected output: a node is either a type assertion
// (a map containing a "type" key) or a nested object whose keys are recursed
// into. Assertions are partial; keys present in actual but absent from the spec
// are ignored. It returns a joined error describing every mismatch, or nil when
// actual satisfies the spec.
func CheckStructure(spec, actual any) error {
	var errs []error
	checkNode("$", spec, actual, &errs)
	return errors.Join(errs...)
}

func checkNode(path string, spec, actual any, errs *[]error) {
	node, ok := spec.(map[string]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("%s: assertion must be an object, got %T", path, spec))
		return
	}

	// A node carrying a "type" key is a leaf type assertion.
	if raw, hasType := node["type"]; hasType {
		typeName, ok := raw.(string)
		if !ok {
			*errs = append(*errs, fmt.Errorf("%s: assertion type must be a string, got %T", path, raw))
			return
		}

		check, known := typeCheckers[AssertionType(typeName)]
		if !known {
			*errs = append(*errs, fmt.Errorf("%s: unknown assertion type %q", path, typeName))
			return
		}

		if !check(actual) {
			*errs = append(*errs, fmt.Errorf("%s: expected %s, got %T (%v)", path, typeName, actual, actual))
		}
		return
	}

	// Otherwise the node describes a nested object to recurse into.
	actualMap, ok := actual.(map[string]any)
	if !ok {
		*errs = append(*errs, fmt.Errorf("%s: expected an object to match nested assertions, got %T", path, actual))
		return
	}

	for key, childSpec := range node {
		childActual, present := actualMap[key]
		if !present {
			*errs = append(*errs, fmt.Errorf("%s: missing key %q", path, key))
			continue
		}
		checkNode(path+"."+key, childSpec, childActual, errs)
	}
}

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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sampleUUID   = "f47ac10b-58cc-0372-8567-0e02b2c3d479"
	sampleString = "hello"
	keyID        = "id"
	keyKey       = "key"
	keyValue     = "value"
)

// typeSpec is a convenience for building a single type assertion node.
func typeSpec(name AssertionType) map[string]any {
	return map[string]any{"type": string(name)}
}

func TestCheckStructureTypes(t *testing.T) {
	tests := []struct {
		name    string
		assert  AssertionType
		value   any
		wantErr bool
	}{
		// exists
		{"exists with value", AssertExists, "anything", false},
		{"exists with zero value", AssertExists, float64(0), false},
		{"exists with nil", AssertExists, nil, false},

		// string
		{"string match", AssertString, sampleString, false},
		{"string empty still a string", AssertString, "", false},
		{"string mismatch", AssertString, float64(1), true},

		// non-empty-string
		{"non-empty-string match", AssertNonEmptyString, sampleString, false},
		{"non-empty-string empty", AssertNonEmptyString, "", true},
		{"non-empty-string wrong type", AssertNonEmptyString, true, true},

		// number
		{"number float64", AssertNumber, float64(2345), false},
		{"number int", AssertNumber, 42, false},
		{"number string", AssertNumber, "2345", true},

		// boolean
		{"boolean true", AssertBoolean, true, false},
		{"boolean wrong type", AssertBoolean, "true", true},

		// object
		{"object match", AssertObject, map[string]any{"a": 1}, false},
		{"object wrong type", AssertObject, []any{1, 2}, true},

		// array
		{"array match", AssertArray, []any{1, 2, 3}, false},
		{"array wrong type", AssertArray, map[string]any{}, true},

		// timestamp
		{"timestamp rfc3339", AssertTimestamp, "2026-06-14T17:00:00Z", false},
		{"timestamp date only", AssertTimestamp, "2026-06-14", false},
		{"timestamp space separated", AssertTimestamp, "2026-06-14 17:00:00", false},
		{"timestamp invalid", AssertTimestamp, "not a date", true},
		{"timestamp wrong type", AssertTimestamp, float64(123), true},

		// uuid
		{"uuid match", AssertUUID, sampleUUID, false},
		{"uuid invalid", AssertUUID, "not-a-uuid", true},
		{"uuid wrong type", AssertUUID, float64(1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckStructure(typeSpec(tt.assert), tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckStructureUnknownType(t *testing.T) {
	err := CheckStructure(map[string]any{"type": "definitely-not-a-type"}, "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown assertion type")
}

func TestCheckStructureNested(t *testing.T) {
	spec := map[string]any{
		helloDataKey: map[string]any{
			keyID:       typeSpec(AssertUUID),
			"createdAt": typeSpec(AssertTimestamp),
			"requestId": typeSpec(AssertNonEmptyString),
			"count":     typeSpec(AssertNumber),
		},
	}

	actual := map[string]any{
		helloDataKey: map[string]any{
			keyID:       sampleUUID,
			"createdAt": "2026-06-14T17:00:00Z",
			"requestId": "req-123",
			"count":     float64(7),
			// Extra keys not named in the spec must be ignored.
			"extra": "ignored",
		},
	}

	assert.NoError(t, CheckStructure(spec, actual))
}

func TestCheckStructurePartialIgnoresUnlistedKeys(t *testing.T) {
	// Only "message" is asserted; the variable "id" is left unchecked.
	spec := map[string]any{helloMessageKey: typeSpec(AssertNonEmptyString)}
	actual := map[string]any{
		helloMessageKey: helloMessage,
		keyID:           "anything-goes-here",
	}

	assert.NoError(t, CheckStructure(spec, actual))
}

func TestCheckStructureMissingKey(t *testing.T) {
	spec := map[string]any{
		helloDataKey: map[string]any{
			keyID: typeSpec(AssertUUID),
		},
	}
	actual := map[string]any{
		helloDataKey: map[string]any{},
	}

	err := CheckStructure(spec, actual)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing key")
}

func TestCheckStructureNestedExpectsObject(t *testing.T) {
	// The spec recurses into "data" but the actual value is not an object.
	spec := map[string]any{
		helloDataKey: map[string]any{
			keyID: typeSpec(AssertUUID),
		},
	}
	actual := map[string]any{
		helloDataKey: "not-an-object",
	}

	err := CheckStructure(spec, actual)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected an object")
}

func TestCheckStructureReportsAllMismatches(t *testing.T) {
	spec := map[string]any{
		"a": typeSpec(AssertString),
		"b": typeSpec(AssertNumber),
	}
	actual := map[string]any{
		"a": float64(1), // wrong
		"b": "two",      // wrong
	}

	err := CheckStructure(spec, actual)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "$.a")
	assert.Contains(t, err.Error(), "$.b")
}

func TestCheckStructureMatchesForLoopShape(t *testing.T) {
	// Mirrors the assertion used by the for-loop example test.yaml against the
	// real output shape that example produces.
	spec := map[string]any{
		"forTaskMap": map[string]any{
			"key1": map[string]any{
				keyKey:   typeSpec(AssertNonEmptyString),
				keyValue: typeSpec(AssertString),
			},
			"key2": map[string]any{keyValue: typeSpec(AssertNumber)},
			"key3": map[string]any{keyValue: typeSpec(AssertBoolean)},
		},
		"forTaskArray":          typeSpec(AssertArray),
		"forTaskNumber":         typeSpec(AssertArray),
		"forTaskStateCarryOver": typeSpec(AssertArray),
	}

	actual := map[string]any{
		"forTaskMap": map[string]any{
			"key1": map[string]any{keyKey: "hello: key1", keyValue: sampleString},
			"key2": map[string]any{keyKey: "hello: key2", keyValue: float64(37)},
			"key3": map[string]any{keyKey: "hello: key3", keyValue: true},
		},
		"forTaskArray":          []any{map[string]any{"userId": float64(3)}},
		"forTaskNumber":         []any{map[string]any{"number": float64(0)}},
		"forTaskStateCarryOver": []any{map[string]any{"pageNumber": float64(1)}},
	}

	assert.NoError(t, CheckStructure(spec, actual))
}

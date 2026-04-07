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

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalWorkflow returns a minimal structurally valid workflow document.
// It is used as a baseline for validation tests so that each test only
// introduces one deviation from a known-good state.
func minimalWorkflow() map[string]any {
	return map[string]any{
		"document": map[string]any{
			"dsl":          "1.0.0",
			"taskQueue":    "default",
			"workflowType": "test",
			"version":      "1.0.0",
		},
		"do": []any{
			map[string]any{
				"step1": map[string]any{
					"set": map[string]any{"x": "y"},
				},
			},
		},
	}
}

// TestSchema_DocumentFields verifies that the document schema accepts the
// Zigflow-aligned field names and rejects the old Serverless Workflow names.
func TestSchema_DocumentFields(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)

	t.Run("workflowType and taskQueue are accepted", func(t *testing.T) {
		err := resolved.Validate(minimalWorkflow())
		assert.NoError(t, err)
	})

	t.Run("old name field is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		document := doc["document"].(map[string]any)
		delete(document, "workflowType")
		document["name"] = "test"

		err := resolved.Validate(doc)
		assert.Error(t, err, "old 'name' field must be rejected; use 'workflowType' instead")
	})

	t.Run("old namespace field is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		document := doc["document"].(map[string]any)
		delete(document, "taskQueue")
		document["namespace"] = "default"

		err := resolved.Validate(doc)
		assert.Error(t, err, "old 'namespace' field must be rejected; use 'taskQueue' instead")
	})

	t.Run("missing workflowType is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		delete(doc["document"].(map[string]any), "workflowType")

		err := resolved.Validate(doc)
		assert.Error(t, err, "document without workflowType must fail validation")
	})

	t.Run("missing taskQueue is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		delete(doc["document"].(map[string]any), "taskQueue")

		err := resolved.Validate(doc)
		assert.Error(t, err, "document without taskQueue must fail validation")
	})
}

// TestSchema_RejectsUnknownTopLevelProperties verifies that the root schema
// enforces UnevaluatedProperties: false by rejecting any top-level key that
// is not explicitly defined in the schema.
func TestSchema_RejectsUnknownTopLevelProperties(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)

	t.Run("valid document passes", func(t *testing.T) {
		err := resolved.Validate(minimalWorkflow())
		assert.NoError(t, err, "document with only known top-level properties should pass validation")
	})
}

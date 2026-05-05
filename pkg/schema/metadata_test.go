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

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isOpenSchema returns true when s represents an open (additionalProperties:
// true) schema. trueSchema() returns &jsonschema.Schema{} (Not == nil).
// falseSchema() returns &jsonschema.Schema{Not: &jsonschema.Schema{}} (Not != nil).
func isOpenSchema(s *jsonschema.Schema) bool {
	return s != nil && s.Not == nil
}

// resolvedTestSchema builds and resolves the schema once for use in
// validation tests.
func resolvedTestSchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)
	resolved, err := s.Resolve(nil)
	require.NoError(t, err)
	return resolved
}

// withDocumentMetadata returns a copy of minimalWorkflow() with the given
// map merged into document.metadata.
func withDocumentMetadata(meta map[string]any) map[string]any {
	doc := minimalWorkflow()
	document := doc["document"].(map[string]any)
	document["metadata"] = meta
	return doc
}

// withTaskMetadata returns a copy of minimalWorkflow() where the single
// task body has the given map as its metadata.
func withTaskMetadata(meta map[string]any) map[string]any {
	doc := minimalWorkflow()
	tasks := doc["do"].([]any)
	taskItem := tasks[0].(map[string]any)
	for taskName, rawBody := range taskItem {
		body := rawBody.(map[string]any)
		body["metadata"] = meta
		taskItem[taskName] = body
	}
	return doc
}

// --- definition-presence and shape tests -------------------------------------

// TestMetadataDefinitionsPresent verifies that all three metadata-related
// definitions are registered in buildDefinitions().
func TestMetadataDefinitionsPresent(t *testing.T) {
	defs := buildDefinitions()

	for _, key := range []string{defCommonMetadata, defDocumentMetadata, defTaskMetadata} {
		assert.Contains(t, defs, key, "buildDefinitions() must contain %q", key)
	}
}

// TestCommonMetadataDefinitionShape verifies that commonMetadataDefinition
// declares activityOptions with all its expected sub-properties, and that the
// definition is intentionally open.
func TestCommonMetadataDefinitionShape(t *testing.T) {
	def := commonMetadataDefinition

	assert.Equal(t, typeObject, def.Type)
	assert.True(t, isOpenSchema(def.AdditionalProperties),
		"commonMetadata must be open (additionalProperties: true)")

	require.Contains(t, def.Properties, propActivityOptions)
	actOpts := def.Properties[propActivityOptions]

	// All expected activity-options sub-keys must be present.
	for _, key := range []string{
		propDisableEager,
		propHeartbeatTimeout,
		propRetryPolicy,
		"scheduleToCloseTimeout",
		"scheduleToStartTimeout",
		propStartToCloseTimeout,
		propSummary,
	} {
		assert.Contains(t, actOpts.Properties, key,
			"activityOptions must have property %q", key)
	}

	assert.Equal(t, typeBoolean, actOpts.Properties[propDisableEager].Type)
	assert.Equal(t, SchemaRef("duration"), actOpts.Properties[propHeartbeatTimeout].Ref)
	assert.Equal(t, SchemaRef("duration"), actOpts.Properties["scheduleToCloseTimeout"].Ref)
	assert.Equal(t, SchemaRef("duration"), actOpts.Properties["scheduleToStartTimeout"].Ref)
	assert.Equal(t, SchemaRef("duration"), actOpts.Properties[propStartToCloseTimeout].Ref)
	assert.Equal(t, typeString, actOpts.Properties[propSummary].Type)
}

// TestRetryPolicyDefinitionShape verifies that retryPolicy inside
// activityOptions declares its expected properties and is intentionally
// closed (additionalProperties: false) so that unknown retry keys are
// rejected.
func TestRetryPolicyDefinitionShape(t *testing.T) {
	actOpts := commonMetadataDefinition.Properties[propActivityOptions]
	require.Contains(t, actOpts.Properties, propRetryPolicy)
	rp := actOpts.Properties[propRetryPolicy]

	assert.Equal(t, typeObject, rp.Type)
	assert.False(t, isOpenSchema(rp.AdditionalProperties),
		"retryPolicy must be closed (additionalProperties: false)")

	for _, key := range []string{
		"backoffCoefficient",
		"initialInterval",
		propMaximumAttempts,
		"maximumInterval",
		"nonRetryableErrorTypes",
	} {
		assert.Contains(t, rp.Properties, key, "retryPolicy must have property %q", key)
	}

	assert.Equal(t, "number", rp.Properties["backoffCoefficient"].Type)
	assert.Equal(t, typeInteger, rp.Properties[propMaximumAttempts].Type)
	assert.Equal(t, SchemaRef("duration"), rp.Properties["initialInterval"].Ref)
	assert.Equal(t, SchemaRef("duration"), rp.Properties["maximumInterval"].Ref)

	nonRetryable := rp.Properties["nonRetryableErrorTypes"]
	assert.Equal(t, typeArray, nonRetryable.Type)
	require.NotNil(t, nonRetryable.Items)
	assert.Equal(t, typeString, nonRetryable.Items.Type)
}

// TestDocumentMetadataDefinitionShape verifies that documentMetadataDefinition
// declares all expected Zigflow schedule/document metadata keys and is
// intentionally open.
func TestDocumentMetadataDefinitionShape(t *testing.T) {
	def := documentMetadataDefinition

	assert.Equal(t, typeObject, def.Type)
	assert.True(t, isOpenSchema(def.AdditionalProperties),
		"documentMetadata must be open (additionalProperties: true)")

	for _, key := range []string{
		propCanMaxHistory,
		propScheduleWorkflowName,
		propScheduleID,
		propScheduleInput,
	} {
		assert.Contains(t, def.Properties, key,
			"documentMetadata must have property %q", key)
	}

	assert.Equal(t, typeInteger, def.Properties[propCanMaxHistory].Type)
	assert.Equal(t, typeString, def.Properties[propScheduleWorkflowName].Type)
	assert.Equal(t, typeString, def.Properties[propScheduleID].Type)
	assert.Equal(t, typeArray, def.Properties[propScheduleInput].Type)
}

// TestTaskMetadataDefinitionShape verifies that taskMetadataDefinition
// declares __zigflow_id and heartbeat, and is intentionally open.
func TestTaskMetadataDefinitionShape(t *testing.T) {
	def := taskMetadataDefinition

	assert.Equal(t, typeObject, def.Type)
	assert.True(t, isOpenSchema(def.AdditionalProperties),
		"taskMetadata must be open (additionalProperties: true)")

	require.Contains(t, def.Properties, propZigflowID)
	assert.Equal(t, typeString, def.Properties[propZigflowID].Type)

	require.Contains(t, def.Properties, propHeartbeat)
	assert.Equal(t, SchemaRef("duration"), def.Properties[propHeartbeat].Ref)
}

// TestTaskBaseMetadataAllOf verifies that taskBase wires metadata through
// both commonMetadata and taskMetadata.
func TestTaskBaseMetadataAllOf(t *testing.T) {
	meta := taskBaseDefinition.Properties[propMetadata]
	require.NotNil(t, meta, "taskBase must have a metadata property")

	refs := schemaRefs(meta.AllOf)
	assert.Contains(t, refs, SchemaRef(defCommonMetadata))
	assert.Contains(t, refs, SchemaRef("taskMetadata"))
}

// TestDocumentMetadataAllOf verifies that the document.metadata property in
// the root schema wires through both commonMetadata and documentMetadata.
func TestDocumentMetadataAllOf(t *testing.T) {
	meta := schemaProperties[propDocument].Properties[propMetadata]
	require.NotNil(t, meta, "document must have a metadata property")

	refs := schemaRefs(meta.AllOf)
	assert.Contains(t, refs, SchemaRef(defCommonMetadata))
	assert.Contains(t, refs, SchemaRef(defDocumentMetadata))
}

// --- validation tests ---------------------------------------------------------

// TestDocumentMetadata_Validation exercises the full schema validator against
// various document.metadata objects.
func TestDocumentMetadata_Validation(t *testing.T) {
	resolved := resolvedTestSchema(t)

	tests := []struct {
		name        string
		meta        map[string]any
		expectError bool
	}{
		// ---- valid cases ----
		{
			name: "scheduleWorkflowName string is accepted",
			meta: map[string]any{"scheduleWorkflowName": "my-workflow"},
		},
		{
			name: "scheduleId string is accepted",
			meta: map[string]any{"scheduleId": "my-schedule-id"},
		},
		{
			name: "scheduleInput array is accepted",
			meta: map[string]any{"scheduleInput": []any{map[string]any{"key": "val"}}},
		},
		{
			name: "canMaxHistoryLength integer is accepted",
			meta: map[string]any{propCanMaxHistory: 100},
		},
		{
			name: "unknown metadata key is accepted (open schema)",
			meta: map[string]any{"arbitrary-user-key": "anything"},
		},
		{
			name: "activityOptions with valid startToCloseTimeout is accepted",
			meta: map[string]any{
				propActivityOptions: map[string]any{
					"startToCloseTimeout": map[string]any{propSeconds: 30},
				},
			},
		},
		{
			name: "activityOptions with valid retryPolicy is accepted",
			meta: map[string]any{
				propActivityOptions: map[string]any{
					propRetryPolicy: map[string]any{"maximumAttempts": 3},
				},
			},
		},

		// ---- invalid cases ----
		{
			name:        "scheduleWorkflowName as integer is rejected",
			meta:        map[string]any{"scheduleWorkflowName": 42},
			expectError: true,
		},
		{
			name:        "canMaxHistoryLength as string is rejected",
			meta:        map[string]any{propCanMaxHistory: "not-an-int"},
			expectError: true,
		},
		{
			name:        "activityOptions.disableEagerExecution as string is rejected",
			meta:        map[string]any{propActivityOptions: map[string]any{propDisableEager: "yes"}},
			expectError: true,
		},
		{
			name: "unknown key in retryPolicy is rejected (closed schema)",
			meta: map[string]any{
				propActivityOptions: map[string]any{
					propRetryPolicy: map[string]any{"unknownRetryKey": "val"},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := withDocumentMetadata(tc.meta)
			err := resolved.Validate(doc)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTaskMetadata_Validation exercises the full schema validator against
// various task metadata objects embedded in a set task.
func TestTaskMetadata_Validation(t *testing.T) {
	resolved := resolvedTestSchema(t)

	tests := []struct {
		name        string
		meta        map[string]any
		expectError bool
	}{
		// ---- valid cases ----
		{
			name: "__zigflow_id string is accepted",
			meta: map[string]any{propZigflowID: "abc-123"},
		},
		{
			name: "heartbeat duration object is accepted",
			meta: map[string]any{propHeartbeat: map[string]any{propSeconds: 30}},
		},
		{
			name: "unknown task metadata key is accepted (open schema)",
			meta: map[string]any{"arbitrary-task-key": "anything"},
		},
		{
			name: "activityOptions with valid heartbeatTimeout is accepted",
			meta: map[string]any{
				propActivityOptions: map[string]any{
					propHeartbeatTimeout: map[string]any{propMinutes: 1},
				},
			},
		},

		// ---- invalid cases ----
		{
			name:        "__zigflow_id as integer is rejected",
			meta:        map[string]any{propZigflowID: 99},
			expectError: true,
		},
		{
			name:        "heartbeat as plain string is rejected",
			meta:        map[string]any{propHeartbeat: "30s"},
			expectError: true,
		},
		{
			name: "unknown key in retryPolicy is rejected (closed schema)",
			meta: map[string]any{
				propActivityOptions: map[string]any{
					propRetryPolicy: map[string]any{"bogusKey": true},
				},
			},
			expectError: true,
		},
		{
			name:        "activityOptions.disableEagerExecution as string is rejected",
			meta:        map[string]any{propActivityOptions: map[string]any{propDisableEager: "true"}},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := withTaskMetadata(tc.meta)
			err := resolved.Validate(doc)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

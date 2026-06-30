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

package models

import (
	"encoding/json"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow/extensions"
)

// TestWaitExtTask_ImplementsTaskInterface confirms WaitExtTask satisfies the
// SDK's model.Task contract so the SDK can construct and return it from its
// task registry.
func TestWaitExtTask_ImplementsTaskInterface(t *testing.T) {
	var _ model.Task = (*WaitExtTask)(nil)

	w := &WaitExtTask{}
	assert.Same(t, &w.TaskBase, w.GetBase(), "GetBase must return a pointer to the embedded TaskBase")
}

// TestWaitExtTask_RegisteredWithSDK confirms that init() has registered the
// extension task-type key with the SDK so the SDK can construct a
// *WaitExtTask when it encounters the renamed key.
func TestWaitExtTask_RegisteredWithSDK(t *testing.T) {
	key := extensions.ZigflowExtKeyPrefix + keyWait
	ctor, ok := model.GetTaskConstructor(key)
	require.True(t, ok, "%s must be registered with the SDK task registry", key)

	task := ctor()
	_, isWaitExt := task.(*WaitExtTask)
	assert.True(t, isWaitExt, "registered constructor must return a *WaitExtTask")
}

// TestWaitExtBody_UnmarshalUntilForm exercises the absolute-time form,
// covering both a literal RFC 3339 timestamp and a runtime expression.
func TestWaitExtBody_UnmarshalUntilForm(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "literal RFC 3339",
			json: `{"__zigflow_ext_wait":{"until":"2026-12-31T23:59:59Z"}}`,
			want: "2026-12-31T23:59:59Z",
		},
		{
			name: "runtime expression",
			json: `{"__zigflow_ext_wait":{"until":"${ $data.deadline }"}}`,
			want: "${ $data.deadline }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w WaitExtTask
			require.NoError(t, json.Unmarshal([]byte(tt.json), &w))
			require.NotNil(t, w.Wait)
			assert.Equal(t, tt.want, w.Wait.Until)
		})
	}
}

// TestWaitExtBody_UnmarshalDurationForm exercises the duration form,
// covering literal integers, integer-valued floats (JSON default), and
// runtime expressions as values.
func TestWaitExtBody_UnmarshalDurationForm(t *testing.T) {
	const blob = `{"__zigflow_ext_wait":{"hours":1,"seconds":"${ $data.cooldownSeconds }"}}`

	var w WaitExtTask
	require.NoError(t, json.Unmarshal([]byte(blob), &w))
	require.NotNil(t, w.Wait)

	// JSON numbers unmarshal to float64 when the target is `any`.
	assert.Equal(t, float64(1), w.Wait.Hours)
	assert.Equal(t, "${ $data.cooldownSeconds }", w.Wait.Seconds)
	assert.Empty(t, w.Wait.Until, "Until must remain empty when the body uses the duration form")
}

// TestWaitExtTask_TaskBaseFieldsArePreserved confirms that the embedded
// TaskBase fields (if, metadata, etc.) survive JSON round-trips through
// the extension type.
func TestWaitExtTask_TaskBaseFieldsArePreserved(t *testing.T) {
	const blob = `{
		"__zigflow_ext_wait": {"until": "2026-12-31T23:59:59Z"},
		"if": "${ $data.shouldWait }",
		"metadata": {"reason": "demo"}
	}`

	var w WaitExtTask
	require.NoError(t, json.Unmarshal([]byte(blob), &w))

	require.NotNil(t, w.If, "if field must be populated on TaskBase")
	assert.Equal(t, "${ $data.shouldWait }", w.If.String())
	assert.Equal(t, "demo", w.Metadata["reason"])
}

// TestWaitExtension_ClaimsUntilForm verifies the extension claims any
// wait body that carries an `until` field, regardless of other contents.
func TestWaitExtension_ClaimsUntilForm(t *testing.T) {
	body := map[string]any{keyUntil: "2026-12-31T23:59:59Z"}
	assert.True(t, waitExtension{}.Claims(body))
}

// TestWaitExtension_ClaimsExpressionDuration verifies the extension claims
// a duration body that contains at least one string-valued numeric field.
func TestWaitExtension_ClaimsExpressionDuration(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
	}{
		{"expression seconds", map[string]any{keySeconds: "${ $data.x }"}},
		{"expression minutes", map[string]any{keyMinutes: "${ .delay }"}},
		{"mixed literal and expression", map[string]any{keyHours: 1, keySeconds: "${ $data.x }"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, waitExtension{}.Claims(tt.body))
		})
	}
}

// TestWaitExtension_DoesNotClaimVanillaForms verifies the extension leaves
// the literal-numeric duration form alone so the SDK constructs its
// native *model.WaitTask for it.
func TestWaitExtension_DoesNotClaimVanillaForms(t *testing.T) {
	tests := []struct {
		name string
		body any
	}{
		{"integer seconds only", map[string]any{keySeconds: 5}},
		{"integer-valued float seconds", map[string]any{keySeconds: float64(5)}},
		{"all integer fields", map[string]any{keyDays: 1, keyHours: 2, keyMinutes: 3, keySeconds: 4, keyMilliseconds: 5}},
		{"non-map body (ISO 8601 string)", "PT5S"},
		{"nil body", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, waitExtension{}.Claims(tt.body))
		})
	}
}

// TestWaitExtension_TaskType verifies the extension reports the expected
// Open Workflow Specification task type.
func TestWaitExtension_TaskType(t *testing.T) {
	assert.Equal(t, "wait", waitExtension{}.TaskType())
}

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
)

// waitUntilTimestamp is a literal RFC 3339 timestamp reused across the
// until-form tests below.
const waitUntilTimestamp = "2026-12-31T23:59:59Z"

// waitWorkflow returns a minimal workflow whose only task is a wait task
// with the given wait body. Tests use this to exercise the wait task
// schema branches in isolation.
func waitWorkflow(waitBody map[string]any) map[string]any {
	doc := minimalWorkflow()
	doc[propDo] = []any{
		map[string]any{
			"step1": map[string]any{
				propWait: waitBody,
			},
		},
	}
	return doc
}

// TestSchema_WaitTask_DurationForm verifies that the existing literal-numeric
// duration form still validates after the wait schema rewrite.
func TestSchema_WaitTask_DurationForm(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("integer seconds is accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{propSeconds: 5})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("multiple integer duration fields are accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{
			propDays:         1,
			propHours:        2,
			propMinutes:      3,
			propSeconds:      4,
			propMilliseconds: 5,
		})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("empty duration object is rejected", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{})
		assert.Error(t, resolved.Validate(doc), "wait with empty body must fail validation")
	})
}

// TestSchema_WaitTask_DurationWithExpressions verifies that the duration form
// accepts runtime expressions in any numeric field.
func TestSchema_WaitTask_DurationWithExpressions(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("expression seconds is accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{propSeconds: "${ $data.cooldownSeconds }"})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("expression minutes is accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{propMinutes: "${ .delay }"})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("mixed integer and expression fields are accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{
			propHours:   1,
			propSeconds: "${ $data.extraSeconds }",
		})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("non-expression string is rejected", func(t *testing.T) {
		// A bare string that is not a ${...} expression must not be accepted
		// as a numeric duration value.
		doc := waitWorkflow(map[string]any{propSeconds: "5"})
		assert.Error(t, resolved.Validate(doc), "non-expression string seconds must fail validation")
	})
}

// TestSchema_WaitTask_UntilForm verifies that the absolute-time until form
// accepts both a literal RFC 3339 timestamp and a runtime expression.
func TestSchema_WaitTask_UntilForm(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("literal RFC 3339 until is accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{propUntil: waitUntilTimestamp})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("expression until is accepted", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{propUntil: "${ $data.deadline }"})
		assert.NoError(t, resolved.Validate(doc))
	})

	t.Run("non-RFC 3339 string until is rejected", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{propUntil: "tomorrow"})
		assert.Error(t, resolved.Validate(doc), "non-RFC 3339, non-expression until must fail validation")
	})
}

// TestSchema_WaitTask_RejectsMixedAndUnknown verifies the OneOf boundary:
// mixing until with duration fields, or adding an unknown field, is rejected.
func TestSchema_WaitTask_RejectsMixedAndUnknown(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("until combined with seconds is rejected", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{
			propUntil:   waitUntilTimestamp,
			propSeconds: 5,
		})
		assert.Error(t, resolved.Validate(doc), "until + seconds must fail validation")
	})

	t.Run("unknown key inside wait is rejected", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{
			propSeconds: 5,
			"unknown":   "value",
		})
		assert.Error(t, resolved.Validate(doc), "unknown key inside wait must fail validation")
	})

	t.Run("unknown key alongside until is rejected", func(t *testing.T) {
		doc := waitWorkflow(map[string]any{
			propUntil: waitUntilTimestamp,
			"unknown": "value",
		})
		assert.Error(t, resolved.Validate(doc), "unknown key alongside until must fail validation")
	})
}

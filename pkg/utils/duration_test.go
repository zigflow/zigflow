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

package utils_test

import (
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
)

// TestToDuration covers the SDK-pointer-specific behaviour of the wrapper.
// Summation across duration fields is covered by TestDurationFromMap.
func TestToDuration(t *testing.T) {
	t.Run("nil pointer returns zero", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), utils.ToDuration(nil))
	})

	t.Run("expression-form duration returns zero", func(t *testing.T) {
		// AsInline returns nil for the ISO 8601 expression form, which
		// Zigflow does not support; the wrapper must surface zero rather
		// than panicking.
		d := &model.Duration{Value: model.DurationExpression{Expression: "PT5S"}}
		assert.Equal(t, time.Duration(0), utils.ToDuration(d))
	})

	t.Run("inline duration is summed via DurationFromMap", func(t *testing.T) {
		d := &model.Duration{Value: model.DurationInline{
			Days:         4,
			Hours:        6,
			Minutes:      43,
			Seconds:      32,
			Milliseconds: 472,
		}}
		want := (time.Hour * 24 * 4) + (time.Hour * 6) + (time.Minute * 43) + (time.Second * 32) + (time.Millisecond * 472)
		assert.Equal(t, want, utils.ToDuration(d))
	})
}

func TestDurationFromMap(t *testing.T) {
	t.Run("empty map produces zero duration", func(t *testing.T) {
		d, err := utils.DurationFromMap(map[string]any{})
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
	})

	t.Run("integer seconds", func(t *testing.T) {
		d, err := utils.DurationFromMap(map[string]any{"seconds": 5})
		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, d)
	})

	t.Run("integer-valued float", func(t *testing.T) {
		// JSON unmarshalling produces float64 for numeric values by default;
		// integer-valued floats must be accepted.
		d, err := utils.DurationFromMap(map[string]any{"seconds": float64(5)})
		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, d)
	})

	t.Run("int32 and int64", func(t *testing.T) {
		d, err := utils.DurationFromMap(map[string]any{
			"minutes": int32(2),
			"seconds": int64(30),
		})
		assert.NoError(t, err)
		assert.Equal(t, 2*time.Minute+30*time.Second, d)
	})

	t.Run("all five units sum correctly", func(t *testing.T) {
		d, err := utils.DurationFromMap(map[string]any{
			"days":         1,
			"hours":        2,
			"minutes":      3,
			"seconds":      4,
			"milliseconds": 5,
		})
		assert.NoError(t, err)
		want := 24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second + 5*time.Millisecond
		assert.Equal(t, want, d)
	})

	t.Run("unknown keys are ignored", func(t *testing.T) {
		d, err := utils.DurationFromMap(map[string]any{
			"seconds": 1,
			"weeks":   99, // unknown unit; the schema layer rejects these earlier.
		})
		assert.NoError(t, err)
		assert.Equal(t, time.Second, d)
	})

	t.Run("fractional float is rejected", func(t *testing.T) {
		_, err := utils.DurationFromMap(map[string]any{"seconds": 1.5})
		assert.ErrorContains(t, err, "seconds")
		assert.ErrorContains(t, err, "integer")
	})

	t.Run("string value is rejected", func(t *testing.T) {
		// After expression evaluation a numeric field must resolve to a
		// number. A literal string (or an unresolved expression like
		// "${ ... }") must fail with a clear error.
		_, err := utils.DurationFromMap(map[string]any{"seconds": "5"})
		assert.ErrorContains(t, err, "seconds")
		assert.ErrorContains(t, err, "string")
	})

	t.Run("bool value is rejected", func(t *testing.T) {
		_, err := utils.DurationFromMap(map[string]any{"seconds": true})
		assert.ErrorContains(t, err, "seconds")
		assert.ErrorContains(t, err, "unsupported type")
	})
}

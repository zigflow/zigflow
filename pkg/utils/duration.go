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
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog/log"
)

// ExecuteEvery executes the given function on the duration until the context has stopped.
// The returned cancel function stops the background ticker; callers must call it when done.
func ExecuteEvery(ctx context.Context, duration time.Duration, fn func(context.Context)) (cctx context.Context, cancel func()) {
	doneCh := make(chan struct{})
	var once sync.Once
	cctx = ctx
	cancel = func() { once.Do(func() { close(doneCh) }) }

	go func() {
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		l := log.With().Ctx(ctx).Dur("duration", duration).Logger()

		for {
			select {
			case <-ticker.C:
				l.Debug().Msg("Triggering background function")
				fn(ctx)
			case <-doneCh:
				l.Debug().Msg("Stopping background function")
				return
			case <-ctx.Done():
				l.Debug().Msg("Stopping background function")
				return
			}
		}
	}()

	return cctx, cancel
}

// ToDuration converts an Open Workflow Specification duration into a time.Duration
func ToDuration(v *model.Duration) time.Duration {
	if v == nil {
		return 0
	}
	inline := v.AsInline()
	if inline == nil {
		return 0
	}
	// DurationInline fields are int32 and durationFieldToInt has an int32
	// case, so this call cannot produce an error. The map carries the SDK
	// types verbatim; if DurationFromMap ever adds stricter validation,
	// update the fields below to match.
	d, _ := DurationFromMap(map[string]any{
		"days":         inline.Days,
		"hours":        inline.Hours,
		"minutes":      inline.Minutes,
		"seconds":      inline.Seconds,
		"milliseconds": inline.Milliseconds,
	})
	return d
}

// DurationFromMap builds a time.Duration from a map of duration fields.
// Each known field is summed using its unit.
// Numeric values must be int, int32, int64 or an integer-valued float64;
// anything else is rejected with an error.
// This strictness is deliberate: no string-to-number coercion.
// Keys other than the five duration fields are ignored.
func DurationFromMap(m map[string]any) (time.Duration, error) {
	units := []struct {
		key  string
		unit time.Duration
	}{
		{"days", 24 * time.Hour},
		{"hours", time.Hour},
		{"minutes", time.Minute},
		{"seconds", time.Second},
		{"milliseconds", time.Millisecond},
	}

	var total time.Duration
	for _, u := range units {
		v, ok := m[u.key]
		if !ok {
			continue
		}
		n, err := durationFieldToInt(u.key, v)
		if err != nil {
			return 0, err
		}
		total += time.Duration(n) * u.unit
	}
	return total, nil
}

// durationFieldToInt converts a resolved duration field value to int64,
// rejecting non-integer numeric values and any non-numeric types.
func durationFieldToInt(field string, v any) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case float64:
		if math.Trunc(x) != x {
			return 0, fmt.Errorf("duration field %s must be an integer, got %v", field, x)
		}
		return int64(x), nil
	case string:
		return 0, fmt.Errorf("duration field %s must resolve to a number, got string %q", field, x)
	default:
		return 0, fmt.Errorf("duration field %s has unsupported type %T", field, v)
	}
}

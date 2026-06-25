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
	"time"
)

// Regression test for the gojq "invalid type: time.Time" panic: Temporal
// metadata carries time.Time / time.Duration values, which gojq cannot handle.
// jsonNormalize must convert them to JSON-native types so runtime expressions
// can reference them (e.g. $data.workflow.workflow_start_time).
func TestWorkflowMetadataTimeFieldsAreJQUsable(t *testing.T) {
	ts := time.Date(2026, 6, 25, 20, 56, 48, 0, time.UTC)
	state := NewState()
	state.AddData(map[string]any{
		"workflow": jsonNormalize(map[string]any{
			"workflow_start_time":        ts,
			"workflow_execution_timeout": 5 * time.Minute,
		}),
	})

	// Plain selection must yield the RFC3339 string, not a Go time.Time.
	got, err := EvaluateString("${ $data.workflow.workflow_start_time }", nil, state)
	if err != nil {
		t.Fatalf("selecting workflow_start_time: %v", err)
	}
	if got != "2026-06-25T20:56:48Z" {
		t.Fatalf("workflow_start_time = %#v, want \"2026-06-25T20:56:48Z\"", got)
	}

	// A string operation is the real-world failure mode; before the fix this
	// panicked gojq. It must now succeed.
	str, err := EvaluateString("${ $data.workflow.workflow_start_time | tostring }", nil, state)
	if err != nil {
		t.Fatalf("tostring on workflow_start_time: %v", err)
	}
	if str != "2026-06-25T20:56:48Z" {
		t.Fatalf("tostring(workflow_start_time) = %#v", str)
	}

	// time.Duration must also be usable (normalised to a number), not panic.
	if _, err := EvaluateString("${ $data.workflow.workflow_execution_timeout }", nil, state); err != nil {
		t.Fatalf("selecting workflow_execution_timeout: %v", err)
	}
}

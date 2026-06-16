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

func TestExpressionReferencesActivityState(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want bool
	}{
		{name: "plain activity attempt", expr: "${ $data.activity.attempt }", want: true},
		{name: "activity root", expr: "${ $data.activity }", want: true},
		{name: "bracket index", expr: `${ $data["activity"].attempt }`, want: true},
		{name: "activity nested in string interpolation", expr: `${ "attempt-\($data.activity.attempt)" }`, want: true},
		{name: "activity inside arithmetic", expr: "${ $data.activity.attempt + 1 }", want: true},
		{name: "activity guarded by if", expr: "${ if $data.activity.attempt > 1 then 1 else 0 end }", want: true},
		{name: "env reference", expr: "${ $env.GRPC_INPUT }", want: false},
		{name: "input reference", expr: "${ $input.amount }", want: false},
		{name: "ordinary data reference", expr: "${ $data.idempotencyKey }", want: false},
		{name: "data field that merely starts with activity", expr: "${ $data.activityLog }", want: false},
		{name: "unparseable expression fails safe", expr: "${ $data.activity.( }", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExpressionReferencesActivityState(tc.expr))
		})
	}
}

// keyAmount, exprInputAmount and exprActivityAttempt are shared across the
// resolution tests; defining them once keeps the table readable and avoids
// repeating the same literals.
const (
	keyAmount           = "amount"
	keyAttempt          = "attempt"
	exprInputAmount     = "${ $input.amount }"
	exprActivityAttempt = "${ $data.activity.attempt }"
)

func TestResolveActivityInputResolvesWorkflowSideExpressions(t *testing.T) {
	state := NewState()
	state.Env["GRPC_INPUT"] = "resolved-input"
	state.Input = map[string]any{keyAmount: float64(100)}
	state.Data["idempotencyKey"] = "key-123"

	input := map[string]any{
		"input":          "${ $env.GRPC_INPUT }",
		keyAmount:        exprInputAmount,
		"idempotencyKey": "${ $data.idempotencyKey }",
		keyAttempt:       exprActivityAttempt,
		"static":         "unchanged",
	}

	got, err := ResolveActivityInput(input, state)
	require.NoError(t, err)

	resolved := got.(map[string]any)
	assert.Equal(t, "resolved-input", resolved["input"], "env expression resolved before scheduling")
	assert.Equal(t, float64(100), resolved[keyAmount], "input expression resolved before scheduling")
	assert.Equal(t, "key-123", resolved["idempotencyKey"], "ordinary data expression resolved before scheduling")
	assert.Equal(t, "unchanged", resolved["static"], "non-expression value untouched")

	// The activity-runtime expression must be preserved verbatim, not resolved
	// to null against the workflow-side state where $data.activity is absent.
	assert.Equal(t, exprActivityAttempt, resolved[keyAttempt],
		"activity-runtime expression preserved for activity-side evaluation")
}

func TestResolveActivityInputPreservesNestedActivityExpressions(t *testing.T) {
	state := NewState()
	state.Input = map[string]any{keyAmount: float64(50)}

	input := map[string]any{
		"body": map[string]any{
			keyAmount:  exprInputAmount,
			keyAttempt: exprActivityAttempt,
			"nested": []any{
				exprActivityAttempt,
				exprInputAmount,
			},
		},
	}

	got, err := ResolveActivityInput(input, state)
	require.NoError(t, err)

	body := got.(map[string]any)["body"].(map[string]any)
	assert.Equal(t, float64(50), body[keyAmount])
	assert.Equal(t, exprActivityAttempt, body[keyAttempt])

	nested := body["nested"].([]any)
	assert.Equal(t, exprActivityAttempt, nested[0], "preserved inside slice")
	assert.Equal(t, float64(50), nested[1], "resolved inside slice")
}

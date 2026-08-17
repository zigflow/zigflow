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

package tasks

import (
	"encoding/json"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
)

const (
	// exprActivityAttempt is the activity-runtime expression that must survive
	// workflow-side resolution and be evaluated inside the activity.
	exprActivityAttempt = "${ $data.activity.attempt }"
	// keyAttempt is the body field carrying the activity attempt counter.
	keyAttempt = "attempt"
)

// TestResolveActivityWithGRPCResolvesEnvBeforeScheduling covers the gRPC case
// from issue #462: ${ $env.GRPC_INPUT } in the activity `with` payload must be
// resolved to its concrete value before the activity is scheduled, so the
// Temporal activity input carries the resolved value rather than the raw
// expression.
func TestResolveActivityWithGRPCResolvesEnvBeforeScheduling(t *testing.T) {
	task := newTestGRPCTask()
	task.With.Method = "Command1"
	task.With.Arguments = map[string]any{
		"input": "${ $env.GRPC_INPUT }",
	}

	b, err := NewCallGRPCTaskBuilder(nil, task, "grpc", nil, testEvents, nil)
	require.NoError(t, err)

	state := utils.NewState()
	state.Env["GRPC_INPUT"] = "resolved-grpc-input"

	resolved, err := b.resolveActivityWith(state)
	require.NoError(t, err)

	assert.Equal(t, "resolved-grpc-input", resolved.With.Arguments["input"],
		"env expression resolved before scheduling")
	// Static fields are carried through untouched.
	assert.Equal(t, "localhost", resolved.With.Service.Host)
	assert.Equal(t, "Command1", resolved.With.Method)

	// The shared original task definition must not be mutated.
	assert.Equal(t, "${ $env.GRPC_INPUT }", task.With.Arguments["input"],
		"original task definition is not mutated")
}

// TestResolveActivityWithHTTPResolvesAndPreserves covers the HTTP case: ordinary
// workflow-side expressions in the body are resolved before scheduling, while
// the activity-runtime expression $data.activity.attempt is preserved for the
// activity to evaluate against activity-enriched state.
func TestResolveActivityWithHTTPResolvesAndPreserves(t *testing.T) {
	const bodyKeyAmount = "amount"

	body, err := json.Marshal(map[string]any{
		bodyKeyAmount:    "${ $input.amount }",
		"idempotencyKey": "${ $data.idempotencyKey }",
		keyAttempt:       exprActivityAttempt,
	})
	require.NoError(t, err)

	task := newTestHTTPTask()
	task.With.Method = "POST"
	task.With.Endpoint = model.NewEndpoint("http://server:3000/withdraw")
	// #nosec G101 -- DSL expression, not a hardcoded credential; resolved at runtime from env.
	task.With.Headers = map[string]string{"X-Token": "${ $env.token }"}
	task.With.Body = body

	b, err := NewCallHTTPTaskBuilder(nil, task, "withdraw", nil, testEvents, nil)
	require.NoError(t, err)

	state := utils.NewState()
	state.Env["token"] = "abc-123"
	state.Input = map[string]any{bodyKeyAmount: float64(100)}
	state.Data["idempotencyKey"] = "key-123"

	resolved, err := b.resolveActivityWith(state)
	require.NoError(t, err)

	assert.Equal(t, "abc-123", resolved.With.Headers["X-Token"], "header env expression resolved")

	var gotBody map[string]any
	require.NoError(t, json.Unmarshal(resolved.With.Body, &gotBody))

	assert.Equal(t, float64(100), gotBody[bodyKeyAmount], "input expression resolved before scheduling")
	assert.Equal(t, "key-123", gotBody["idempotencyKey"], "ordinary data expression resolved before scheduling")
	assert.Equal(t, exprActivityAttempt, gotBody[keyAttempt],
		"activity-runtime expression preserved for activity-side evaluation")

	// No unresolved workflow-side expression should remain in the scheduled input.
	raw, err := json.Marshal(resolved.With)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "$env.token")
	assert.NotContains(t, string(raw), "$input.amount")
	assert.NotContains(t, string(raw), "$data.idempotencyKey")

	// The shared original task definition must not be mutated.
	var originalBody map[string]any
	require.NoError(t, json.Unmarshal(task.With.Body, &originalBody))
	assert.Equal(t, "${ $input.amount }", originalBody[bodyKeyAmount], "original task definition is not mutated")
}

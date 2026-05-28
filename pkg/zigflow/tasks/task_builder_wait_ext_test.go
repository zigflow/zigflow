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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// runWaitExtBuilder is a small helper that wires the builder into a test
// workflow environment with a given start time and state, executes it, and
// returns the resulting environment for assertions about env.Now() and any
// workflow error.
func runWaitExtBuilder(t *testing.T, body *models.WaitExtBody, start time.Time, state *utils.State) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	builder, err := NewWaitExtTaskBuilder(nil, &models.WaitExtTask{Wait: body}, "wait-ext", nil, testEvents, nil)
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetStartTime(start)

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "wait-ext"})

	env.ExecuteWorkflow("wait-ext")
	return env
}

func TestWaitExtBuilder_LiteralUntilFuture(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	until := start.Add(30 * time.Minute)

	env := runWaitExtBuilder(t, &models.WaitExtBody{Until: until.Format(time.RFC3339)}, start, utils.NewState())

	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, env.Now().UTC().Equal(until), "workflow clock must advance to the until timestamp")
}

func TestWaitExtBuilder_LiteralUntilPast(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	past := start.Add(-1 * time.Hour)

	env := runWaitExtBuilder(t, &models.WaitExtBody{Until: past.Format(time.RFC3339)}, start, utils.NewState())

	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, env.Now().UTC().Equal(start), "past until must be a no-op; workflow clock must not advance")
}

func TestWaitExtBuilder_UntilFromData(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	until := start.Add(2 * time.Hour)

	state := utils.NewState()
	state.AddData(map[string]any{"deadline": until.Format(time.RFC3339)})

	env := runWaitExtBuilder(t, &models.WaitExtBody{Until: "${ $data.deadline }"}, start, state)

	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, env.Now().UTC().Equal(until), "workflow clock must advance to the resolved until timestamp")
}

func TestWaitExtBuilder_ExpressionSeconds(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	state := utils.NewState()
	state.AddData(map[string]any{"cooldownSeconds": 90})

	env := runWaitExtBuilder(t, &models.WaitExtBody{Seconds: "${ $data.cooldownSeconds }"}, start, state)

	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, env.Now().UTC().Equal(start.Add(90*time.Second)),
		"workflow clock must advance by the resolved seconds value")
}

func TestWaitExtBuilder_ExpressionMixedDurationFields(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	state := utils.NewState()
	state.AddData(map[string]any{"extraSeconds": 30})

	env := runWaitExtBuilder(t, &models.WaitExtBody{
		Hours:   1,
		Seconds: "${ $data.extraSeconds }",
	}, start, state)

	assert.NoError(t, env.GetWorkflowError())
	assert.True(t, env.Now().UTC().Equal(start.Add(1*time.Hour+30*time.Second)),
		"workflow clock must advance by the sum of literal hours and resolved seconds")
}

func TestWaitExtBuilder_InvalidRFC3339Errors(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	env := runWaitExtBuilder(t, &models.WaitExtBody{Until: "tomorrow"}, start, utils.NewState())

	err := env.GetWorkflowError()
	require.Error(t, err, "invalid RFC 3339 must surface as a workflow error")
	assert.Contains(t, err.Error(), "RFC 3339")
}

func TestWaitExtBuilder_StringValuedDurationErrors(t *testing.T) {
	start := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	// A runtime expression that resolves to a string rather than a number
	// must fail loudly rather than be silently coerced.
	state := utils.NewState()
	state.AddData(map[string]any{"cooldown": "ninety"})

	env := runWaitExtBuilder(t, &models.WaitExtBody{Seconds: "${ $data.cooldown }"}, start, state)

	err := env.GetWorkflowError()
	require.Error(t, err, "non-numeric resolved duration must surface as a workflow error")
	assert.Contains(t, err.Error(), "seconds")
}

// TestWaitExtBuilder_ValidateRejectsBadRFC3339 covers literal `until`
// values that the JSON-schema regex accepts (case-insensitive separator
// and zone, no range checks on components) but that time.Parse(RFC3339)
// rejects. Validate is the single source of truth for "is this a real
// RFC 3339 timestamp" and must catch them at validate time.
func TestWaitExtBuilder_ValidateRejectsBadRFC3339(t *testing.T) {
	tests := []struct {
		name  string
		until string
	}{
		{"lowercase t separator", "2026-12-31t23:59:59Z"},
		{"lowercase z zone", "2026-12-31T23:59:59z"},
		{"month 13", "2026-13-01T00:00:00Z"},
		{"day 32", "2026-01-32T00:00:00Z"},
		{"hour 25", "2026-12-31T25:00:00Z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder, err := NewWaitExtTaskBuilder(nil,
				&models.WaitExtTask{Wait: &models.WaitExtBody{Until: tc.until}},
				"wait-ext",
				nil,
				testEvents,
				nil)
			require.NoError(t, err)

			err = builder.Validate()
			require.Error(t, err, "Validate must reject malformed RFC 3339 timestamp")
			assert.Contains(t, err.Error(), "RFC 3339")
		})
	}
}

// TestWaitExtBuilder_ValidateAcceptsValidUntil ensures Validate does not
// reject the legitimate `until` shapes: real RFC 3339 timestamps (with or
// without offset), runtime expressions (resolved later), and the empty
// string used by the duration form.
func TestWaitExtBuilder_ValidateAcceptsValidUntil(t *testing.T) {
	cases := []struct {
		name  string
		until string
	}{
		{"valid RFC 3339 with Z", "2026-12-31T23:59:59Z"},
		{"valid RFC 3339 with offset", "2026-12-31T23:59:59+02:00"},
		{"runtime expression", "${ $data.deadline }"},
		{"empty (duration form)", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			builder, err := NewWaitExtTaskBuilder(nil,
				&models.WaitExtTask{Wait: &models.WaitExtBody{Until: tc.until}},
				"wait-ext",
				nil,
				testEvents,
				nil)
			require.NoError(t, err)
			assert.NoError(t, builder.Validate())
		})
	}
}

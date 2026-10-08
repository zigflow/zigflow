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

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	activityAttemptExpr = "${ $data.activity.attempt }"
	// Text a resolved value can hold; evaluating it again would read the environment.
	injectedExpr = "${ $env.API_TOKEN }"
	inputIDExpr  = "${ $input.id }"
	keyArgs      = "args"
	keyBody      = "body"
)

// withAttempt adds the activity metadata an attempt sees, reduced to its number.
func withAttempt(state *State, attempt int) *State {
	return state.AddData(map[string]any{"activity": map[string]any{"attempt": attempt}})
}

func TestReferencesActivityState(t *testing.T) {
	tests := map[string]bool{
		activityAttemptExpr:                        true,
		"${ $data.activity }":                      true,
		`${ $data["activity"].attempt }`:           true,
		`${ $data."activity".attempt }`:            true,
		"${ $data | .activity.attempt }":           true,
		"${ $data as $d | $d.activity.attempt }":   true,
		"${ [$input.id, $data.activity.attempt] }": true,
		"${ $data[$input.key] }":                   true,
		"${ $data[] }":                             true,
		"${ $data }":                               true,
		"${ $data | }":                             true, // does not parse: left for the activity, which reports it
		"${ $data.user.activity }":                 false,
		"${ $data.activity_log }":                  false,
		`${ $data["user"] }`:                       false,
		"${ $input.activity.attempt }":             false,
		injectedExpr:                               false,
		"${ .activity }":                           false,
	}
	for expr, want := range tests {
		assert.Equal(t, want, referencesActivityState(expr), expr)
	}
}

func TestResolveActivityInputsDefersOnlyActivityState(t *testing.T) {
	state := NewState()
	state.Env = map[string]any{testKeyHOST: "example.com"}
	state.Input = map[string]any{"id": 2}

	env := map[string]string{"ID": inputIDExpr, "NONE": "${ $input.missing }"}
	value := map[string]any{
		"endpoint": `${ "https://" + $env.HOST }`,
		"headers":  map[string]any{"X-Attempt": activityAttemptExpr, "a/b~c": activityAttemptExpr},
		keyBody:    []any{"plain", inputIDExpr, map[string]any{"n": activityAttemptExpr}},
		testKeyEnv: env,
		keyArgs:    []string{inputIDExpr},
	}

	resolved, inputs, err := ResolveActivityInputs(value, state, true)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"endpoint": "https://example.com",
		"headers":  map[string]any{"X-Attempt": activityAttemptExpr, "a/b~c": activityAttemptExpr},
		keyBody:    []any{"plain", 2, map[string]any{"n": activityAttemptExpr}},
		testKeyEnv: map[string]string{"ID": "2", "NONE": ""},
		keyArgs:    []any{2},
	}, resolved)
	assert.Equal(t, []string{"/body/2/n", "/headers/X-Attempt", "/headers/a~1b~0c"}, inputs.Deferred)

	assert.Equal(t, `${ "https://" + $env.HOST }`, value["endpoint"], "the input must not be modified")
	assert.Equal(t, inputIDExpr, env["ID"], "a map[string]string input must not be modified")
}

func TestResolveActivityInputsWithoutDeferringResolvesEverything(t *testing.T) {
	resolved, inputs, err := ResolveActivityInputs(map[string]any{keyBody: activityAttemptExpr}, NewState(), false)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{keyBody: nil}, resolved)
	assert.Empty(t, inputs.Deferred)
}

func TestResolveActivityInputsReturnsEvaluationErrors(t *testing.T) {
	_, _, err := ResolveActivityInputs(map[string]any{"bad": `${ error("boom") }`}, NewState(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

// Temporal decodes activity arguments as JSON, where every number is a float64:
// a result that would not survive that is left for the activity, as before.
func TestResolveActivityInputsDefersValuesJSONWouldChange(t *testing.T) {
	value := map[string]any{
		"safe":     "${ 9007199254740992 }",
		"float":    "${ 1.5 }",
		"big":      "${ 9007199254740993 }",
		"huge":     "${ 1000000000000000000000000000000 }",
		"nested":   "${ {a: [1, 9007199254740993]} }",
		"nan":      "${ nan }",
		"infinite": "${ infinite }",
	}

	resolved, inputs, err := ResolveActivityInputs(value, NewState(), false)
	require.NoError(t, err)

	assert.Equal(t, []string{"/big", "/huge", "/infinite", "/nan", "/nested"}, inputs.Deferred)
	got := resolved.(map[string]any)
	assert.Equal(t, 9007199254740992, got["safe"])
	assert.InDelta(t, 1.5, got["float"], 0)
	assert.Equal(t, "${ 9007199254740993 }", got["big"])
}

// Resolution runs in workflow code, so with several failing expressions the
// reported error must be the same on every run, including a replay.
func TestResolveActivityInputsReportsTheSameErrorEveryTime(t *testing.T) {
	failing := map[string]any{
		"b": `${ error("B") }`,
		"a": `${ error("A") }`,
		"c": map[string]string{"y": `${ error("C") }`, "x": `${ error("D") }`},
	}
	for range 50 {
		_, _, err := ResolveActivityInputs(failing, NewState(), true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error: A")
	}

	env := map[string]string{"b": `${ error("B") }`, "a": `${ error("A") }`, "c": `${ error("C") }`}
	for range 50 {
		_, _, err := ResolveActivityInputs(env, NewState(), true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error: A")
	}
}

// A resolved value that looks like an expression is data. It reaches the
// activity as is and is never evaluated again; that is what the
// ActivityInputs marker on the activity's state guarantees.
func TestActivityInputsAreEvaluatedOnce(t *testing.T) {
	state := NewState()
	state.Env = map[string]any{"API_TOKEN": "s3cr3t"}
	state.Input = map[string]any{"comment": injectedExpr}

	resolved, inputs, err := ResolveActivityInputs(map[string]any{keyBody: "${ $input.comment }"}, state, true)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{keyBody: injectedExpr}, resolved)

	activityState := state.Clone()
	activityState.ActivityInputs = inputs
	got, err := EvaluateActivityInput("", resolved, activityState)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{keyBody: injectedExpr}, got)

	// Without the marker the activity would evaluate the resolved value again.
	legacy, err := EvaluateActivityInput("", resolved, state)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{keyBody: "s3cr3t"}, legacy)
}

func TestEvaluateActivityInputEvaluatesOnlyWhatWasDeferredUnderPrefix(t *testing.T) {
	state := withAttempt(NewState(), 3)
	state.ActivityInputs = &ActivityInputs{Deferred: []string{"/environment/A", "/exec/args/1"}}

	got, err := EvaluateActivityInput("/exec", map[string]any{
		keyArgs:    []string{activityAttemptExpr, activityAttemptExpr},
		testKeyEnv: map[string]string{"A": activityAttemptExpr},
	}, state)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		keyArgs:    []any{activityAttemptExpr, 3},
		testKeyEnv: map[string]string{"A": activityAttemptExpr},
	}, got)
}

func TestEvaluateActivityInputWithoutResolvedInputsMatchesTraverseAndEvaluate(t *testing.T) {
	state := withAttempt(NewState(), 3)
	state.Env = map[string]any{testKeyHOST: "example.com"}
	value := func() map[string]any {
		return map[string]any{
			keyArgs:    []string{"${ $env.HOST }", "literal"},
			testKeyEnv: map[string]string{"A": activityAttemptExpr, "B": "${ $input.missing }"},
			"nest":     map[string]any{"list": []any{activityAttemptExpr, 1}},
		}
	}

	got, err := EvaluateActivityInput("/exec", value(), state)
	require.NoError(t, err)
	want, err := TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(value()), nil, state)
	require.NoError(t, err)

	assert.Equal(t, want, got)
}

func TestEvaluateDeferredActivityInputLeavesUnresolvedInputAlone(t *testing.T) {
	args := []string{"${ $env.HOST }"}

	got, err := EvaluateDeferredActivityInput("/arguments", args, NewState())
	require.NoError(t, err)

	assert.Equal(t, args, got, "an activity scheduled with unresolved inputs evaluates them elsewhere, as before")
}

func TestStateCloneKeepsActivityInputs(t *testing.T) {
	state := NewState()
	state.ActivityInputs = &ActivityInputs{Deferred: []string{"/a"}}

	assert.Equal(t, state.ActivityInputs, state.Clone().ActivityInputs)
}

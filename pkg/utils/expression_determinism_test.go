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

const (
	exprNow         = "${ now }"
	exprTimestamp   = "${ timestamp }"
	exprUUID        = "${ uuid }"
	symbolNow       = "now"
	symbolTimestamp = "timestamp"
	symbolUUID      = "uuid"
)

func TestAnalyseExpressionDeterminism_Deterministic(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"state variable", "${ $data.delay }"},
		{"state variable plus literal", "${ $data.delay + 10 }"},
		{"length of state array", "${ length($data.items) }"},
		{"identity", "${ . }"},
		{"context access", "${ $context.user.id }"},
		{"input access", "${ $input.name }"},
		{"map over state", "${ $data.items | map(. * 2) }"},
		{"select over state", "${ $data.items[] | select(.active) }"},
		{"object literal from state", `${ {name: $data.name, age: $data.age} }`},
		{"array literal from state", "${ [$data.a, $data.b] }"},
		{"string interpolation from state", `${ "hello \($data.name)" }`},
		{"if-then-else over state", "${ if $data.x > 0 then $data.a else $data.b end }"},
		{"reduce over state", "${ reduce $data.items[] as $i (0; . + $i) }"},
		{"foreach over state", "${ foreach $data.items[] as $i (0; . + $i; .) }"},
		{"try over state", "${ try $data.x catch $data.y }"},
		{"local def referencing state", "${ def double: . * 2; $data.x | double }"},
		{"label and break", "${ label $out | $data.items[] | if . == 5 then ., break $out else . end }"},
		{"strftime takes input not clock", `${ $data.epoch | strftime("%Y-%m-%d") }`},
		{"plain string literal", `${ "static" }`},
		{"arithmetic on literals", "${ 1 + 2 * 3 }"},
		{"slice on state", "${ $data.items[0:2] }"},
		{"iter on state", "${ $data.items[] }"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism(tc.expr)
			require.NoError(t, err)
			assert.True(t, analysis.Deterministic,
				"expected expression to be deterministic, got issues: %+v", analysis.Issues)
		})
	}
}

func TestAnalyseExpressionDeterminism_NonDeterministic(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantSymbol string
	}{
		{"bare timestamp", exprTimestamp, symbolTimestamp},
		{"bare uuid", exprUUID, symbolUUID},
		{"bare random", "${ random }", "random"},
		{"bare now", exprNow, symbolNow},
		{"bare env", "${ env.HOST }", "env"},
		{"$ENV variable", "${ $ENV.HOST }", "$ENV"},
		{"timestamp nested in arithmetic", "${ $data.delay + timestamp }", symbolTimestamp},
		{"uuid nested in array literal", "${ length([$data.id, uuid]) }", symbolUUID},
		{"timestamp inside object", `${ {created: timestamp} }`, symbolTimestamp},
		{"now inside if branch", "${ if $data.flag then now else 0 end }", symbolNow},
		{"uuid as argument to map", "${ $data.items | map(uuid) }", symbolUUID},
		{"unknown function", "${ totally_made_up }", "totally_made_up"},
		{"timestamp_iso8601 piped", "${ timestamp_iso8601 | length }", "timestamp_iso8601"},
		{"localtime nested", "${ $data.epoch | localtime }", "localtime"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism(tc.expr)
			require.NoError(t, err)
			assert.False(t, analysis.Deterministic,
				"expected expression to be non-deterministic")
			require.NotEmpty(t, analysis.Issues)
			symbols := make([]string, 0, len(analysis.Issues))
			for _, issue := range analysis.Issues {
				symbols = append(symbols, issue.Symbol)
			}
			assert.Contains(t, symbols, tc.wantSymbol,
				"expected issue for %q, got: %+v", tc.wantSymbol, analysis.Issues)
		})
	}
}

func TestAnalyseExpressionDeterminism_StateVariablesAreReplaySafe(t *testing.T) {
	for v := range stateVars {
		t.Run(v, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism("${ " + v + " }")
			require.NoError(t, err)
			assert.True(t, analysis.Deterministic,
				"state variable %s must be replay-safe, got: %+v", v, analysis.Issues)
		})
	}
}

func TestAnalyseExpressionDeterminism_ParseError(t *testing.T) {
	_, err := AnalyseExpressionDeterminism("${ @@@ }")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestAnalyseExpressionDeterminism_UnknownFunctionsFailClosed(t *testing.T) {
	// Custom functions that are not registered as replay-safe must be flagged
	// as non-deterministic. This is the fail-closed behaviour: we'd rather
	// reject a safe expression than silently accept an unsafe one.
	analysis, err := AnalyseExpressionDeterminism("${ my_custom_func($data.x) }")
	require.NoError(t, err)
	assert.False(t, analysis.Deterministic)
	require.NotEmpty(t, analysis.Issues)
	assert.Equal(t, "my_custom_func", analysis.Issues[0].Symbol)
}

func TestAnalyseExpressionDeterminism_LocallyDefinedFuncIsSafe(t *testing.T) {
	// Locally defined functions that only reference state are safe.
	analysis, err := AnalyseExpressionDeterminism("${ def add_one: . + 1; $data.x | add_one }")
	require.NoError(t, err)
	assert.True(t, analysis.Deterministic,
		"locally defined function over state must be deterministic, got: %+v", analysis.Issues)
}

func TestAnalyseExpressionDeterminism_LocallyDefinedFuncBodyIsChecked(t *testing.T) {
	// A user-defined function whose body uses `timestamp` is still
	// non-deterministic, even if the function name itself is local.
	analysis, err := AnalyseExpressionDeterminism("${ def tick: timestamp; tick }")
	require.NoError(t, err)
	assert.False(t, analysis.Deterministic)
	require.NotEmpty(t, analysis.Issues)
	assert.Equal(t, "timestamp", analysis.Issues[0].Symbol)
}

func TestAnalyseExpressionDeterminism_AsPatternVarIsSafe(t *testing.T) {
	// Variables introduced by `as $x` patterns are deterministic — they bind
	// to the value of the left-hand side, which is itself walked.
	analysis, err := AnalyseExpressionDeterminism("${ $data.items[] as $i | $i + 1 }")
	require.NoError(t, err)
	assert.True(t, analysis.Deterministic, "got issues: %+v", analysis.Issues)
}

func TestAnalyseExpressionDeterminism_ImportsAreRejected(t *testing.T) {
	// Imports load arbitrary modules at evaluation time; treat them as
	// non-deterministic.
	analysis, err := AnalyseExpressionDeterminism(`${ import "./mod" as m; m::foo }`)
	if err != nil {
		// Some gojq builds may reject the import syntax at parse time; either
		// outcome rejects the expression, which is the contract we care about.
		return
	}
	assert.False(t, analysis.Deterministic)
}

func TestIsExpressionDeterministic(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"deterministic", "${ $data.delay }", true},
		{"non-deterministic", exprTimestamp, false},
		{"parse error fails closed", "${ @@@ }", false},
		{"nested non-deterministic", "${ $data.delay + uuid }", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsExpressionDeterministic(tc.expr))
		})
	}
}

func TestIsDeterministicBuiltin(t *testing.T) {
	t.Run("length is deterministic", func(t *testing.T) {
		assert.True(t, IsDeterministicBuiltin("length"))
	})
	t.Run("map is deterministic", func(t *testing.T) {
		assert.True(t, IsDeterministicBuiltin("map"))
	})
	t.Run("now is not on the allow-list", func(t *testing.T) {
		assert.False(t, IsDeterministicBuiltin(symbolNow))
	})
	t.Run("env is not on the allow-list", func(t *testing.T) {
		assert.False(t, IsDeterministicBuiltin("env"))
	})
	t.Run("unknown identifier is not on the allow-list", func(t *testing.T) {
		assert.False(t, IsDeterministicBuiltin("totally_made_up"))
	})
}

func TestAnalyseExpressionDeterminism_BareExpressionWithoutWrapper(t *testing.T) {
	// The analyser should also handle expressions that are not wrapped in
	// ${ ... } — for example when callers have already sanitised.
	analysis, err := AnalyseExpressionDeterminism("$data.x + 1")
	require.NoError(t, err)
	assert.True(t, analysis.Deterministic)

	analysis, err = AnalyseExpressionDeterminism(symbolTimestamp)
	require.NoError(t, err)
	assert.False(t, analysis.Deterministic)
}

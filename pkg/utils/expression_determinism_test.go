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
		{"propagated access", "${ $propagated.correlationId }"},
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

func TestAnalyseExpressionDeterminism_DestructuringAlternatives(t *testing.T) {
	// gojq v0.12.20 replaced Reduce.Pattern and Foreach.Pattern with
	// Patterns []*Pattern to support `?//` destructuring alternatives. Every
	// variable bound by any alternative must be in scope for the body, and
	// non-deterministic calls inside the body must still be detected.
	deterministic := []struct {
		name string
		expr string
	}{
		{"reduce with alternatives", "${ reduce $data.items[] as [$a] ?// {b: $b} (0; . + ($a // 0) + ($b // 0)) }"},
		{"reduce with three alternatives", "${ reduce $data.items[] as [$a, $b] ?// {x: $c} ?// $d (0; . + ($a // $c // $d // 0) + ($b // 0)) }"},
		{"foreach with alternatives", "${ foreach $data.items[] as [$a] ?// {b: $b} (0; . + ($a // 0); [., $b]) }"},
		{"foreach with nested alternatives", "${ foreach $data.items[] as {k: [$a]} ?// [$b] (null; $a // $b; [., $a, $b]) }"},
	}
	for _, tc := range deterministic {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism(tc.expr)
			require.NoError(t, err)
			assert.True(t, analysis.Deterministic,
				"expected expression to be deterministic, got issues: %+v", analysis.Issues)
		})
	}

	nonDeterministic := []struct {
		name       string
		expr       string
		wantSymbol string
	}{
		{"reduce alternatives with now in update", "${ reduce $data.items[] as [$a] ?// {b: $b} (0; . + ($a // $b) + now) }", symbolNow},
		{"reduce alternatives with now in start", "${ reduce $data.items[] as [$a] ?// $b (now; . + ($a // $b)) }", symbolNow},
		{"reduce alternatives with now in source", "${ reduce (now, $data.x) as [$a] ?// $b (0; . + ($a // $b)) }", symbolNow},
		{"foreach alternatives with now in update", "${ foreach $data.items[] as [$a] ?// {b: $b} (0; . + now; [., $a, $b]) }", symbolNow},
		{"foreach alternatives with now in extract", "${ foreach $data.items[] as [$a] ?// $b (0; . + ($a // $b); . + now) }", symbolNow},
		{"foreach alternatives with uuid in start", "${ foreach $data.items[] as [$a] ?// $b (uuid; $a // $b) }", symbolUUID},
	}
	for _, tc := range nonDeterministic {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism(tc.expr)
			require.NoError(t, err)
			assert.False(t, analysis.Deterministic, "expected expression to be non-deterministic")
			symbols := make([]string, 0, len(analysis.Issues))
			for _, issue := range analysis.Issues {
				symbols = append(symbols, issue.Symbol)
			}
			assert.Contains(t, symbols, tc.wantSymbol,
				"expected issue for %q, got: %+v", tc.wantSymbol, analysis.Issues)
			assert.Len(t, analysis.Issues, 1,
				"pattern variables must not be flagged, got: %+v", analysis.Issues)
		})
	}
}

func TestAnalyseExpressionDeterminism_PatternObjectKeys(t *testing.T) {
	// Object patterns can bind variables through the key (`{$a}`,
	// `{$a: [$b]}`) and can compute keys from expressions (`{(expr): $x}`,
	// `{"\(expr)": $x}`). Key bindings must be in scope and key expressions
	// must be analysed for `as`, `reduce` and `foreach`.
	deterministic := []struct {
		name string
		expr string
	}{
		{"as shorthand", "${ $data as {$a} | $a }"},
		{"as key binding with nested pattern", "${ $data as {$a: [$b]} | [$a, $b] }"},
		{"as nested shorthand", "${ $data as [{$a: {$b}}] | [$a, $b] }"},
		{"reduce shorthand with alternatives", "${ reduce $data.items[] as {$a} ?// [$b] (0; . + ($a // $b)) }"},
		{"foreach key binding with alternatives", "${ foreach $data.items[] as {$a: [$b]} ?// $c (0; . + ($a // $c); [., $b]) }"},
		{"as computed key from state", "${ $data as {($data.key): $a} | $a }"},
		{"as interpolated key from state", `${ $data as {"\($data.key)": $a} | $a }`},
		{"as computed key using earlier binding", "${ $data as {$k, ($k): $v} | $v }"},
		{"as computed key using other alternative", `${ $data as {($b // "x"): $a} ?// [$b] | [$a, $b] }`},
		{"reduce computed key from state", "${ reduce $data.items[] as {($data.key): $a} ?// [$a] (0; . + $a) }"},
		{"foreach interpolated key from state", `${ foreach $data.items[] as {"\($data.key)": $a} (0; . + $a; .) }`},
	}
	for _, tc := range deterministic {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism(tc.expr)
			require.NoError(t, err)
			assert.True(t, analysis.Deterministic,
				"expected expression to be deterministic, got issues: %+v", analysis.Issues)
		})
	}

	nonDeterministic := []struct {
		name       string
		expr       string
		wantSymbol string
	}{
		{"as computed key with now", "${ $data as {(now | tostring): $a} | $a }", symbolNow},
		{"as interpolated key with now", `${ $data as {"\(now)": $a} | $a }`, symbolNow},
		{"as nested computed key with now", "${ $data as [{x: {(now | tostring): $a}}] | $a }", symbolNow},
		{"as shorthand alongside computed key with now", "${ $data as {$a, (now | tostring): $b} | [$a, $b] }", symbolNow},
		{"reduce computed key with now in alternative", "${ reduce $data.items[] as [$a] ?// {(now | tostring): $a} (0; . + $a) }", symbolNow},
		{
			"foreach interpolated key with now in alternative",
			`${ foreach $data.items[] as [$b] ?// {"\(now)": $a} (0; . + ($a // $b); .) }`,
			symbolNow,
		},
		// Within a single pattern, a key is evaluated before its own entry is
		// bound, so $v is not in scope for the key expression.
		{"computed key referencing its own binding", "${ $data as {($v): $v} | $v }", "$v"},
	}
	for _, tc := range nonDeterministic {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := AnalyseExpressionDeterminism(tc.expr)
			require.NoError(t, err)
			assert.False(t, analysis.Deterministic, "expected expression to be non-deterministic")
			require.Len(t, analysis.Issues, 1,
				"bound variables must not be flagged, got: %+v", analysis.Issues)
			assert.Equal(t, tc.wantSymbol, analysis.Issues[0].Symbol)
		})
	}
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

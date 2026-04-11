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
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newState(input, output, ctx any, data map[string]any) *State {
	s := NewState()
	s.Input = input
	s.Output = output
	s.Context = ctx
	if data != nil {
		s.AddData(data)
	}
	return s
}

func TestEvaluateString(t *testing.T) {
	tests := []struct {
		Name      string
		Str       string
		Ctx       any
		State     *State
		Expected  any
		ExpectErr bool
	}{
		{
			Name:     "plain string is returned as-is",
			Str:      "hello world",
			State:    NewState(),
			Expected: "hello world",
		},
		{
			Name:     "empty string is returned as-is",
			Str:      "",
			State:    NewState(),
			Expected: "",
		},
		{
			Name:     "non-expression dollar sign is returned as-is",
			Str:      "$not_an_expression",
			State:    NewState(),
			Expected: "$not_an_expression",
		},
		{
			Name:     "simple field access on context",
			Str:      "${ .name }",
			Ctx:      map[string]any{"name": "zigflow"},
			State:    NewState(),
			Expected: "zigflow",
		},
		{
			Name:     "nested field access on context",
			Str:      "${ .user.email }",
			Ctx:      map[string]any{"user": map[string]any{"email": "test@example.com"}},
			State:    NewState(),
			Expected: "test@example.com",
		},
		{
			Name:     "access state input variable",
			Str:      "${ $input.value }",
			State:    newState(map[string]any{"value": 42}, nil, nil, nil),
			Expected: 42,
		},
		{
			Name:     "access state output variable",
			Str:      "${ $output.result }",
			State:    newState(nil, map[string]any{"result": "done"}, nil, nil),
			Expected: "done",
		},
		{
			Name:     "access state context variable",
			Str:      "${ $context.key }",
			State:    newState(nil, nil, map[string]any{"key": "ctx-value"}, nil),
			Expected: "ctx-value",
		},
		{
			Name:     "access state data variable",
			Str:      "${ $data.counter }",
			State:    newState(nil, nil, nil, map[string]any{"counter": 7}),
			Expected: 7,
		},
		{
			Name:     "jq arithmetic expression",
			Str:      "${ 1 + 2 }",
			State:    NewState(),
			Expected: 3,
		},
		{
			Name:     "jq boolean expression returns true",
			Str:      "${ 5 > 3 }",
			State:    NewState(),
			Expected: true,
		},
		{
			Name:     "jq boolean expression returns false",
			Str:      "${ 1 > 3 }",
			State:    NewState(),
			Expected: false,
		},
		{
			Name:     "jq string literal expression",
			Str:      `${ "static" }`,
			State:    NewState(),
			Expected: "static",
		},
		{
			Name:     "expression without spaces inside braces",
			Str:      "${.name}",
			Ctx:      map[string]any{"name": "compact"},
			State:    NewState(),
			Expected: "compact",
		},
		{
			Name:      "invalid jq expression returns error",
			Str:       "${ @@@ }",
			State:     NewState(),
			ExpectErr: true,
		},
		{
			Name:     "expression accessing missing field returns null",
			Str:      "${ .nonexistent }",
			Ctx:      map[string]any{},
			State:    NewState(),
			Expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result, err := EvaluateString(test.Str, test.Ctx, test.State)
			if test.ExpectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.Expected, result)
			}
		})
	}
}

func TestEvaluateStringWithWrapper(t *testing.T) {
	t.Run("wrapper intercepts evaluation", func(t *testing.T) {
		called := false
		wrapper := ExpressionWrapperFunc(func(f func() (any, error)) (any, error) {
			called = true
			return f()
		})

		result, err := EvaluateString("${ .x }", map[string]any{"x": 99}, NewState(), wrapper)
		require.NoError(t, err)
		assert.Equal(t, 99, result)
		assert.True(t, called, "wrapper should have been called")
	})

	t.Run("wrapper can override result", func(t *testing.T) {
		wrapper := ExpressionWrapperFunc(func(_ func() (any, error)) (any, error) {
			return "overridden", nil
		})

		result, err := EvaluateString("${ .x }", map[string]any{"x": 99}, NewState(), wrapper)
		require.NoError(t, err)
		assert.Equal(t, "overridden", result)
	})

	t.Run("wrapper is not called for plain strings", func(t *testing.T) {
		called := false
		wrapper := ExpressionWrapperFunc(func(f func() (any, error)) (any, error) {
			called = true
			return f()
		})

		result, err := EvaluateString("plain text", nil, NewState(), wrapper)
		require.NoError(t, err)
		assert.Equal(t, "plain text", result)
		assert.False(t, called, "wrapper should not be called for non-expressions")
	})

	t.Run("wrapper can return an error", func(t *testing.T) {
		wrapper := ExpressionWrapperFunc(func(_ func() (any, error)) (any, error) {
			return nil, fmt.Errorf("wrapper error")
		})

		_, err := EvaluateString("${ .x }", nil, NewState(), wrapper)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wrapper error")
	})
}

func TestEvaluateStringJqFunctions(t *testing.T) {
	t.Run("uuid returns a valid UUID", func(t *testing.T) {
		result, err := EvaluateString("${ uuid }", nil, NewState())
		require.NoError(t, err)

		uuidStr, ok := result.(string)
		require.True(t, ok, "uuid should return a string")

		uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		assert.Regexp(t, uuidPattern, uuidStr)
	})

	t.Run("timestamp returns an integer unix timestamp", func(t *testing.T) {
		result, err := EvaluateString("${ timestamp }", nil, NewState())
		require.NoError(t, err)

		ts, ok := result.(int)
		require.True(t, ok, "timestamp should return an int")
		assert.Greater(t, ts, 0)
	})

	t.Run("timestamp_iso8601 returns an RFC3339 formatted string", func(t *testing.T) {
		result, err := EvaluateString("${ timestamp_iso8601 }", nil, NewState())
		require.NoError(t, err)

		tsStr, ok := result.(string)
		require.True(t, ok, "timestamp_iso8601 should return a string")

		rfc3339Pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
		assert.Regexp(t, rfc3339Pattern, tsStr)
	})

	t.Run("uuid returns unique values on repeated calls", func(t *testing.T) {
		result1, err := EvaluateString("${ uuid }", nil, NewState())
		require.NoError(t, err)

		result2, err := EvaluateString("${ uuid }", nil, NewState())
		require.NoError(t, err)

		assert.NotEqual(t, result1, result2)
	})
}

func TestEvaluateStringNonDeterministicWithWrapper(t *testing.T) {
	tests := []struct {
		Name string
		Expr string
	}{
		{Name: "uuid with wrapper", Expr: "${ uuid }"},
		{Name: "timestamp with wrapper", Expr: "${ timestamp }"},
		{Name: "timestamp_iso8601 with wrapper", Expr: "${ timestamp_iso8601 }"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			wrapper := ExpressionWrapperFunc(func(f func() (any, error)) (any, error) {
				return f()
			})
			result, err := EvaluateString(test.Expr, nil, NewState(), wrapper)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestLeadingNonDeterministicFunc(t *testing.T) {
	tests := []struct {
		Name     string
		Expr     string
		Expected string
	}{
		// Registered non-deterministic functions
		{Name: "uuid is detected", Expr: "uuid", Expected: "uuid"},
		{Name: "timestamp is detected", Expr: "timestamp", Expected: "timestamp"},
		{Name: "timestamp_iso8601 is detected", Expr: "timestamp_iso8601", Expected: "timestamp_iso8601"},

		// Leading whitespace is trimmed
		{Name: "uuid with leading spaces", Expr: "  uuid", Expected: "uuid"},

		// Only the leading identifier matters; a pipe still makes it non-det
		{Name: "uuid piped to length", Expr: "uuid | length", Expected: "uuid"},

		// Field accesses and variable references are NOT function calls
		{Name: "field access .uuid is not a function", Expr: ".uuid", Expected: ""},
		{Name: "variable $data.uuid is not a function", Expr: "$data.uuid", Expected: ""},
		{Name: "jq path .foo is not a function", Expr: ".foo", Expected: ""},

		// Unknown identifiers that look like functions but aren't registered
		{Name: "unknown identifier returns empty", Expr: "my_func", Expected: ""},
		{Name: "length is not registered", Expr: "length", Expected: ""},

		// Edge cases
		{Name: "empty string returns empty", Expr: "", Expected: ""},
		{Name: "only whitespace returns empty", Expr: "   ", Expected: ""},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got := leadingNonDeterministicFunc(test.Expr)
			assert.Equal(t, test.Expected, got)
		})
	}
}

func TestIsIdentStart(t *testing.T) {
	tests := []struct {
		Name     string
		Char     byte
		Expected bool
	}{
		{Name: "lowercase letter", Char: 'a', Expected: true},
		{Name: "uppercase letter", Char: 'Z', Expected: true},
		{Name: "underscore", Char: '_', Expected: true},
		{Name: "digit is not an ident start", Char: '0', Expected: false},
		{Name: "dot is not an ident start", Char: '.', Expected: false},
		{Name: "dollar is not an ident start", Char: '$', Expected: false},
		{Name: "hyphen is not an ident start", Char: '-', Expected: false},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, isIdentStart(test.Char))
		})
	}
}

func TestIsIdentChar(t *testing.T) {
	tests := []struct {
		Name     string
		Char     byte
		Expected bool
	}{
		{Name: "lowercase letter", Char: 'z', Expected: true},
		{Name: "uppercase letter", Char: 'A', Expected: true},
		{Name: "underscore", Char: '_', Expected: true},
		{Name: "digit", Char: '9', Expected: true},
		{Name: "dot is not an ident char", Char: '.', Expected: false},
		{Name: "dollar is not an ident char", Char: '$', Expected: false},
		{Name: "space is not an ident char", Char: ' ', Expected: false},
		{Name: "pipe is not an ident char", Char: '|', Expected: false},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, isIdentChar(test.Char))
		})
	}
}

func TestTraverseAndEvaluateObj(t *testing.T) {
	tests := []struct {
		Name      string
		Obj       *model.ObjectOrRuntimeExpr
		Ctx       any
		State     *State
		Expected  any
		ExpectErr bool
	}{
		{
			Name:     "nil input returns nil",
			Obj:      nil,
			State:    NewState(),
			Expected: nil,
		},
		{
			Name:     "plain string value",
			Obj:      model.NewObjectOrRuntimeExpr("hello"),
			State:    NewState(),
			Expected: "hello",
		},
		{
			Name:  "map with expression value",
			Obj:   model.NewObjectOrRuntimeExpr(map[string]any{"greeting": "${ .msg }"}),
			Ctx:   map[string]any{"msg": "hi"},
			State: NewState(),
			Expected: map[string]any{
				"greeting": "hi",
			},
		},
		{
			Name: "map with mixed plain and expression values",
			Obj: model.NewObjectOrRuntimeExpr(map[string]any{
				"static":  "constant",
				"dynamic": "${ .value }",
			}),
			Ctx:   map[string]any{"value": "evaluated"},
			State: NewState(),
			Expected: map[string]any{
				"static":  "constant",
				"dynamic": "evaluated",
			},
		},
		{
			Name: "map containing an array of expressions",
			Obj: model.NewObjectOrRuntimeExpr(map[string]any{
				"items": []any{
					"${ .a }",
					"${ .b }",
					"plain",
				},
			}),
			Ctx:   map[string]any{"a": 1, "b": 2},
			State: NewState(),
			Expected: map[string]any{
				"items": []any{1, 2, "plain"},
			},
		},
		{
			Name: "map with numeric value (no evaluation)",
			Obj: model.NewObjectOrRuntimeExpr(map[string]any{
				"count": 42,
			}),
			State: NewState(),
			Expected: map[string]any{
				"count": 42,
			},
		},
		{
			Name: "map with state variable access",
			Obj: model.NewObjectOrRuntimeExpr(map[string]any{
				"inputVal": "${ $input.name }",
			}),
			State: newState(map[string]any{"name": "from-input"}, nil, nil, nil),
			Expected: map[string]any{
				"inputVal": "from-input",
			},
		},
		{
			Name: "map with invalid expression returns error",
			Obj: model.NewObjectOrRuntimeExpr(map[string]any{
				"bad": "${ @@@ }",
			}),
			State:     NewState(),
			ExpectErr: true,
		},
		{
			Name: "map containing array with invalid expression returns error",
			Obj: model.NewObjectOrRuntimeExpr(map[string]any{
				"items": []any{"${ @@@ }"},
			}),
			State:     NewState(),
			ExpectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result, err := TraverseAndEvaluateObj(test.Obj, test.Ctx, test.State)
			if test.ExpectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.Expected, result)
			}
		})
	}
}

func TestTraverseAndEvaluateObjMapStringStringNilCoercedToEmpty(t *testing.T) {
	obj := model.NewObjectOrRuntimeExpr(map[string]any{
		"env": map[string]string{
			"NULLABLE": "${ null }",
			"PLAIN":    "value",
		},
	})

	result, err := TraverseAndEvaluateObj(obj, nil, NewState())
	require.NoError(t, err)
	// A null result must become "" not "<nil>".
	assert.Equal(t, map[string]any{
		"env": map[string]string{
			"NULLABLE": "",
			"PLAIN":    "value",
		},
	}, result)
}

func TestTraverseAndEvaluateObjMapStringString(t *testing.T) {
	state := NewState()
	state.Env["OPENAI_API_KEY"] = "secret-key"

	obj := model.NewObjectOrRuntimeExpr(map[string]any{
		"env": map[string]string{
			"OPENAI_API_KEY": "${ $env.OPENAI_API_KEY }",
		},
	})

	result, err := TraverseAndEvaluateObj(obj, nil, state)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"env": map[string]string{
			"OPENAI_API_KEY": "secret-key",
		},
	}, result)
}

func TestTraverseAndEvaluateObjDoesNotMutateMapStringString(t *testing.T) {
	env := map[string]string{
		"HOST": "${ .host }",
		"PORT": "8080",
	}
	obj := model.NewObjectOrRuntimeExpr(map[string]any{"env": env})

	result, err := TraverseAndEvaluateObj(obj, map[string]any{"host": "localhost"}, NewState())
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"env": map[string]string{
			"HOST": "localhost",
			"PORT": "8080",
		},
	}, result)

	// Original map[string]string must not be mutated.
	// DeepCloneValue does not clone map[string]string, so traverseAndEvaluate
	// must allocate its own copy before writing evaluated values.
	assert.Equal(t, "${ .host }", env["HOST"], "original map[string]string was mutated")
	assert.Equal(t, "8080", env["PORT"])
}

func TestTraverseAndEvaluateObjMapStringStringConcurrent(t *testing.T) {
	// Run many goroutines against the same shared ObjectOrRuntimeExpr containing
	// a map[string]string. The race detector will catch any concurrent writes to
	// the original map if the clone is missing.
	env := map[string]string{
		"HOST": "${ .host }",
		"PORT": "8080",
	}
	obj := model.NewObjectOrRuntimeExpr(map[string]any{"env": env})

	const goroutines = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = TraverseAndEvaluateObj(obj, map[string]any{"host": "localhost"}, NewState())
		}()
	}
	wg.Wait()

	assert.Equal(t, "${ .host }", env["HOST"], "original map[string]string was mutated")
}

func TestTraverseAndEvaluateObjDoesNotMutateOriginal(t *testing.T) {
	obj := model.NewObjectOrRuntimeExpr(map[string]any{
		"greeting": "${ .msg }",
		"nested": map[string]any{
			"items": []any{"${ .a }", "plain"},
		},
	})

	result, err := TraverseAndEvaluateObj(obj, map[string]any{
		"msg": "hi",
		"a":   1,
	}, NewState())
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"greeting": "hi",
		"nested": map[string]any{
			"items": []any{1, "plain"},
		},
	}, result)

	// The original workflow definition object should be unchanged so it can be
	// safely reused by concurrent workflow executions.
	assert.Equal(t, map[string]any{
		"greeting": "${ .msg }",
		"nested": map[string]any{
			"items": []any{"${ .a }", "plain"},
		},
	}, obj.AsStringOrMap())
}

func TestTraverseAndEvaluateObjWithWrapper(t *testing.T) {
	t.Run("wrapper is called for expression values in a map", func(t *testing.T) {
		callCount := 0
		wrapper := ExpressionWrapperFunc(func(f func() (any, error)) (any, error) {
			callCount++
			return f()
		})

		obj := model.NewObjectOrRuntimeExpr(map[string]any{
			"a": "${ .x }",
			"b": "plain",
		})
		result, err := TraverseAndEvaluateObj(obj, map[string]any{"x": 1}, NewState(), wrapper)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"a": 1, "b": "plain"}, result)
		assert.Equal(t, 1, callCount, "wrapper should be called once for the single expression value")
	})

	t.Run("wrapper is called for expressions inside a nested array", func(t *testing.T) {
		callCount := 0
		wrapper := ExpressionWrapperFunc(func(f func() (any, error)) (any, error) {
			callCount++
			return f()
		})

		obj := model.NewObjectOrRuntimeExpr(map[string]any{
			"items": []any{"${ .a }", "${ .b }", "plain"},
		})
		result, err := TraverseAndEvaluateObj(obj, map[string]any{"a": 10, "b": 20}, NewState(), wrapper)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"items": []any{10, 20, "plain"}}, result)
		assert.Equal(t, 2, callCount, "wrapper should be called once per expression in the array")
	})

	t.Run("non-deterministic function works correctly with wrapper", func(t *testing.T) {
		wrapper := ExpressionWrapperFunc(func(f func() (any, error)) (any, error) {
			return f()
		})

		obj := model.NewObjectOrRuntimeExpr(map[string]any{
			"id": "${ uuid }",
		})
		result, err := TraverseAndEvaluateObj(obj, nil, NewState(), wrapper)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)

		uuidVal, ok := resultMap["id"].(string)
		require.True(t, ok)

		uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		assert.Regexp(t, uuidPattern, uuidVal)
	})
}

func TestCheckIfStatement(t *testing.T) {
	tests := []struct {
		Name        string
		Expression  *model.RuntimeExpression
		State       *State
		Expected    bool
		ExpectErr   bool
		ErrContains string
	}{
		{
			Name:       "nil expression returns true",
			Expression: nil,
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "boolean true expression",
			Expression: model.NewExpr("${ true }"),
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "boolean false expression",
			Expression: model.NewExpr("${ false }"),
			State:      NewState(),
			Expected:   false,
		},
		{
			Name:       "comparison that evaluates to true",
			Expression: model.NewExpr("${ 5 > 3 }"),
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "comparison that evaluates to false",
			Expression: model.NewExpr("${ 1 > 3 }"),
			State:      NewState(),
			Expected:   false,
		},
		{
			Name:       "string TRUE returns true",
			Expression: model.NewExpr(`${ "TRUE" }`),
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "string true (lowercase) returns true",
			Expression: model.NewExpr(`${ "true" }`),
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "string True (mixed case) returns true",
			Expression: model.NewExpr(`${ "True" }`),
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "string 1 returns true",
			Expression: model.NewExpr(`${ "1" }`),
			State:      NewState(),
			Expected:   true,
		},
		{
			Name:       "string false returns false",
			Expression: model.NewExpr(`${ "false" }`),
			State:      NewState(),
			Expected:   false,
		},
		{
			Name:       "string 0 returns false",
			Expression: model.NewExpr(`${ "0" }`),
			State:      NewState(),
			Expected:   false,
		},
		{
			Name:       "expression using input state",
			Expression: model.NewExpr("${ $input.active }"),
			State:      newState(map[string]any{"active": true}, nil, nil, nil),
			Expected:   true,
		},
		{
			Name:        "numeric result returns error (unknown type)",
			Expression:  model.NewExpr("${ 42 }"),
			State:       NewState(),
			ExpectErr:   true,
			ErrContains: "If statement response type unknown",
		},
		{
			Name:        "invalid jq expression returns non-retryable error",
			Expression:  model.NewExpr("${ @@@ }"),
			State:       NewState(),
			ExpectErr:   true,
			ErrContains: "Error parsing if statement",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result, err := CheckIfStatement(test.Expression, test.State)
			if test.ExpectErr {
				require.Error(t, err)
				if test.ErrContains != "" {
					assert.Contains(t, err.Error(), test.ErrContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.Expected, result)
			}
		})
	}
}

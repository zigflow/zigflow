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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itchyny/gojq"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
)

const ndeDocsURL = "https://zigflow.dev/docs/concepts/data-and-expressions/#why-must-generated-values-be-in-a-set-task"

type ExpressionWrapperFunc func(func() (any, error)) (any, error)

type jqFunc struct {
	Name    string                         // Becomes the name of the function to use (eg, ${ uuid })
	MinArgs int                            // Minimum number of args
	MaxArgs int                            // Maximum number of args
	Func    func(vars any, args []any) any // The function - receives the variables and arguments
}

// List of functions that are available as a function.
// All registered functions are non-deterministic and will trigger an error
// when used outside of a set task (i.e. without a side-effect wrapper).
var jqFuncs []jqFunc = []jqFunc{
	{
		Name: "uuid",
		Func: func(_ any, _ []any) any {
			return uuid.New().String()
		},
	},
	{
		Name: "timestamp",
		Func: func(_ any, _ []any) any {
			// Convert to int so it can be formatted by strftime
			return int(time.Now().Unix())
		},
	},
	{
		Name: "timestamp_iso8601",
		Func: func(_ any, _ []any) any {
			return time.Now().UTC().Format(time.RFC3339)
		},
	},
}

// The return value could be any value depending upon how it's parsed
func EvaluateString(str string, ctx any, state *State, evaluationWrapper ...ExpressionWrapperFunc) (any, error) {
	// Check if the string is a runtime expression (e.g., ${ .some.path })
	if model.IsStrictExpr(str) {
		// Error if a non-deterministic function is used without a side-effect wrapper.
		// A wrapper is only provided by tasks (e.g. set) that correctly isolate
		// non-deterministic calls inside workflow.SideEffect.
		if len(evaluationWrapper) == 0 {
			expr := model.SanitizeExpr(str)
			if fnName := leadingNonDeterministicFunc(expr); fnName != "" {
				log.Error().
					Str("expression", str).
					Str("function", fnName).
					Str("docs", ndeDocsURL).
					Msg("Non-deterministic function used outside of a set task — this may cause workflow replay failures")
			}
		}

		// Wrapper exists to allow JQ evaluation to be put inside a workflow to make deterministic
		fn := buildEvaluationWrapperFn(evaluationWrapper...)

		return fn(func() (any, error) {
			return evaluateJQExpression(model.SanitizeExpr(str), ctx, state)
		})
	}
	return str, nil
}

func buildEvaluationWrapperFn(evaluationWrapper ...ExpressionWrapperFunc) ExpressionWrapperFunc {
	var wrapperFn ExpressionWrapperFunc = func(f func() (any, error)) (any, error) {
		return f()
	}
	if len(evaluationWrapper) > 0 {
		// If a function is passed in, use that instead
		wrapperFn = evaluationWrapper[0]
	}

	return wrapperFn
}

// leadingNonDeterministicFunc returns the name of the non-deterministic function
// that expr begins with, or an empty string if it does not.
// Only the leading identifier is inspected: `uuid` errors, but `$data.uuid`
// and `.uuid` do not, as those are variable or field references, not calls.
func leadingNonDeterministicFunc(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" || !isIdentStart(expr[0]) {
		return ""
	}
	end := 1
	for end < len(expr) && isIdentChar(expr[end]) {
		end++
	}
	ident := expr[:end]
	for _, fn := range jqFuncs {
		if fn.Name == ident {
			return ident
		}
	}
	return ""
}

func isIdentStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

func TraverseAndEvaluateObj(
	runtimeExpr *model.ObjectOrRuntimeExpr,
	ctx any,
	state *State,
	evaluationWrapper ...ExpressionWrapperFunc,
) (any, error) {
	if runtimeExpr == nil {
		return nil, nil
	}

	// Keep nil when no external wrapper was provided so that EvaluateString can
	// detect the absence of a side-effect wrapper and emit errors accordingly.
	var wrapperFn ExpressionWrapperFunc
	if len(evaluationWrapper) > 0 {
		wrapperFn = evaluationWrapper[0]
	}

	return traverseAndEvaluate(runtimeExpr.AsStringOrMap(), ctx, state, wrapperFn)
}

func traverseAndEvaluate(node, ctx any, state *State, evaluationWrapper ExpressionWrapperFunc) (any, error) {
	switch v := node.(type) {
	case map[string]any:
		// Traverse a object
		for key, value := range v {
			evaluatedValue, err := traverseAndEvaluate(value, ctx, state, evaluationWrapper)
			if err != nil {
				return nil, err
			}
			v[key] = evaluatedValue
		}
		return v, nil
	case []any:
		// Traverse an array
		return traverseSlice(v, ctx, state, evaluationWrapper)
	case []string:
		// Traverse an array
		return traverseSlice(toAnySlice(v), ctx, state, evaluationWrapper)
	case string:
		if evaluationWrapper != nil {
			return EvaluateString(v, ctx, state, evaluationWrapper)
		}
		return EvaluateString(v, ctx, state)
	default:
		// Return as-is
		return v, nil
	}
}

func traverseSlice(v []any, ctx any, state *State, evaluationWrapper ExpressionWrapperFunc) ([]any, error) {
	for i, value := range v {
		evaluatedValue, err := traverseAndEvaluate(value, ctx, state, evaluationWrapper)
		if err != nil {
			return nil, err
		}
		v[i] = evaluatedValue
	}
	return v, nil
}

func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func evaluateJQExpression(expression string, ctx any, state *State) (any, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq expression: %s, error: %w", expression, err)
	}

	// Get the variable names & values in a single pass:
	names, values := getVariableNamesAndValues(state.GetAsMap())

	fns := []gojq.CompilerOption{
		gojq.WithVariables(names),
	}

	for _, j := range jqFuncs {
		fns = append(fns, gojq.WithFunction(j.Name, j.MinArgs, j.MaxArgs, j.Func))
	}

	code, err := gojq.Compile(query, fns...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jq expression: %s, error: %w", expression, err)
	}

	iter := code.Run(ctx, values...)
	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("no result from jq evaluation")
	}
	// If there's an error from the jq engine, report it
	if errVal, isErr := v.(error); isErr {
		return nil, fmt.Errorf("jq evaluation error: %w", errVal)
	}

	return v, nil
}

func CheckIfStatement(ifStatement *model.RuntimeExpression, state *State) (bool, error) {
	if ifStatement == nil {
		return true, nil
	}

	res, err := EvaluateString(ifStatement.String(), nil, state)
	if err != nil {
		// Treat a parsing error as non-retryable
		return false, temporal.NewNonRetryableApplicationError("Error parsing if statement", "If statement error", err)
	}

	// Response can be a boolean, "TRUE" (case-insensitive) or "1"
	switch r := res.(type) {
	case bool:
		return r, nil
	case string:
		return strings.EqualFold(r, "TRUE") || r == "1", nil
	default:
		return false, temporal.NewNonRetryableApplicationError(
			"If statement response type unknown",
			"If statement error",
			fmt.Errorf("response not string or bool"),
		)
	}
}

func getVariableNamesAndValues(vars map[string]any) (names []string, values []any) {
	names = make([]string, 0, len(vars))
	values = make([]any, 0, len(vars))

	for k, v := range vars {
		names = append(names, k)
		values = append(values, v)
	}
	return names, values
}

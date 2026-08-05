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
	swUtil "github.com/open-workflow-specification/sdk-go/v4/impl/utils"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/temporal"
)

const (
	ndeDocsURL = "https://zigflow.dev/docs/concepts/data-and-expressions/#why-must-generated-values-be-in-a-set-task"

	jqFuncUUID         = "uuid"
	jqFuncTimestamp    = "timestamp"
	jqFuncTimestampISO = "timestamp_iso8601"
)

type ExpressionWrapperFunc func(func() (any, error)) (any, error)

// PreserveExprFunc reports whether a strict-form runtime expression should be
// left unevaluated during a traversal. It is used to perform partial
// evaluation: resolve the expressions that can be resolved now and preserve the
// rest verbatim for a later evaluation pass.
type PreserveExprFunc func(expr string) bool

// jqFunc describes a function exposed to Zigflow runtime expressions. The
// Deterministic flag is the source of truth for replay-safety classification;
// AnalyseExpressionDeterminism reads this list when walking expressions.
type jqFunc struct {
	Name          string
	MinArgs       int
	MaxArgs       int
	Deterministic bool
	Func          func(vars any, args []any) any
}

// jqFuncs is the registry of Zigflow-defined jq functions. Anything marked
// Deterministic=false is rejected outside a Set task because its result would
// not survive Temporal workflow replay.
var jqFuncs []jqFunc = []jqFunc{
	{
		Name: jqFuncUUID,
		Func: func(_ any, _ []any) any {
			return uuid.New().String()
		},
	},
	{
		Name: jqFuncTimestamp,
		Func: func(_ any, _ []any) any {
			// Convert to int so it can be formatted by strftime
			return int(time.Now().Unix())
		},
	},
	{
		Name: jqFuncTimestampISO,
		Func: func(_ any, _ []any) any {
			return time.Now().UTC().Format(time.RFC3339)
		},
	},
}

// The return value could be any value depending upon how it's parsed
func EvaluateString(str string, ctx any, state *State, evaluationWrapper ...ExpressionWrapperFunc) (any, error) {
	// Check if the string is a runtime expression (e.g., ${ .some.path })
	if model.IsStrictExpr(str) {
		// Error if a non-deterministic construct is used without a side-effect
		// wrapper. A wrapper is only provided by tasks (e.g. set) that
		// correctly isolate non-deterministic calls inside workflow.SideEffect.
		if len(evaluationWrapper) == 0 {
			if analysis, err := AnalyseExpressionDeterminism(str); err == nil && !analysis.Deterministic {
				for _, issue := range analysis.Issues {
					log.Error().
						Str("expression", str).
						Str("symbol", issue.Symbol).
						Str("reason", issue.Reason).
						Str("docs", ndeDocsURL).
						Msg("Non-deterministic expression used outside of a set task — this may cause workflow replay failures")
				}
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

	// The workflow/task definition objects are shared across workflow executions.
	// traverseAndEvaluate mutates maps/slices in place while replacing runtime
	// expressions with their evaluated values, so traversing the original value
	// will mutate shared workflow definition state. Under load this shows up as
	// `fatal error: concurrent map writes` when many workflows evaluate the same
	// export/output definitions at once.
	//
	// Clone the value first so each evaluation works on an isolated copy.
	return traverseAndEvaluate(swUtil.DeepCloneValue(runtimeExpr.AsStringOrMap()), ctx, state, wrapperFn, nil)
}

// ResolveActivityInput evaluates the strict-form runtime expressions in value
// against workflow state, except those that reference activity-runtime state
// (e.g. $data.activity.*), which are preserved verbatim. It is used to resolve
// an activity's `with` payload in the workflow, before the activity is
// scheduled, so the Temporal activity input records concrete values for normal
// workflow-side expressions while leaving genuinely activity-runtime
// expressions for the activity to evaluate against activity-enriched state.
//
// value is expected to be a freshly decoded structure (map/slice/scalar) that
// is safe to mutate; callers obtain it by JSON round-tripping the typed `with`
// payload.
func ResolveActivityInput(value any, state *State) (any, error) {
	return traverseAndEvaluate(value, nil, state, nil, ExpressionReferencesActivityState)
}

func traverseAndEvaluate(
	node, ctx any,
	state *State,
	evaluationWrapper ExpressionWrapperFunc,
	preserve PreserveExprFunc,
) (any, error) {
	switch v := node.(type) {
	case map[string]any:
		// Traverse a object
		for key, value := range v {
			evaluatedValue, err := traverseAndEvaluate(value, ctx, state, evaluationWrapper, preserve)
			if err != nil {
				return nil, err
			}
			v[key] = evaluatedValue
		}
		return v, nil
	case map[string]string:
		// Traverse map values for string-only maps (for example, run task environment vars).
		// DeepCloneValue does not clone map[string]string, so allocate a fresh map here
		// to avoid mutating the original, which may be shared across workflow executions.
		clone := make(map[string]string, len(v))
		for key, value := range v {
			evaluatedValue, err := traverseAndEvaluate(value, ctx, state, evaluationWrapper, preserve)
			if err != nil {
				return nil, err
			}
			if evaluatedValue == nil {
				clone[key] = ""
			} else {
				clone[key] = fmt.Sprintf("%v", evaluatedValue)
			}
		}
		return clone, nil
	case []any:
		// Traverse an array
		return traverseSlice(v, ctx, state, evaluationWrapper, preserve)
	case []string:
		// Traverse an array
		return traverseSlice(toAnySlice(v), ctx, state, evaluationWrapper, preserve)
	case string:
		// Partial evaluation: leave preserved expressions untouched so a later
		// pass (for example, inside the activity) can evaluate them against a
		// more complete state.
		if preserve != nil && model.IsStrictExpr(v) && preserve(v) {
			return v, nil
		}
		if evaluationWrapper != nil {
			return EvaluateString(v, ctx, state, evaluationWrapper)
		}
		return EvaluateString(v, ctx, state)
	default:
		// Return as-is
		return v, nil
	}
}

func traverseSlice(
	v []any,
	ctx any,
	state *State,
	evaluationWrapper ExpressionWrapperFunc,
	preserve PreserveExprFunc,
) ([]any, error) {
	for i, value := range v {
		evaluatedValue, err := traverseAndEvaluate(value, ctx, state, evaluationWrapper, preserve)
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

// jqCompilerOptions builds the gojq compiler options shared by runtime
// evaluation and validation: the given workflow state variables plus every
// Zigflow-registered function. Keeping this in one place ensures validation
// compiles expressions under exactly the same rules as runtime, so the two
// cannot drift.
func jqCompilerOptions(variableNames []string) []gojq.CompilerOption {
	opts := []gojq.CompilerOption{
		gojq.WithVariables(variableNames),
	}
	for _, j := range jqFuncs {
		opts = append(opts, gojq.WithFunction(j.Name, j.MinArgs, j.MaxArgs, j.Func))
	}
	return opts
}

// CompileExpression parses and compiles expr using the exact same gojq options
// as runtime evaluation (the Zigflow workflow state variables and registered
// functions). It returns an error if the expression cannot parse or compile —
// for example, referencing an unregistered variable or function — which would
// otherwise only surface at execution time. The strict expression wrapper
// (${ ... }) is stripped before parsing.
func CompileExpression(expr string) error {
	if model.IsStrictExpr(expr) {
		expr = model.SanitizeExpr(expr)
	}

	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("failed to parse expression %q: %w", expr, err)
	}

	// Use the canonical state variable names so validation matches runtime,
	// where the same names are always injected via State.GetAsMap.
	names, _ := getVariableNamesAndValues(NewState().GetAsMap())
	if _, err := gojq.Compile(query, jqCompilerOptions(names)...); err != nil {
		return fmt.Errorf("failed to compile expression %q: %w", expr, err)
	}

	return nil
}

func evaluateJQExpression(expression string, ctx any, state *State) (any, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq expression: %s, error: %w", expression, err)
	}

	// Get the variable names & values in a single pass:
	names, values := getVariableNamesAndValues(state.GetAsMap())

	code, err := gojq.Compile(query, jqCompilerOptions(names)...)
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

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
	"maps"
	"math"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/itchyny/gojq"
	swUtil "github.com/open-workflow-specification/sdk-go/v4/impl/utils"
	"github.com/open-workflow-specification/sdk-go/v4/model"
)

// ActivityInputs is set on the state an activity receives when the workflow
// resolved the activity's inputs before scheduling it. Deferred lists, as
// JSON Pointers (RFC 6901), the expressions left for the activity: those that
// may read $data.activity, which only exists once an attempt runs (see
// AddActivityInfo), and those whose result Temporal's JSON encoding would
// change (see survivesJSON). Every input expression is evaluated exactly once.
type ActivityInputs struct {
	Deferred []string `json:"deferred,omitempty"`
}

// ResolveActivityInputs evaluates the runtime expressions in an activity's
// input against the workflow state, before the activity is scheduled. An
// expression whose result would not survive Temporal's JSON encoding is left
// in place and its location recorded for the activity; with
// deferActivityState, so is one that may read $data.activity. value is not
// modified.
func ResolveActivityInputs(value any, state *State, deferActivityState bool) (any, *ActivityInputs, error) {
	inputs := &ActivityInputs{}
	resolved, err := walkInputs(swUtil.DeepCloneValue(value), "", func(path, s string) (any, error) {
		if !model.IsStrictExpr(s) {
			return s, nil
		}
		if deferActivityState && referencesActivityState(s) {
			inputs.Deferred = append(inputs.Deferred, path)
			return s, nil
		}
		evaluated, err := EvaluateString(s, nil, state)
		if err != nil {
			return nil, err
		}
		if !survivesJSON(evaluated) {
			inputs.Deferred = append(inputs.Deferred, path)
			return s, nil
		}
		return evaluated, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return resolved, inputs, nil
}

// EvaluateActivityInput evaluates value, the part of an activity's input at
// prefix, inside the activity. When the workflow resolved the inputs, only the
// expressions it deferred under prefix are evaluated. Otherwise the activity
// was scheduled before inputs were resolved in the workflow, and every
// expression in value is evaluated, as it always was.
func EvaluateActivityInput(prefix string, value any, state *State) (any, error) {
	if state.ActivityInputs == nil {
		return walkInputs(swUtil.DeepCloneValue(value), prefix, func(_, s string) (any, error) {
			return EvaluateString(s, nil, state)
		})
	}
	return EvaluateDeferredActivityInput(prefix, value, state)
}

// EvaluateDeferredActivityInput evaluates only the expressions the workflow
// deferred under prefix, over the same shapes as EvaluateActivityInput. An
// activity scheduled with unresolved inputs gets value back unchanged.
func EvaluateDeferredActivityInput(prefix string, value any, state *State) (any, error) {
	if state.ActivityInputs == nil {
		return value, nil
	}
	return walkInputs(swUtil.DeepCloneValue(value), prefix, func(path, s string) (any, error) {
		if !slices.Contains(state.ActivityInputs.Deferred, path) {
			return s, nil
		}
		return EvaluateString(s, nil, state)
	})
}

// walkInputs replaces every string in node with visit's result, over the same
// shapes, and with the same map[string]string coercion, as traverseAndEvaluate.
// path is the JSON Pointer of node. Map keys are visited in sorted order: this
// runs in workflow code, so the first error and the deferred order must not
// depend on Go's random map iteration.
func walkInputs(node any, path string, visit func(path, s string) (any, error)) (any, error) {
	switch v := node.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(v)) {
			out, err := walkInputs(v[key], path+"/"+escapePointerToken(key), visit)
			if err != nil {
				return nil, err
			}
			v[key] = out
		}
		return v, nil
	case map[string]string:
		// DeepCloneValue does not clone map[string]string, so never write to v.
		clone := make(map[string]string, len(v))
		for _, key := range slices.Sorted(maps.Keys(v)) {
			out, err := visit(path+"/"+escapePointerToken(key), v[key])
			if err != nil {
				return nil, err
			}
			if out == nil {
				clone[key] = ""
			} else {
				clone[key] = fmt.Sprintf("%v", out)
			}
		}
		return clone, nil
	case []any:
		for i, item := range v {
			out, err := walkInputs(item, path+"/"+strconv.Itoa(i), visit)
			if err != nil {
				return nil, err
			}
			v[i] = out
		}
		return v, nil
	case []string:
		return walkInputs(toAnySlice(v), path, visit)
	case string:
		return visit(path, v)
	default:
		return v, nil
	}
}

// maxExactJSONInt is the largest integer a float64 holds exactly. Temporal
// decodes every JSON number in an activity's arguments as a float64.
const maxExactJSONInt = 1 << 53

// survivesJSON reports whether v reaches the activity unchanged: no integer in
// it is beyond what a float64 holds exactly, and no number is NaN or infinite.
func survivesJSON(v any) bool {
	switch x := v.(type) {
	case float64:
		return !math.IsNaN(x) && !math.IsInf(x, 0)
	case int:
		return -maxExactJSONInt <= x && x <= maxExactJSONInt
	case *big.Int:
		return x.IsInt64() && -maxExactJSONInt <= x.Int64() && x.Int64() <= maxExactJSONInt
	case map[string]any:
		for _, item := range x {
			if !survivesJSON(item) {
				return false
			}
		}
	case []any:
		for _, item := range x {
			if !survivesJSON(item) {
				return false
			}
		}
	}
	return true
}

func escapePointerToken(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

const dataVariable = "$data"

// referencesActivityState reports whether a strict-form runtime expression may
// read $data.activity. It fails closed: $data counts unless its first access is
// a static key other than "activity" ($data.foo, $data."foo", $data["foo"]),
// and an expression that does not parse counts too. A false positive only
// leaves the expression for the activity, which evaluates it with the same
// state plus the activity metadata.
func referencesActivityState(expr string) bool {
	query, err := gojq.Parse(model.SanitizeExpr(expr))
	if err != nil {
		return true
	}
	found := false
	walkTerms(reflect.ValueOf(query), func(t *gojq.Term) {
		if !found && t.Type == gojq.TermTypeFunc && t.Func != nil && t.Func.Name == dataVariable {
			found = !firstAccessIsStaticNonActivityKey(t.SuffixList)
		}
	})
	return found
}

func firstAccessIsStaticNonActivityKey(suffixes []*gojq.Suffix) bool {
	if len(suffixes) == 0 || suffixes[0].Index == nil || suffixes[0].Iter {
		return false
	}
	key, ok := staticIndexKey(suffixes[0].Index)
	return ok && key != "activity"
}

func staticIndexKey(index *gojq.Index) (string, bool) {
	switch {
	case index.IsSlice:
		return "", false
	case index.Name != "":
		return index.Name, true
	case index.Str != nil && len(index.Str.Queries) == 0:
		return index.Str.Str, true
	case index.Start != nil && index.End == nil && index.Start.Term != nil &&
		index.Start.Term.Type == gojq.TermTypeString && index.Start.Term.Str != nil &&
		len(index.Start.Term.Str.Queries) == 0 && len(index.Start.Term.SuffixList) == 0:
		return index.Start.Term.Str.Str, true
	}
	return "", false
}

// walkTerms calls visit for every *gojq.Term reachable from v. Reflection keeps
// the walk complete without mirroring every gojq node type by hand.
func walkTerms(v reflect.Value, visit func(*gojq.Term)) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return
		}
		if t, ok := reflect.TypeAssert[*gojq.Term](v); ok {
			visit(t)
		}
		walkTerms(v.Elem(), visit)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				walkTerms(v.Field(i), visit)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			walkTerms(v.Index(i), visit)
		}
	}
}

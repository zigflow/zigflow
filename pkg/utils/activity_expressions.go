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
	"github.com/itchyny/gojq"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

// ExpressionReferencesActivityState reports whether expr reads from
// $data.activity, the activity-runtime metadata branch (e.g.
// $data.activity.attempt) that only exists once an activity is running.
//
// Such expressions must not be evaluated workflow-side before the activity is
// scheduled: $data.activity is absent there and the expression would silently
// resolve to null. They are instead preserved verbatim and evaluated inside the
// activity against activity-enriched state.
//
// The detection walks the parsed jq AST and flags a $data variable that is
// indexed by .activity as its first access. It fails safe: an expression that
// cannot be parsed is treated as referencing activity state, so an
// un-analysable expression is preserved rather than resolved against
// incomplete workflow-side state. Unparseable expressions are independently
// rejected by validation.
func ExpressionReferencesActivityState(expr string) bool {
	if model.IsStrictExpr(expr) {
		expr = model.SanitizeExpr(expr)
	}
	query, err := gojq.Parse(expr)
	if err != nil {
		return true
	}
	return queryRefsActivity(query)
}

func queryRefsActivity(q *gojq.Query) bool {
	if q == nil {
		return false
	}
	for _, fd := range q.FuncDefs {
		if fd != nil && queryRefsActivity(fd.Body) {
			return true
		}
	}
	if q.Term != nil {
		return termRefsActivity(q.Term)
	}
	return queryRefsActivity(q.Left) || queryRefsActivity(q.Right)
}

//nolint:gocyclo // a flat type switch over the gojq term variants; each arm is trivial
func termRefsActivity(t *gojq.Term) bool {
	if t == nil {
		return false
	}

	// The pattern we are detecting: the $data variable indexed by .activity as
	// its first access ($data.activity, $data.activity.attempt, $data["activity"]).
	if t.Type == gojq.TermTypeFunc && t.Func != nil && t.Func.Name == StateVarData &&
		firstSuffixIsActivity(t.SuffixList) {
		return true
	}

	switch t.Type {
	case gojq.TermTypeIndex:
		if indexRefsActivity(t.Index) {
			return true
		}
	case gojq.TermTypeFunc:
		if t.Func != nil {
			for _, arg := range t.Func.Args {
				if queryRefsActivity(arg) {
					return true
				}
			}
		}
	case gojq.TermTypeObject:
		if objectRefsActivity(t.Object) {
			return true
		}
	case gojq.TermTypeArray:
		if t.Array != nil && queryRefsActivity(t.Array.Query) {
			return true
		}
	case gojq.TermTypeUnary:
		if t.Unary != nil && termRefsActivity(t.Unary.Term) {
			return true
		}
	case gojq.TermTypeString:
		if stringRefsActivity(t.Str) {
			return true
		}
	case gojq.TermTypeIf:
		if ifRefsActivity(t.If) {
			return true
		}
	case gojq.TermTypeTry:
		if t.Try != nil && (queryRefsActivity(t.Try.Body) || queryRefsActivity(t.Try.Catch)) {
			return true
		}
	case gojq.TermTypeReduce:
		if r := t.Reduce; r != nil &&
			(queryRefsActivity(r.Query) || queryRefsActivity(r.Start) || queryRefsActivity(r.Update)) {
			return true
		}
	case gojq.TermTypeForeach:
		if f := t.Foreach; f != nil &&
			(queryRefsActivity(f.Query) || queryRefsActivity(f.Start) ||
				queryRefsActivity(f.Update) || queryRefsActivity(f.Extract)) {
			return true
		}
	case gojq.TermTypeLabel:
		if t.Label != nil && queryRefsActivity(t.Label.Body) {
			return true
		}
	case gojq.TermTypeQuery:
		if queryRefsActivity(t.Query) {
			return true
		}
	}

	// Suffixes can carry their own inner queries (slice bounds, interpolated
	// index strings); walk them so a nested reference is not missed.
	for _, s := range t.SuffixList {
		if s != nil && indexRefsActivity(s.Index) {
			return true
		}
	}
	return false
}

// firstSuffixIsActivity reports whether the first index access in a suffix list
// indexes the "activity" key. A leading iterator (.[]) means the first access
// is not a plain .activity index, so it stops looking.
func firstSuffixIsActivity(suffixes []*gojq.Suffix) bool {
	for _, s := range suffixes {
		if s == nil {
			continue
		}
		if s.Index != nil {
			return indexNameIs(s.Index, ActivityStateKey)
		}
		if s.Iter {
			return false
		}
		// An optional (?) suffix guards the following index; keep scanning.
	}
	return false
}

// indexNameIs reports whether an index selects the given literal key, via the
// dot form (.activity) or a constant bracket form (["activity"]).
func indexNameIs(i *gojq.Index, name string) bool {
	if i == nil {
		return false
	}
	if i.Name == name {
		return true
	}
	if constStringIs(i.Str, name) {
		return true
	}
	// Bracket form, e.g. $data["activity"]: gojq stores the constant key as a
	// string-literal query in Start.
	if !i.IsSlice && i.End == nil && queryConstStringIs(i.Start, name) {
		return true
	}
	return false
}

// constStringIs reports whether s is a constant (non-interpolated) string
// literal equal to name.
func constStringIs(s *gojq.String, name string) bool {
	return s != nil && len(s.Queries) == 0 && s.Str == name
}

// queryConstStringIs reports whether q is a bare constant string literal equal
// to name (the shape gojq produces for a constant bracket index key).
func queryConstStringIs(q *gojq.Query, name string) bool {
	if q == nil || q.Term == nil {
		return false
	}
	t := q.Term
	return t.Type == gojq.TermTypeString && len(t.SuffixList) == 0 && constStringIs(t.Str, name)
}

func indexRefsActivity(i *gojq.Index) bool {
	if i == nil {
		return false
	}
	return stringRefsActivity(i.Str) || queryRefsActivity(i.Start) || queryRefsActivity(i.End)
}

func objectRefsActivity(o *gojq.Object) bool {
	if o == nil {
		return false
	}
	for _, kv := range o.KeyVals {
		if kv == nil {
			continue
		}
		if stringRefsActivity(kv.KeyString) || queryRefsActivity(kv.KeyQuery) || queryRefsActivity(kv.Val) {
			return true
		}
	}
	return false
}

func stringRefsActivity(s *gojq.String) bool {
	if s == nil {
		return false
	}
	for _, q := range s.Queries {
		if queryRefsActivity(q) {
			return true
		}
	}
	return false
}

func ifRefsActivity(i *gojq.If) bool {
	if i == nil {
		return false
	}
	if queryRefsActivity(i.Cond) || queryRefsActivity(i.Then) || queryRefsActivity(i.Else) {
		return true
	}
	for _, el := range i.Elif {
		if el != nil && (queryRefsActivity(el.Cond) || queryRefsActivity(el.Then)) {
			return true
		}
	}
	return false
}

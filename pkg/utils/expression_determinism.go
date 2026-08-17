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

	"github.com/itchyny/gojq"
	"github.com/open-workflow-specification/sdk-go/v4/model"
)

// DeterminismIssue describes one non-deterministic construct found in an
// expression. Path is a best-effort breadcrumb describing where in the
// expression tree the issue was found, useful for error messages.
type DeterminismIssue struct {
	Symbol string
	Reason string
	Path   string
}

func (d DeterminismIssue) String() string {
	if d.Path != "" {
		return fmt.Sprintf("%s (%s) at %s", d.Symbol, d.Reason, d.Path)
	}
	return fmt.Sprintf("%s (%s)", d.Symbol, d.Reason)
}

// DeterminismAnalysis is the result of analysing an expression for Temporal
// replay safety. Deterministic is true when every symbol referenced by the
// expression is derivable from workflow state or from a known pure builtin.
type DeterminismAnalysis struct {
	Deterministic bool
	Issues        []DeterminismIssue
}

// AnalyseExpressionDeterminism parses expr with gojq and walks the full AST to
// determine whether the expression is replay-safe for Temporal durable
// execution. An expression is replay-safe when every value it produces is a
// function of workflow state, input, or known pure builtins. Anything that
// reads the host clock, host environment, or any other ambient state is
// non-deterministic and must be wrapped in a Set task.
//
// expr may include the strict expression form (${ ... }); the wrapper is
// stripped before parsing.
func AnalyseExpressionDeterminism(expr string) (*DeterminismAnalysis, error) {
	if model.IsStrictExpr(expr) {
		expr = model.SanitizeExpr(expr)
	}
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression %q: %w", expr, err)
	}
	a := &determinismAnalyser{}
	a.walkQuery(query, newScope(nil), "")
	return &DeterminismAnalysis{
		Deterministic: len(a.issues) == 0,
		Issues:        a.issues,
	}, nil
}

// IsExpressionDeterministic is a convenience wrapper around
// AnalyseExpressionDeterminism that returns false on parse errors so callers
// fail closed.
func IsExpressionDeterministic(expr string) bool {
	a, err := AnalyseExpressionDeterminism(expr)
	if err != nil {
		return false
	}
	return a.Deterministic
}

// stateVars is the set form of StateVarNames, the variable names Zigflow
// injects into every jq evaluation from workflow State. Reading these values is
// always replay-safe because they come from workflow history. It is derived
// from StateVarNames so the determinism allow-list cannot drift from the names
// actually injected by State.GetAsMap.
var stateVars = func() map[string]struct{} {
	set := make(map[string]struct{}, len(StateVarNames))
	for _, name := range StateVarNames {
		set[name] = struct{}{}
	}
	return set
}()

// nonDeterministicReasons lists symbols that are explicitly non-deterministic
// and the reason they are unsafe. Used to produce useful error messages; the
// allow-list in deterministicBuiltins is what actually drives classification.
var nonDeterministicReasons = map[string]string{
	"now":           "reads the host wall-clock",
	"localtime":     "reads the host timezone",
	"strflocaltime": "reads the host timezone",
	"env":           "reads host environment variables",
	"input":         "reads from an external input stream",
	"inputs":        "reads from an external input stream",
	"debug":         "writes to stderr as a side effect",
	"stderr":        "writes to stderr as a side effect",
	"$ENV":          "reads host environment variables",
	"$__loc__":      "exposes compile-time location which varies between builds",
}

// deterministicBuiltins is the allow-list of gojq builtins that are pure and
// replay-safe. Anything not listed here is treated as non-deterministic, so
// the analyser fails closed for builtins added in future gojq releases.
//
// Sourced from gojq's internalFuncs and builtin.jq. Functions that read the
// host clock, host environment, or any external stream are deliberately
// omitted.
var deterministicBuiltins = map[string]struct{}{
	// Pure value operations
	"empty":           {},
	"not":             {},
	"path":            {},
	"paths":           {},
	"leaf_paths":      {},
	"builtins":        {},
	"abs":             {},
	"length":          {},
	"utf8bytelength":  {},
	"keys":            {},
	"keys_unsorted":   {},
	"values":          {},
	"has":             {},
	"in":              {},
	"add":             {},
	"any":             {},
	"all":             {},
	"toboolean":       {},
	"tonumber":        {},
	"tostring":        {},
	"type":            {},
	"reverse":         {},
	"contains":        {},
	"inside":          {},
	"indices":         {},
	"index":           {},
	"rindex":          {},
	"startswith":      {},
	"endswith":        {},
	"ltrimstr":        {},
	"rtrimstr":        {},
	"trimstr":         {},
	"ltrim":           {},
	"rtrim":           {},
	"trim":            {},
	"explode":         {},
	"implode":         {},
	"split":           {},
	"splits":          {},
	"join":            {},
	"ascii":           {},
	"ascii_downcase":  {},
	"ascii_upcase":    {},
	"tojson":          {},
	"fromjson":        {},
	"tostream":        {},
	"fromstream":      {},
	"truncate_stream": {},
	"format":          {},

	// Selection / iteration
	"select":       {},
	"map":          {},
	"map_values":   {},
	"recurse":      {},
	"recurse_down": {},
	"walk":         {},
	"to_entries":   {},
	"from_entries": {},
	"with_entries": {},
	"while":        {},
	"until":        {},
	"repeat":       {},
	"range":        {},
	"limit":        {},
	"first":        {},
	"last":         {},
	"nth":          {},
	"isempty":      {},
	"truncate":     {},
	"skip":         {},

	// Type predicates
	"arrays":    {},
	"objects":   {},
	"iterables": {},
	"booleans":  {},
	"numbers":   {},
	"strings":   {},
	"nulls":     {},
	"values_":   {},
	"scalars":   {},
	"finites":   {},
	"infinites": {},
	"normals":   {},

	// Sorting / grouping
	"min":       {},
	"min_by":    {},
	"max":       {},
	"max_by":    {},
	"sort":      {},
	"sort_by":   {},
	"group_by":  {},
	"unique":    {},
	"unique_by": {},

	// Path operations
	"flatten":   {},
	"setpath":   {},
	"delpaths":  {},
	"getpath":   {},
	"transpose": {},
	"bsearch":   {},
	"del":       {},
	"pick":      {},
	"paths_":    {},

	// Time formatting (operate on input values; do not read the clock)
	"gmtime":          {},
	"mktime":          {},
	"strftime":        {},
	"strptime":        {},
	"todate":          {},
	"fromdate":        {},
	"todateiso8601":   {},
	"fromdateiso8601": {},

	// Regex (deterministic against input)
	"test":    {},
	"match":   {},
	"capture": {},
	"scan":    {},
	"sub":     {},
	"gsub":    {},
	"splits_": {},

	// Math (pure)
	"sin": {}, "cos": {}, "tan": {},
	"asin": {}, "acos": {}, "atan": {},
	"sinh": {}, "cosh": {}, "tanh": {},
	"asinh": {}, "acosh": {}, "atanh": {},
	"floor": {}, "round": {}, "nearbyint": {}, "rint": {},
	"ceil": {}, "trunc": {}, "significand": {},
	"fabs": {}, "sqrt": {}, "cbrt": {},
	"exp": {}, "exp10": {}, "exp2": {}, "expm1": {},
	"frexp": {}, "modf": {},
	"log": {}, "log10": {}, "log1p": {}, "log2": {}, "logb": {},
	"gamma": {}, "tgamma": {}, "lgamma": {},
	"erf": {}, "erfc": {},
	"j0": {}, "j1": {}, "jn": {},
	"y0": {}, "y1": {}, "yn": {},
	"atan2": {}, "copysign": {}, "drem": {},
	"fdim": {}, "fmax": {}, "fmin": {}, "fmod": {},
	"hypot": {}, "nextafter": {}, "nexttoward": {},
	"remainder": {}, "ldexp": {}, "scalb": {}, "scalbln": {},
	"pow": {}, "fma": {},
	"infinite": {}, "isfinite": {}, "isinfinite": {},
	"nan": {}, "isnan": {}, "isnormal": {},

	// Collection helpers
	"combinations": {},
	"INDEX":        {},
	"IN":           {},
	"JOIN":         {},

	// Error / control flow (deterministic, terminates the query)
	"error":      {},
	"halt":       {},
	"halt_error": {},

	// Aliases / private helpers prefixed with _ (used internally for operators)
	"_plus":        {},
	"_negate":      {},
	"_add":         {},
	"_subtract":    {},
	"_multiply":    {},
	"_divide":      {},
	"_modulo":      {},
	"_alternative": {},
	"_equal":       {},
	"_notequal":    {},
	"_greater":     {},
	"_less":        {},
	"_greatereq":   {},
	"_lesseq":      {},
	"_index":       {},
	"_slice":       {},
	"_range":       {},
	"_min_by":      {},
	"_max_by":      {},
	"_sort_by":     {},
	"_group_by":    {},
	"_unique_by":   {},
	"_match":       {},
	"_captures":    {},
}

// IsDeterministicBuiltin reports whether name is a known pure gojq builtin.
// Exposed for tests and to keep classification centralised in one place.
func IsDeterministicBuiltin(name string) bool {
	_, ok := deterministicBuiltins[name]
	return ok
}

// scope tracks locally introduced names (FuncDef names and arguments, plus
// pattern variables from `as $x`). Names are looked up walking from innermost
// to outermost scope.
type scope struct {
	parent *scope
	names  map[string]struct{}
}

func newScope(parent *scope) *scope {
	return &scope{parent: parent, names: map[string]struct{}{}}
}

func (s *scope) add(name string) {
	s.names[name] = struct{}{}
}

func (s *scope) contains(name string) bool {
	for cur := s; cur != nil; cur = cur.parent {
		if _, ok := cur.names[name]; ok {
			return true
		}
	}
	return false
}

type determinismAnalyser struct {
	issues []DeterminismIssue
}

func (a *determinismAnalyser) flag(symbol, reason, path string) {
	a.issues = append(a.issues, DeterminismIssue{Symbol: symbol, Reason: reason, Path: path})
}

func joinPath(path, segment string) string {
	if path == "" {
		return segment
	}
	return path + "/" + segment
}

func (a *determinismAnalyser) walkQuery(q *gojq.Query, sc *scope, path string) {
	if q == nil {
		return
	}
	// Imports are not replay-safe: the loaded module can introduce arbitrary
	// definitions including ones that read external state.
	for _, im := range q.Imports {
		target := im.ImportPath
		if target == "" {
			target = im.IncludePath
		}
		a.flag("import", fmt.Sprintf("module %q is loaded at evaluation time and is not replay-safe", target), path)
	}

	// FuncDef names are hoisted within their containing query so a definition
	// can be referenced before it appears textually. Add them all first, then
	// walk each body in a child scope containing its argument names.
	inner := newScope(sc)
	for _, fd := range q.FuncDefs {
		inner.add(fd.Name)
	}
	for _, fd := range q.FuncDefs {
		fnScope := newScope(inner)
		for _, arg := range fd.Args {
			fnScope.add(arg)
		}
		a.walkQuery(fd.Body, fnScope, joinPath(path, "def "+fd.Name))
	}

	if q.Term != nil {
		a.walkTerm(q.Term, inner, path)
		return
	}

	if q.Right != nil {
		a.walkQuery(q.Left, inner, joinPath(path, q.Op.String()))
		// `Left as Pattern | Right` introduces pattern variables that are in
		// scope for Right only.
		rightScope := inner
		if len(q.Patterns) > 0 {
			rightScope = newScope(inner)
			for _, p := range q.Patterns {
				collectPatternVars(rightScope, p)
			}
		}
		a.walkQuery(q.Right, rightScope, joinPath(path, q.Op.String()))
	}
}

func (a *determinismAnalyser) walkTerm(t *gojq.Term, sc *scope, path string) {
	if t == nil {
		return
	}
	a.walkTermBody(t, sc, path)
	for _, suffix := range t.SuffixList {
		if suffix.Index != nil {
			a.walkIndex(suffix.Index, sc, path)
		}
	}
}

func (a *determinismAnalyser) walkTermBody(t *gojq.Term, sc *scope, path string) {
	switch t.Type {
	case gojq.TermTypeIndex:
		a.walkIndex(t.Index, sc, path)
	case gojq.TermTypeFunc:
		a.walkFunc(t.Func, sc, path)
	case gojq.TermTypeObject:
		a.walkObject(t.Object, sc, path)
	case gojq.TermTypeArray:
		if t.Array != nil {
			a.walkQuery(t.Array.Query, sc, path)
		}
	case gojq.TermTypeUnary:
		if t.Unary != nil {
			a.walkTerm(t.Unary.Term, sc, joinPath(path, t.Unary.Op.String()))
		}
	case gojq.TermTypeString:
		a.walkString(t.Str, sc, path)
	case gojq.TermTypeIf:
		a.walkIf(t.If, sc, joinPath(path, "if"))
	case gojq.TermTypeTry:
		a.walkTry(t.Try, sc, path)
	case gojq.TermTypeReduce:
		a.walkReduce(t.Reduce, sc, path)
	case gojq.TermTypeForeach:
		a.walkForeach(t.Foreach, sc, path)
	case gojq.TermTypeLabel:
		a.walkLabel(t.Label, sc, path)
	case gojq.TermTypeQuery:
		a.walkQuery(t.Query, sc, path)
	}
	// Remaining cases (Identity, Recurse, Null, True, False, Number, Format,
	// Break) are deterministic leaves with no children to walk.
}

func (a *determinismAnalyser) walkObject(o *gojq.Object, sc *scope, path string) {
	if o == nil {
		return
	}
	for _, kv := range o.KeyVals {
		if kv.KeyString != nil {
			a.walkString(kv.KeyString, sc, path)
		}
		if kv.KeyQuery != nil {
			a.walkQuery(kv.KeyQuery, sc, path)
		}
		if kv.Val != nil {
			a.walkQuery(kv.Val, sc, path)
		}
	}
}

func (a *determinismAnalyser) walkTry(t *gojq.Try, sc *scope, path string) {
	if t == nil {
		return
	}
	a.walkQuery(t.Body, sc, joinPath(path, "try"))
	a.walkQuery(t.Catch, sc, joinPath(path, "catch"))
}

func (a *determinismAnalyser) walkReduce(r *gojq.Reduce, sc *scope, path string) {
	if r == nil {
		return
	}
	a.walkQuery(r.Query, sc, joinPath(path, "reduce"))
	inner := newScope(sc)
	collectPatternVars(inner, r.Pattern)
	a.walkQuery(r.Start, inner, joinPath(path, "reduce/start"))
	a.walkQuery(r.Update, inner, joinPath(path, "reduce/update"))
}

func (a *determinismAnalyser) walkForeach(f *gojq.Foreach, sc *scope, path string) {
	if f == nil {
		return
	}
	a.walkQuery(f.Query, sc, joinPath(path, "foreach"))
	inner := newScope(sc)
	collectPatternVars(inner, f.Pattern)
	a.walkQuery(f.Start, inner, joinPath(path, "foreach/start"))
	a.walkQuery(f.Update, inner, joinPath(path, "foreach/update"))
	a.walkQuery(f.Extract, inner, joinPath(path, "foreach/extract"))
}

func (a *determinismAnalyser) walkLabel(l *gojq.Label, sc *scope, path string) {
	if l == nil {
		return
	}
	inner := newScope(sc)
	inner.add(l.Ident)
	a.walkQuery(l.Body, inner, joinPath(path, "label "+l.Ident))
}

// walkFunc classifies a referenced name as deterministic or not. A name is
// replay-safe when any of the following hold:
//
//  1. the name is locally declared (FuncDef, FuncDef arg, pattern var, label),
//  2. the name is a Zigflow workflow state variable ($context, $data, etc.),
//  3. the name is a Zigflow registered function explicitly marked deterministic,
//  4. the name is a gojq builtin on the deterministic allow-list.
//
// Any other identifier is flagged as non-deterministic. Unknown identifiers
// fail closed, so analysis remains safe for future gojq builtins and for
// user-defined names that have not been classified.
func (a *determinismAnalyser) walkFunc(f *gojq.Func, sc *scope, path string) {
	if f == nil {
		return
	}
	a.classifyName(f.Name, sc, path)
	for i, arg := range f.Args {
		a.walkQuery(arg, sc, fmt.Sprintf("%s(arg%d)", joinPath(path, f.Name), i))
	}
}

func (a *determinismAnalyser) classifyName(name string, sc *scope, path string) {
	if name == "" {
		return
	}
	// Locally declared names (FuncDefs, args, pattern vars, labels) are safe;
	// their bodies are walked separately.
	if sc.contains(name) {
		return
	}
	// Zigflow state variables are populated from workflow history.
	if _, ok := stateVars[name]; ok {
		return
	}
	// Zigflow-registered functions: trust the registry's classification.
	for _, fn := range jqFuncs {
		if fn.Name == name {
			if fn.Deterministic {
				return
			}
			a.flag(name, reasonFor(name, fmt.Sprintf("Zigflow function %q is non-deterministic", name)), path)
			return
		}
	}
	// Known deterministic gojq builtins.
	if _, ok := deterministicBuiltins[name]; ok {
		return
	}
	// Fail closed for everything else.
	a.flag(name, reasonFor(name, fmt.Sprintf("identifier %q is not a known replay-safe symbol", name)), path)
}

func reasonFor(name, fallback string) string {
	if r, ok := nonDeterministicReasons[name]; ok {
		return r
	}
	return fallback
}

// walkIndex walks an index suffix such as `.foo`, `.["x"]`, or `.[start:end]`.
// Index keys are deterministic; the inner queries (if any) are walked.
func (a *determinismAnalyser) walkIndex(i *gojq.Index, sc *scope, path string) {
	if i == nil {
		return
	}
	if i.Str != nil {
		a.walkString(i.Str, sc, path)
	}
	if i.Start != nil {
		a.walkQuery(i.Start, sc, path)
	}
	if i.End != nil {
		a.walkQuery(i.End, sc, path)
	}
}

func (a *determinismAnalyser) walkString(s *gojq.String, sc *scope, path string) {
	if s == nil {
		return
	}
	for _, q := range s.Queries {
		a.walkQuery(q, sc, path)
	}
}

func (a *determinismAnalyser) walkIf(i *gojq.If, sc *scope, path string) {
	if i == nil {
		return
	}
	a.walkQuery(i.Cond, sc, joinPath(path, "cond"))
	a.walkQuery(i.Then, sc, joinPath(path, "then"))
	for idx, el := range i.Elif {
		a.walkQuery(el.Cond, sc, fmt.Sprintf("%s/elif%d/cond", path, idx))
		a.walkQuery(el.Then, sc, fmt.Sprintf("%s/elif%d/then", path, idx))
	}
	a.walkQuery(i.Else, sc, joinPath(path, "else"))
}

// collectPatternVars walks a pattern and adds every variable it introduces to
// sc. Patterns appear in `as`, `reduce` and `foreach` constructs.
func collectPatternVars(sc *scope, p *gojq.Pattern) {
	if p == nil {
		return
	}
	if p.Name != "" {
		sc.add(p.Name)
	}
	for _, child := range p.Array {
		collectPatternVars(sc, child)
	}
	for _, kv := range p.Object {
		if kv.Val != nil {
			collectPatternVars(sc, kv.Val)
		}
	}
}

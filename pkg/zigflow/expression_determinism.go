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

package zigflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
)

// ErrNonDeterministicExpression is returned when a workflow contains a runtime
// expression whose result is not derivable from workflow state, outside a Set
// task. Such expressions break Temporal replay because the result is not
// captured in workflow history.
var ErrNonDeterministicExpression = errors.New("non-deterministic expression")

// NonDeterministicExpression identifies a single offending runtime expression
// and where it sits in the workflow document. It is the machine-readable unit
// behind a determinism failure.
type NonDeterministicExpression struct {
	Path       string `json:"path"`
	Expression string `json:"expression"`
}

// NonDeterministicExpressionError is returned by ValidateWorkflowDeterminism
// when one or more expressions outside a Set task are not replay-safe. It
// wraps ErrNonDeterministicExpression (so errors.Is still matches) and its
// Error() string is unchanged from the previous behaviour, while Expressions
// exposes the failures in structured form for callers that want to render or
// process them without parsing the message.
type NonDeterministicExpressionError struct {
	Expressions []NonDeterministicExpression
	msg         string
}

func (e *NonDeterministicExpressionError) Error() string { return e.msg }

func (e *NonDeterministicExpressionError) Unwrap() error { return ErrNonDeterministicExpression }

// ErrInvalidRuntimeExpression is returned when a workflow contains a runtime
// expression that cannot be parsed as jq, regardless of where it appears. This
// is a syntax failure, distinct from non-determinism: it would only surface at
// execution time otherwise.
var ErrInvalidRuntimeExpression = errors.New("invalid runtime expression")

// InvalidRuntimeExpression identifies a single runtime expression that failed
// to parse, where it sits in the workflow document, and the underlying parse
// error.
type InvalidRuntimeExpression struct {
	Path       string `json:"path"`
	Expression string `json:"expression"`
	Err        error  `json:"-"`
}

// InvalidRuntimeExpressionError is returned by ValidateWorkflowDeterminism when
// one or more collected expressions cannot be parsed. It wraps
// ErrInvalidRuntimeExpression (so errors.Is matches) and exposes the failures
// in structured form for callers that want to render them.
type InvalidRuntimeExpressionError struct {
	Expressions []InvalidRuntimeExpression
	msg         string
}

func (e *InvalidRuntimeExpressionError) Error() string { return e.msg }

func (e *InvalidRuntimeExpressionError) Unwrap() error { return ErrInvalidRuntimeExpression }

// ExpressionRef identifies one strict-form runtime expression found inside a
// workflow definition. It records where the expression is in the document and
// whether the location permits non-deterministic constructs.
//
// Non-determinism is permitted in two kinds of location: Set task bodies (the
// `set` map), which the SetTask builder evaluates inside workflow.SideEffect so
// the result is captured in workflow history; and document.metadata.scheduleInput,
// which is interpolated at schedule-registration time, outside the workflow.
// Every other location, including a Set task's own TaskBase fields
// (if/input/output/export/then/metadata), must be deterministic.
type ExpressionRef struct {
	Value                string
	Path                 string
	AllowsNonDeterminism bool
}

// CollectExpressions walks doc and returns every strict-form runtime
// expression (${ ... }) it contains, tagged with location and whether
// non-determinism is permitted there. The returned slice is in document
// order so error messages remain stable across runs.
func CollectExpressions(doc *model.Workflow) ([]ExpressionRef, error) {
	if doc == nil {
		return nil, nil
	}
	c := &expressionCollector{}
	if doc.Input != nil {
		if err := c.collectRaw(doc.Input, "workflow.input", false); err != nil {
			return nil, err
		}
	}
	if doc.Output != nil {
		if err := c.collectRaw(doc.Output, "workflow.output", false); err != nil {
			return nil, err
		}
	}
	if doc.Schedule != nil {
		if err := c.collectRaw(doc.Schedule, "workflow.schedule", false); err != nil {
			return nil, err
		}
	}
	// document.metadata.scheduleInput is interpolated at schedule-registration
	// time (see metadata.GetScheduleInfo), so it must be parse/compile-validated
	// like any other runtime expression. It runs outside the workflow, so it is
	// exempt from the determinism rule (AllowsNonDeterminism=true), exactly like
	// a Set body. Other document.metadata keys are static config and are not
	// evaluated as expressions, so they are deliberately not collected.
	if doc.Document.Metadata != nil {
		if input, ok := doc.Document.Metadata[metadata.MetadataScheduleInput]; ok {
			path := "document.metadata." + metadata.MetadataScheduleInput
			if err := c.collectRaw(input, path, true); err != nil {
				return nil, err
			}
		}
	}
	if doc.Do != nil {
		if err := c.collectTaskList(doc.Do, "do"); err != nil {
			return nil, err
		}
	}
	return c.refs, nil
}

// ValidateWorkflowDeterminism is the single workflow-level expression
// validation check. It walks doc and validates every collected expression.
//
// Every expression is parsed and compiled (via utils.CompileExpression, using
// the same gojq options as runtime) regardless of location. An expression that
// cannot parse or compile returns an *InvalidRuntimeExpressionError (wrapping
// ErrInvalidRuntimeExpression), which takes priority over any determinism
// findings.
//
// Determinism is only evaluated for expressions that parse and compile
// successfully. Expressions inside a Set task's `set` body are exempt from the
// determinism rule — but not from parse/compile validation — because they are
// evaluated under workflow.SideEffect and their result is captured in workflow
// history. A non-deterministic expression elsewhere returns an
// *NonDeterministicExpressionError (wrapping ErrNonDeterministicExpression).
func ValidateWorkflowDeterminism(doc *model.Workflow) error {
	refs, err := CollectExpressions(doc)
	if err != nil {
		return fmt.Errorf("collect expressions: %w", err)
	}
	var issues []string
	var failures []NonDeterministicExpression
	seen := map[NonDeterministicExpression]bool{}
	// addFailure records one offending location+expression exactly once, so the
	// same expression flagged for multiple non-deterministic symbols is not
	// reported as multiple failures to callers.
	addFailure := func(path, expression string) {
		f := NonDeterministicExpression{Path: path, Expression: expression}
		if seen[f] {
			return
		}
		seen[f] = true
		failures = append(failures, f)
	}

	var invalidIssues []string
	var invalids []InvalidRuntimeExpression
	addInvalid := func(path, expression string, err error) {
		invalidIssues = append(invalidIssues, fmt.Sprintf("%s: %s", path, err))
		invalids = append(invalids, InvalidRuntimeExpression{
			Path:       path,
			Expression: expression,
			Err:        err,
		})
	}

	for _, ref := range refs {
		// Every expression must parse and compile under the same gojq options
		// as runtime, regardless of where it appears. Set bodies are exempt
		// from the determinism rule, not from parse/compile validation, so this
		// gate runs before the AllowsNonDeterminism check.
		if err := utils.CompileExpression(ref.Value); err != nil {
			addInvalid(ref.Path, ref.Value, err)
			continue
		}

		// Determinism is only evaluated once parse and compile have succeeded,
		// and only for locations that do not permit non-determinism.
		if ref.AllowsNonDeterminism {
			continue
		}
		analysis, err := utils.AnalyseExpressionDeterminism(ref.Value)
		if err != nil {
			// Unreachable in practice: CompileExpression already parsed the
			// same expression. Classify defensively as invalid rather than
			// silently dropping the error.
			addInvalid(ref.Path, ref.Value, err)
			continue
		}
		if analysis.Deterministic {
			continue
		}
		addFailure(ref.Path, ref.Value)
		for _, issue := range analysis.Issues {
			issues = append(issues, fmt.Sprintf(
				"%s: expression %q uses non-deterministic value %q (%s). "+
					"Non-deterministic expressions are only allowed in Set tasks, "+
					"where the result is captured in workflow history. "+
					"Move this expression into a Set task and reference the stored value from workflow state",
				ref.Path, ref.Value, issue.Symbol, issue.Reason,
			))
		}
	}

	// Parse failures take priority over and are reported separately from
	// non-determinism: an unparseable expression has no meaningful determinism
	// verdict.
	if len(invalids) > 0 {
		return &InvalidRuntimeExpressionError{
			Expressions: invalids,
			msg:         formatValidationIssues(ErrInvalidRuntimeExpression, invalidIssues),
		}
	}

	if len(issues) == 0 {
		return nil
	}
	return &NonDeterministicExpressionError{
		Expressions: failures,
		msg:         formatValidationIssues(ErrNonDeterministicExpression, issues),
	}
}

// formatValidationIssues renders a sentinel error and its per-expression issue
// lines into the bullet-list message shared by the structured expression
// validation errors.
func formatValidationIssues(sentinel error, issues []string) string {
	return fmt.Sprintf("%s:\n  - %s", sentinel, strings.Join(issues, "\n  - "))
}

type expressionCollector struct {
	refs []ExpressionRef
}

func (c *expressionCollector) collectTaskList(list *model.TaskList, path string) error {
	if list == nil {
		return nil
	}
	for i, item := range *list {
		itemPath := fmt.Sprintf("%s[%d].%s", path, i, item.Key)
		if err := c.collectTaskItem(item, itemPath); err != nil {
			return err
		}
	}
	return nil
}

// collectTaskItem walks one TaskItem. TaskBase fields are always
// deterministic-only. The task body is split based on type: container tasks
// recurse into their child task lists, Set tasks contribute their `set` map
// as the only non-deterministic-permissive location, and every other task
// type has its full body walked as deterministic.
func (c *expressionCollector) collectTaskItem(item *model.TaskItem, path string) error {
	if item == nil || item.Task == nil {
		return nil
	}
	if base := item.GetBase(); base != nil {
		if err := c.collectRaw(base, path, false); err != nil {
			return err
		}
	}

	switch {
	case item.AsSetTask() != nil:
		st := item.AsSetTask()
		return c.collectRaw(st.Set, path+".set", true)

	case item.AsDoTask() != nil:
		dt := item.AsDoTask()
		return c.collectTaskList(dt.Do, path+".do")

	case item.AsForkTask() != nil:
		ft := item.AsForkTask()
		return c.collectTaskList(ft.Fork.Branches, path+".fork.branches")

	case item.AsForTask() != nil:
		ft := item.AsForTask()
		if err := c.collectRaw(ft.For, path+".for", false); err != nil {
			return err
		}
		c.addStringExpr(ft.While, path+".while", false)
		return c.collectTaskList(ft.Do, path+".do")

	case item.AsTryTask() != nil:
		tt := item.AsTryTask()
		if err := c.collectTaskList(tt.Try, path+".try"); err != nil {
			return err
		}
		if tt.Catch != nil {
			if err := c.collectTaskList(tt.Catch.Do, path+".catch.do"); err != nil {
				return err
			}
			if err := c.collectCatchControl(tt.Catch, path+".catch"); err != nil {
				return err
			}
		}
		return nil
	}

	// Default: leaf or extension task (Call*, Run, Wait, Listen, Emit, Raise,
	// Switch, WaitExt, …). Walk the entire task body as raw JSON, having
	// already covered the TaskBase fields above, and exclude them here so we
	// do not double-count.
	return c.collectTaskBodyExcludingBase(item.Task, path)
}

// collectCatchControl walks the deterministic parts of a TryTaskCatch: the
// When / ExceptWhen guards and the retry policy. The catch.do task list is
// walked separately by the caller.
func (c *expressionCollector) collectCatchControl(catch *model.TryTaskCatch, path string) error {
	temp := struct {
		Errors     any                      `json:"errors,omitempty"`
		As         string                   `json:"as,omitempty"`
		When       *model.RuntimeExpression `json:"when,omitempty"`
		ExceptWhen *model.RuntimeExpression `json:"exceptWhen,omitempty"`
		Retry      *model.RetryPolicy       `json:"retry,omitempty"`
	}{
		Errors:     catch.Errors,
		As:         catch.As,
		When:       catch.When,
		ExceptWhen: catch.ExceptWhen,
		Retry:      catch.Retry,
	}
	return c.collectRaw(temp, path, false)
}

// collectTaskBodyExcludingBase marshals a Task to JSON, drops the TaskBase
// fields (already walked by the caller), and walks the remainder.
func (c *expressionCollector) collectTaskBodyExcludingBase(task model.Task, path string) error {
	raw, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task at %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("unmarshal task at %s: %w", path, err)
	}
	for _, key := range taskBaseKeys {
		delete(m, key)
	}
	c.walkRaw(m, path, false)
	return nil
}

// taskBaseKeys lists every JSON field on model.TaskBase. Kept as a constant so
// the default case can drop them in one place when walking a task body.
var taskBaseKeys = []string{
	keyIf,
	keyInput,
	keyOutput,
	keyExport,
	keyTimeout,
	keyThen,
	keyMetadata,
}

const (
	keyIf       = "if"
	keyInput    = "input"
	keyOutput   = "output"
	keyExport   = "export"
	keyTimeout  = "timeout"
	keyThen     = "then"
	keyMetadata = "metadata"
)

func (c *expressionCollector) collectRaw(v any, path string, allowsNonDeterminism bool) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal at %s: %w", path, err)
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return fmt.Errorf("unmarshal at %s: %w", path, err)
	}
	c.walkRaw(node, path, allowsNonDeterminism)
	return nil
}

func (c *expressionCollector) walkRaw(node any, path string, allowsNonDeterminism bool) {
	switch v := node.(type) {
	case string:
		c.addStringExpr(v, path, allowsNonDeterminism)
	case map[string]any:
		// Range order over a map is randomised, so sort keys to keep collected
		// expressions in a stable order across runs (affects CLI/JSON output
		// and tests).
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			c.walkRaw(v[k], joinDocPath(path, k), allowsNonDeterminism)
		}
	case []any:
		for i, vv := range v {
			c.walkRaw(vv, fmt.Sprintf("%s[%d]", path, i), allowsNonDeterminism)
		}
	}
}

func (c *expressionCollector) addStringExpr(s, path string, allowsNonDeterminism bool) {
	if !model.IsStrictExpr(s) {
		return
	}
	c.refs = append(c.refs, ExpressionRef{
		Value:                s,
		Path:                 path,
		AllowsNonDeterminism: allowsNonDeterminism,
	})
}

func joinDocPath(path, segment string) string {
	if path == "" {
		return segment
	}
	return path + "." + segment
}

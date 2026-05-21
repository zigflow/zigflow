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

// Package flow models Serverless Workflow flow directive semantics
// (continue, exit, end, and named redirects) for use by Zigflow's
// workflow execution layer.
//
// Tasks emit a directive by returning one of the sentinel errors or
// a RedirectError. The enclosing sequential executor interprets the
// signal in its current scope; these errors are never user-facing
// workflow failures.
//
// A few directives, notably "end", must terminate the overall workflow
// rather than just the current scope. Because raw Go sentinel errors do
// not cross Temporal child workflow boundaries (they are serialised and
// reconstructed as opaque application errors), this package also exposes
// helpers that encode and decode flow.ErrEnd as a Temporal
// ApplicationError of type EndApplicationErrorType. Nested workflow
// executors emit this typed error; parent executors detect it on the
// child workflow result and convert it back into flow.ErrEnd so
// propagation continues upward.
package flow

import (
	"errors"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
)

// Sentinel errors signalling the enumerated flow directives.
var (
	ErrContinue = errors.New("flow directive: continue")
	ErrExit     = errors.New("flow directive: exit")
	ErrEnd      = errors.New("flow directive: end")
)

// EndApplicationErrorType is the Temporal ApplicationError type used to
// carry flow.ErrEnd across child workflow boundaries.
const EndApplicationErrorType = "zigflow.flow.end"

// RedirectError signals that execution should redirect to a named
// task. The enclosing executor decides how to dispatch to the target.
type RedirectError struct {
	Target string
}

func (e RedirectError) Error() string {
	return "flow directive: " + e.Target
}

// FromDirective maps a Serverless Workflow flow directive onto the
// matching sentinel error or RedirectError.
func FromDirective(then *model.FlowDirective) error {
	switch model.FlowDirectiveType(then.Value) {
	case model.FlowDirectiveContinue:
		return ErrContinue
	case model.FlowDirectiveExit:
		return ErrExit
	case model.FlowDirectiveEnd:
		return ErrEnd
	default:
		return RedirectError{Target: then.Value}
	}
}

// IsControlError reports whether err represents an internal flow
// directive rather than a genuine failure.
func IsControlError(err error) bool {
	if errors.Is(err, ErrContinue) ||
		errors.Is(err, ErrExit) ||
		errors.Is(err, ErrEnd) {
		return true
	}
	var redirect RedirectError
	return errors.As(err, &redirect)
}

// EndPayload is the serialisable payload attached to a Temporal end
// signal so that a child workflow's effective output can survive the
// trip back to its parent. Without this payload the end directive
// would terminate the workflow with whatever output happened to be set
// before the redirect, discarding any work the child did.
//
// Only Output is carried. Exported context is intentionally not
// propagated because Zigflow's existing redirect semantics rely on the
// parent's task-level Export directive to evaluate against the child's
// returned output; copying the child's context across the boundary
// would conflict with that contract. See executeRedirect in
// pkg/zigflow/tasks for how a successful redirect handles context.
type EndPayload struct {
	Output any `json:"output"`
}

// NewEndApplicationError returns a non-retryable Temporal ApplicationError
// of type EndApplicationErrorType. Nested workflow executors return this
// error when they observe flow.ErrEnd so parent workflows can detect
// that the end directive needs to keep propagating upward.
//
// The supplied output is attached as Temporal application-error details
// so the parent workflow can reconstruct the child's effective output
// without losing it across the boundary.
func NewEndApplicationError(output any) error {
	return temporal.NewNonRetryableApplicationError(
		ErrEnd.Error(),
		EndApplicationErrorType,
		ErrEnd,
		EndPayload{Output: output},
	)
}

// DecodeEndApplicationError reports whether err is (or wraps) the
// Temporal ApplicationError emitted by NewEndApplicationError, and if
// so returns the attached EndPayload. errors.As descends through
// *ChildWorkflowExecutionError so this works for errors returned from
// workflow.ExecuteChildWorkflow(...).Get(...).
//
// If err is a Zigflow end signal but carries no decodable payload
// (e.g. from an older worker), ok is true and payload is the zero
// value. Callers can still treat the error as an end directive and
// will simply observe a nil output.
func DecodeEndApplicationError(err error) (payload EndPayload, ok bool) {
	if err == nil {
		return EndPayload{}, false
	}
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		return EndPayload{}, false
	}
	if appErr.Type() != EndApplicationErrorType {
		return EndPayload{}, false
	}
	if !appErr.HasDetails() {
		return EndPayload{}, true
	}
	if dErr := appErr.Details(&payload); dErr != nil {
		return EndPayload{}, true
	}
	return payload, true
}

// IsEndApplicationError reports whether err is (or wraps) the Temporal
// ApplicationError emitted by NewEndApplicationError. Callers that
// need the carried output should use DecodeEndApplicationError instead.
func IsEndApplicationError(err error) bool {
	_, ok := DecodeEndApplicationError(err)
	return ok
}

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

package flow_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/temporal"
)

const (
	caseNameNil        = "nil"
	caseNamePlainError = "plain error"
	caseNameOtherType  = "different application error type"
)

func TestNewEndApplicationErrorRoundTrip(t *testing.T) {
	err := flow.NewEndApplicationError(nil)
	assert.Error(t, err)

	var appErr *temporal.ApplicationError
	assert.True(t, errors.As(err, &appErr), "end error must be a Temporal ApplicationError")
	assert.Equal(t, flow.EndApplicationErrorType, appErr.Type())
	assert.True(t, appErr.NonRetryable(), "end error must be non-retryable")
	assert.True(t, flow.IsEndApplicationError(err))
}

func TestIsEndApplicationErrorUnwrapsThroughWrappers(t *testing.T) {
	wrapped := fmt.Errorf("child workflow execution error: %w", flow.NewEndApplicationError(nil))
	assert.True(t, flow.IsEndApplicationError(wrapped),
		"IsEndApplicationError must descend through Unwrap chains")
}

// TestDecodeEndApplicationErrorRoundTripsPayload exercises the payload
// channel that Issue 1 added: nested workflow executors attach the
// effective output to the end signal so parent executors can apply it
// locally before continuing to propagate end upward.
func TestDecodeEndApplicationErrorRoundTripsPayload(t *testing.T) {
	cases := []struct {
		name   string
		output any
	}{
		{name: "string payload", output: "child-output"},
		{name: "map payload", output: map[string]any{"value": "child-output"}},
		{name: "nil payload (no end-time output)", output: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := flow.NewEndApplicationError(tc.output)
			payload, ok := flow.DecodeEndApplicationError(err)
			assert.True(t, ok, "DecodeEndApplicationError must report end signals as such")
			assert.Equal(t, tc.output, payload.Output)
		})
	}
}

func TestDecodeEndApplicationErrorRejectsNonEndErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: caseNameNil, err: nil},
		{name: caseNamePlainError, err: errors.New("boom")},
		{name: caseNameOtherType, err: temporal.NewApplicationError("boom", "other.type")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := flow.DecodeEndApplicationError(tc.err)
			assert.False(t, ok)
		})
	}
}

func TestIsEndApplicationErrorRejectsOtherErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: caseNameNil, err: nil},
		{name: caseNamePlainError, err: errors.New("boom")},
		{name: "ErrEnd sentinel without Temporal wrapping", err: flow.ErrEnd},
		{name: caseNameOtherType, err: temporal.NewApplicationError("boom", "other.type")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, flow.IsEndApplicationError(tc.err))
		})
	}
}

func TestIsControlErrorCoversAllDirectives(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		isCtrl  bool
		details string
	}{
		{name: "continue", err: flow.ErrContinue, isCtrl: true},
		{name: "exit", err: flow.ErrExit, isCtrl: true},
		{name: "end", err: flow.ErrEnd, isCtrl: true},
		{name: "redirect", err: flow.RedirectError{Target: "task-x"}, isCtrl: true},
		{name: caseNamePlainError, err: errors.New("boom"), isCtrl: false},
		{name: caseNameNil, err: nil, isCtrl: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isCtrl, flow.IsControlError(tc.err))
		})
	}
}

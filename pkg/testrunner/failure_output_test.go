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

package testrunner

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestFailureOutputFromError_owsError(t *testing.T) {
	ows := model.NewErrCommunication(errors.New("bad gateway"), "wf-1")
	out := failureOutputFromError(ows)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(500), m["status"])
}

func TestFailureOutputForRun_prefersWorkflowOutput(t *testing.T) {
	partial := map[string]any{"data": map[string]any{"step": "done"}}
	out := failureOutputForRun(partial, assert.AnError)
	assert.Equal(t, partial, out)
}

func TestFormatHuman_failed_includesOutput(t *testing.T) {
	var buf strings.Builder
	res := &Result{
		WorkflowType: "raise",
		Status:       StatusFailed,
		Duration:     800,
		Err:          assert.AnError,
		Output: map[string]any{
			"status": float64(400),
			"type":   "https://example.com/errors/communication",
		},
		TemporalUIURL: HistoryURL("http://localhost:8233", "default", "x", "y"),
	}
	require.NoError(t, FormatHuman(&buf, res))
	s := buf.String()
	assert.Contains(t, s, "✗ raise")
	assert.Contains(t, s, "output:")
	assert.Contains(t, s, "communication")
	assert.Contains(t, s, "error:")
	assert.Contains(t, s, "inspect:")
}

func TestFailureOutputFromError_applicationErrorDetails(t *testing.T) {
	ows := model.NewErrValidation(errors.New("invalid"), "wf-2")
	appErr := temporal.NewApplicationError("validation failed", "Validation", ows)
	out := failureOutputFromError(appErr)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(400), m["status"])
}

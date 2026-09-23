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
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBytesRejectsInvalidDefinition(t *testing.T) {
	result, err := RunBytes(t.Context(), []byte("invalid: ["), Config{})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestNewWorkflowID(t *testing.T) {
	id := NewWorkflowID("hello-world")
	assert.Contains(t, id, "hello-world-")
}

func TestHistoryURL(t *testing.T) {
	url := HistoryURL("http://localhost:8233", "default", "wf-abc", "run-123")
	assert.Equal(
		t,
		"http://localhost:8233/namespaces/default/workflows/wf-abc/run-123/history",
		url,
	)
}

func TestFormatHuman_completed(t *testing.T) {
	var buf strings.Builder
	res := &Result{
		WorkflowType: "hello-world",
		Status:       StatusCompleted,
		Duration:     1500 * time.Millisecond,
		Output:       map[string]any{"message": "hi"},
	}
	require.NoError(t, FormatHuman(&buf, res))
	assert.Contains(t, buf.String(), "✓ hello-world")
	assert.Contains(t, buf.String(), "message")
}

func TestFormatHuman_timeout(t *testing.T) {
	var buf strings.Builder
	timeout := 5 * time.Second
	res := &Result{
		WorkflowType:  "slow",
		Status:        StatusTimeout,
		Duration:      timeout,
		Err:           fmt.Errorf("%w after %s: %w", ErrTestTimeout, timeout, context.DeadlineExceeded),
		TemporalUIURL: HistoryURL("http://localhost:8233", "default", "x", "y"),
	}
	require.NoError(t, FormatHuman(&buf, res))
	assert.Contains(t, buf.String(), "Timeout")
	assert.Contains(t, buf.String(), "workflow test timed out")
	assert.True(t, errors.Is(res.Err, ErrTestTimeout))
	assert.True(t, errors.Is(res.Err, context.DeadlineExceeded))
}

func TestFormatHuman_failed(t *testing.T) {
	var buf strings.Builder
	res := &Result{
		WorkflowType:  "hello-world",
		Status:        StatusFailed,
		Duration:      800 * time.Millisecond,
		Err:           assert.AnError,
		Output:        map[string]any{"message": "workflow failed"},
		TemporalUIURL: HistoryURL("http://localhost:8233", "default", "x", "y"),
	}
	require.NoError(t, FormatHuman(&buf, res))
	assert.Contains(t, buf.String(), "✗ hello-world")
	assert.Contains(t, buf.String(), "output:")
	assert.Contains(t, buf.String(), "inspect:")
}

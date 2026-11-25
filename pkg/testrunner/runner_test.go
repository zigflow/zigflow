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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	zigtemporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/zigflow"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/worker"
)

func TestRunBytesRejectsInvalidDefinition(t *testing.T) {
	result, err := RunBytes(t.Context(), []byte("invalid: ["), &Config{})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestRunBytesDoesNotMutateConnectionOptions(t *testing.T) {
	t.Setenv("TEMPORAL_ADDRESS", "localhost:17233")
	t.Setenv("TEMPORAL_NAMESPACE", "test-namespace")
	connection := &zigtemporal.TemporalOpts{}
	_, err := RunBytes(t.Context(), []byte("invalid: ["), &Config{Temporal: connection})
	require.Error(t, err)
	assert.Equal(t, &zigtemporal.TemporalOpts{}, connection)
}

func TestSelectWorkflowType(t *testing.T) {
	definition, err := os.ReadFile(filepath.Join("..", "..", "tests", "e2e", "tests", "multi-workflow-do-only", "workflow.yaml"))
	require.NoError(t, err)
	wf, err := zigflow.LoadFromBytes(definition)
	require.NoError(t, err)

	_, err = selectWorkflowType(wf, "")
	require.ErrorContains(t, err, "Config.WorkflowType or --workflow-type")
	assert.ErrorContains(t, err, "workflow1, workflow2")

	selected, err := selectWorkflowType(wf, "workflow2")
	require.NoError(t, err)
	assert.Equal(t, "workflow2", selected)

	_, err = selectWorkflowType(wf, "ignored")
	require.ErrorContains(t, err, "not registered")
	assert.ErrorContains(t, err, "workflow1, workflow2")

	singleDefinition, err := os.ReadFile(filepath.Join("..", "..", "examples", "for-loop", "workflow.yaml"))
	require.NoError(t, err)
	single, err := zigflow.LoadFromBytes(singleDefinition)
	require.NoError(t, err)
	selected, err = selectWorkflowType(single, "")
	require.NoError(t, err)
	assert.Equal(t, "for-loop", selected)
}

type emptyHistory struct{}

func (emptyHistory) HasNext() bool                        { return false }
func (emptyHistory) Next() (*history.HistoryEvent, error) { return nil, nil }

type stoppedWorker struct{ worker.Worker }

func (stoppedWorker) Stop() {}

func TestRunBytesTerminatesAmbiguousStart(t *testing.T) {
	c := new(mocks.Client)
	var acceptedID, terminatedID string
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, "ambiguous-start", mock.Anything).
		Run(func(args mock.Arguments) {
			acceptedID = args.Get(1).(client.StartWorkflowOptions).ID
		}).Return(nil, context.DeadlineExceeded).Once()
	c.On("TerminateWorkflow", mock.Anything, mock.Anything, "", testFailedTerminateReason).
		Run(func(args mock.Arguments) { terminatedID = args.String(1) }).Return(nil).Once()
	c.On("GetWorkflowHistory", mock.Anything, mock.Anything, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
		Return(emptyHistory{}).Once()
	c.On("ListWorkflow", mock.Anything, mock.Anything).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).Once()
	c.On("Close").Return().Once()
	definition := []byte(`document:
  dsl: 1.0.0
  taskQueue: original-queue
  workflowType: ambiguous-start
  version: 0.0.1
do:
  - wait:
      wait:
        seconds: 45
`)
	result, err := runBytesWithSession(t.Context(), definition, &Config{Timeout: time.Second},
		func(context.Context, *model.Workflow, *Config) (*session, error) {
			return &session{client: c, worker: stoppedWorker{}, taskQueue: TaskQueuePrefix + "test"}, nil
		})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, StatusFailed, result.Status)
	assert.ErrorIs(t, result.Err, context.DeadlineExceeded)
	assert.NotEmpty(t, acceptedID)
	assert.Equal(t, acceptedID, terminatedID)
	assert.Equal(t, acceptedID, result.WorkflowID)
	c.AssertExpectations(t)
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

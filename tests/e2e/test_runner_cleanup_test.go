//go:build e2e

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

package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/testrunner"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
)

func TestE2EZigflowTestCallerCancellationTerminatesExecution(t *testing.T) {
	c, err := temporal.NewConnectionWithEnvvars()
	require.NoError(t, err)
	defer c.Close()

	workflowType := "cancel-test-" + uuid.NewString()
	definition := fmt.Sprintf(`document:
  dsl: 1.0.0
  taskQueue: shared-document-queue
  workflowType: %s
  version: 0.0.1
do:
  - wait:
      wait:
        seconds: 45
`, workflowType)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type outcome struct {
		result *testrunner.Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, runErr := testrunner.RunBytes(ctx, []byte(definition), &testrunner.Config{
			Timeout: time.Minute,
		})
		done <- outcome{result, runErr}
	}()

	queryCtx, stopQuery := context.WithTimeout(t.Context(), 30*time.Second)
	defer stopQuery()
	var workflowID string
	for workflowID == "" && queryCtx.Err() == nil {
		listed, listErr := c.ListWorkflow(queryCtx, &workflowservice.ListWorkflowExecutionsRequest{
			Query: fmt.Sprintf("WorkflowType = '%s' AND ExecutionStatus = 'Running'", workflowType),
		})
		require.NoError(t, listErr)
		if len(listed.GetExecutions()) > 0 {
			workflowID = listed.GetExecutions()[0].GetExecution().GetWorkflowId()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NotEmpty(t, workflowID, "workflow did not start before the test deadline")
	defer func() {
		_ = c.TerminateWorkflow(context.Background(), workflowID, "", "test cleanup")
	}()
	cancel()

	select {
	case got := <-done:
		require.NoError(t, got.err)
		require.NotNil(t, got.result)
		assert.Equal(t, workflowID, got.result.WorkflowID)
		assert.Equal(t, testrunner.StatusFailed, got.result.Status)
		assert.True(t, errors.Is(got.result.Err, context.Canceled), "expected context cancellation, got %v", got.result.Err)
	case <-time.After(30 * time.Second):
		t.Fatal("test runner did not return after caller cancellation")
	}

	desc, err := c.DescribeWorkflowExecution(t.Context(), workflowID, "")
	require.NoError(t, err)
	assert.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, desc.GetWorkflowExecutionInfo().GetStatus())
}

func TestE2EZigflowTestTimeoutTerminatesContinuedRun(t *testing.T) {
	c, err := temporal.NewConnectionWithEnvvars()
	require.NoError(t, err)
	defer c.Close()

	definition, err := os.ReadFile(filepath.Join("tests", "continue-as-new", "workflow.yaml"))
	require.NoError(t, err)

	result, err := testrunner.RunBytes(t.Context(), definition, &testrunner.Config{
		Timeout: 20 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, testrunner.StatusTimeout, result.Status)
	assert.ErrorIs(t, result.Err, testrunner.ErrTestTimeout)
	assert.ErrorIs(t, result.Err, context.DeadlineExceeded)
	require.NotEmpty(t, result.WorkflowID)
	defer func() {
		_ = c.TerminateWorkflow(context.Background(), result.WorkflowID, "", "test cleanup")
	}()

	current, err := c.DescribeWorkflowExecution(t.Context(), result.WorkflowID, "")
	require.NoError(t, err)
	assert.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, current.GetWorkflowExecutionInfo().GetStatus())

	listed, err := c.ListWorkflow(t.Context(), &workflowservice.ListWorkflowExecutionsRequest{
		Query: fmt.Sprintf("WorkflowId = '%s'", result.WorkflowID),
	})
	require.NoError(t, err)
	continued := false
	for _, execution := range listed.GetExecutions() {
		if execution.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW {
			continued = true
			assert.NotEqual(t, current.GetWorkflowExecutionInfo().GetExecution().GetRunId(), execution.GetExecution().GetRunId())
		}
	}
	assert.True(t, continued, "workflow should continue as new before the timeout")
}

//go:build e2e && !windows

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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
)

func TestE2EZigflowTestSIGINTTerminatesExecution(t *testing.T) {
	c, err := temporal.NewConnectionWithEnvvars()
	require.NoError(t, err)
	defer c.Close()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "zigflow")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".")
	build.Dir = repoRoot
	buildOutput, err := build.CombinedOutput()
	require.NoError(t, err, "build CLI: %s", buildOutput)

	workflowType := "signal-test-" + uuid.NewString()
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
	workflowFile := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowFile, []byte(definition), 0o600))

	var stderr bytes.Buffer
	command := exec.Command(binary, "test", workflowFile, "--timeout", "1m")
	command.Dir = repoRoot
	command.Env = os.Environ()
	command.Stderr = &stderr
	require.NoError(t, command.Start())
	t.Cleanup(func() { _ = command.Process.Kill() })
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()

	queryCtx, stopQuery := context.WithTimeout(t.Context(), 45*time.Second)
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
	require.NotEmpty(t, workflowID, "CLI execution did not start before the test deadline")
	defer func() {
		_ = c.TerminateWorkflow(context.Background(), workflowID, "", "test cleanup")
	}()
	require.NoError(t, command.Process.Signal(os.Interrupt))

	select {
	case waitErr := <-done:
		assert.Error(t, waitErr, "an interrupted CLI test must exit unsuccessfully")
	case <-time.After(30 * time.Second):
		t.Fatal("CLI did not exit after SIGINT")
	}

	description, err := c.DescribeWorkflowExecution(t.Context(), workflowID, "")
	require.NoError(t, err)
	assert.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, description.GetWorkflowExecutionInfo().GetStatus(), stderr.String())
}

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
	"context"
	"testing"

	"github.com/mrsimonemms/golang-helpers/temporal"
	zlog "github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/client"
)

type TestCase struct {
	Name         string
	WorkflowPath string
	// ExtraFiles contains additional workflow file paths to pass to the worker
	// alongside WorkflowPath. The setup harness loads only WorkflowPath into
	// Workflow; test functions are responsible for connecting to the extra
	// workflows by namespace and name.
	ExtraFiles     []string
	Workflow       *model.Workflow
	Input          map[string]any
	ExpectedOutput any
	Test           func(t *testing.T, test *TestCase)
}

// RunToCompletionNamed executes a single workflow identified by taskQueue and
// workflowName, waits for it to finish, and asserts the result equals
// expectedOutput. It is used by multi-file test cases where a single TestCase
// drives more than one workflow and test.Workflow refers only to the primary.
func RunToCompletionNamed[T any](t *testing.T, taskQueue, workflowName string, input, expectedOutput any) {
	t.Helper()

	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&zlog.Logger),
	)
	assert.NoError(t, err)
	defer c.Close()

	wCtx := context.Background()

	we, err := c.ExecuteWorkflow(wCtx, client.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, workflowName, input)
	assert.NoError(t, err)

	var result T
	assert.NoError(t, we.Get(wCtx, &result))
	assert.Equal(t, expectedOutput, result)
}

// RunToCompletion simplest version of the test where it runs to completion and matches the output
func RunToCompletion[T any](t *testing.T, test *TestCase) {
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&zlog.Logger),
	)
	assert.NoError(t, err)
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: test.Workflow.Document.Namespace,
	}

	wCtx := context.Background()

	we, err := c.ExecuteWorkflow(wCtx, workflowOptions, test.Workflow.Document.Name, test.Input)
	assert.NoError(t, err)

	var result T
	assert.NoError(t, we.Get(wCtx, &result))
	assert.Equal(t, test.ExpectedOutput, result)
}

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

package tasks

import (
	"errors"
	"testing"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// A listener that receives no matching event within its timeout must fail
// with the Zigflow timeout error type, the same type a `raise` of
// https://zigflow.dev/spec/1.0.0/errors/timeout produces, so a catch can tell
// a timeout apart from any other failure.
func TestListenTaskBuilderAwaitTimeoutReturnsTimeoutError(t *testing.T) {
	builder := &ListenTaskBuilder{
		name: "timeoutListener",
		task: &model.ListenTask{},
	}

	var awaitErr error

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		awaitErr = builder.await(ctx, ctx, time.Second, false, new(bool), nil)
		return nil
	}, workflow.RegisterOptions{Name: "await-timeout"})

	env.ExecuteWorkflow("await-timeout")
	require.NoError(t, env.GetWorkflowError())

	require.Error(t, awaitErr)

	var zigflowErr *model.Error
	require.True(t, errors.As(awaitErr, &zigflowErr), "expected a *model.Error, got %T: %v", awaitErr, awaitErr)
	assert.Equal(t, models.ErrorTypeTimeout, zigflowErr.Type.String())
	assert.Equal(t, 408, zigflowErr.Status)
}

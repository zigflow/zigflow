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

package metadata_test

import (
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// maximumAttempts runs a workflow that applies the activity options for one task and returns the
// maximum attempts an activity would then get.
func maximumAttempts(t *testing.T, taskMetadata map[string]any) int32 {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	var attempts int32
	env.RegisterWorkflow(func(ctx workflow.Context) error {
		ctx, err := metadata.SetActivityOptions(ctx, &model.Workflow{}, &model.TaskBase{Metadata: taskMetadata}, "task")
		if err != nil {
			return err
		}
		if rp := workflow.GetActivityOptions(ctx).RetryPolicy; rp != nil {
			attempts = rp.MaximumAttempts
		}
		return nil
	})

	env.ExecuteWorkflow("func1")
	assert.NoError(t, env.GetWorkflowError())

	return attempts
}

// The default retry policy is shared by every task in the process, so a task that overrides retry
// options must not change it for the tasks that run after it.
func TestSetActivityOptionsDoesNotChangeTheDefaultRetryPolicy(t *testing.T) {
	once := map[string]any{"activityOptions": map[string]any{"retryPolicy": map[string]any{"maximumAttempts": 1}}}
	three := map[string]any{"activityOptions": map[string]any{"retryPolicy": map[string]any{"maximumAttempts": 3}}}

	defaultAttempts := maximumAttempts(t, nil)

	assert.EqualValues(t, 1, maximumAttempts(t, once))
	assert.Equal(t, defaultAttempts, maximumAttempts(t, nil), "a task with no retry options gets the default")
	assert.EqualValues(t, 3, maximumAttempts(t, three))
	assert.Equal(t, defaultAttempts, maximumAttempts(t, nil), "the default is unchanged afterwards")
}

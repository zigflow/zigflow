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
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestConvertRetryPolicy(t *testing.T) {
	tests := []struct {
		Name        string
		RetryPolicy *metadata.RetryPolicy
		Starting    *temporal.RetryPolicy
		Expected    *temporal.RetryPolicy
	}{
		{
			Name:        "Empty",
			RetryPolicy: &metadata.RetryPolicy{},
			Expected:    &temporal.RetryPolicy{},
		},
		{
			Name: "Full",
			RetryPolicy: &metadata.RetryPolicy{
				InitialInterval:        &model.Duration{Value: model.DurationInline{Seconds: 1}},
				BackoffCoefficient:     utils.Ptr(2.0),
				MaximumAttempts:        utils.Ptr[int32](3),
				MaximumInterval:        &model.Duration{Value: model.DurationInline{Seconds: 3}},
				NonRetryableErrorTypes: []string{"error1"},
			},
			Expected: &temporal.RetryPolicy{
				InitialInterval:        time.Second,
				BackoffCoefficient:     2.0,
				MaximumAttempts:        3,
				MaximumInterval:        time.Second * 3,
				NonRetryableErrorTypes: []string{"error1"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.RetryPolicy.ToTemporal(test.Starting))
		})
	}
}

// runSetActivityOptions executes SetActivityOptions inside a test workflow environment
// and returns the resulting ActivityOptions. It captures the options via a channel
// because workflow functions run in a coroutine managed by the test suite.
func runSetActivityOptions(t *testing.T, wf *model.Workflow, task *model.TaskBase) (workflow.ActivityOptions, error) {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	type result struct {
		opts workflow.ActivityOptions
		err  error
	}
	ch := make(chan result, 1)

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		newCtx, err := metadata.SetActivityOptions(ctx, wf, task, "testTask")
		if err != nil {
			ch <- result{err: err}
			return err
		}
		ch <- result{opts: workflow.GetActivityOptions(newCtx)}
		return nil
	})

	assert.NoError(t, env.GetWorkflowError())
	r := <-ch
	return r.opts, r.err
}

func TestSetActivityOptionsDefaultTimeout(t *testing.T) {
	wf := &model.Workflow{
		Document: model.Document{},
	}
	task := &model.TaskBase{}

	opts, err := runSetActivityOptions(t, wf, task)
	assert.NoError(t, err)
	assert.Equal(t, 15*time.Second, opts.StartToCloseTimeout)
}

func TestSetActivityOptionsWorkflowTimeoutOverride(t *testing.T) {
	after := &model.Duration{Value: model.DurationInline{Minutes: 2}}
	wf := &model.Workflow{
		Document: model.Document{},
		Timeout: &model.TimeoutOrReference{
			Timeout: &model.Timeout{
				After: after,
			},
		},
	}
	task := &model.TaskBase{}

	opts, err := runSetActivityOptions(t, wf, task)
	assert.NoError(t, err)
	assert.Equal(t, 2*time.Minute, opts.StartToCloseTimeout)
}

const testKeyStartToCloseTimeout = "startToCloseTimeout"

func TestSetActivityOptionsGlobalMetadataOverride(t *testing.T) {
	wf := &model.Workflow{
		Document: model.Document{
			Metadata: map[string]any{
				metadata.MetadataActivityOptions: map[string]any{
					testKeyStartToCloseTimeout: map[string]any{"minutes": 3},
				},
			},
		},
	}
	task := &model.TaskBase{}

	opts, err := runSetActivityOptions(t, wf, task)
	assert.NoError(t, err)
	assert.Equal(t, 3*time.Minute, opts.StartToCloseTimeout)
}

func TestSetActivityOptionsTaskMetadataOverridePrecedence(t *testing.T) {
	// Workflow timeout and global metadata both set, task-level must win.
	after := &model.Duration{Value: model.DurationInline{Minutes: 2}}
	wf := &model.Workflow{
		Document: model.Document{
			Metadata: map[string]any{
				metadata.MetadataActivityOptions: map[string]any{
					testKeyStartToCloseTimeout: map[string]any{"minutes": 3},
				},
			},
		},
		Timeout: &model.TimeoutOrReference{
			Timeout: &model.Timeout{
				After: after,
			},
		},
	}
	task := &model.TaskBase{
		Metadata: map[string]any{
			metadata.MetadataActivityOptions: map[string]any{
				testKeyStartToCloseTimeout: map[string]any{"seconds": 30},
			},
		},
	}

	opts, err := runSetActivityOptions(t, wf, task)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, opts.StartToCloseTimeout)
}

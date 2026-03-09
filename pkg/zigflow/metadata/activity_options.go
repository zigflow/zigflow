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

package metadata

import (
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ActivityOptions struct {
	HeartbeatTimeout       *model.Duration   `json:"heartbeatTimeout,omitempty"`
	ScheduleToCloseTimeout *model.Duration   `json:"scheduleToCloseTimeout"`
	ScheduleToStartTimeout *model.Duration   `json:"scheduleToStartTimeout"`
	StartToCloseTimeout    *model.Duration   `json:"startToCloseTimeout"`
	RetryPolicy            *RetryPolicy      `json:"retryPolicy"`
	DisableEagerExecution  *bool             `json:"disableEagerExecution"`
	Summary                string            `json:"summary"`
	Priority               *ActivityPriority `json:"priority"`
}

func (a *ActivityOptions) ToTemporal(opts *workflow.ActivityOptions) workflow.ActivityOptions {
	if a.HeartbeatTimeout != nil {
		opts.HeartbeatTimeout = utils.ToDuration(a.HeartbeatTimeout)
	}

	if a.ScheduleToCloseTimeout != nil {
		opts.ScheduleToCloseTimeout = utils.ToDuration(a.ScheduleToCloseTimeout)
	}

	if a.ScheduleToStartTimeout != nil {
		opts.ScheduleToStartTimeout = utils.ToDuration(a.ScheduleToStartTimeout)
	}

	if a.StartToCloseTimeout != nil {
		opts.StartToCloseTimeout = utils.ToDuration(a.StartToCloseTimeout)
	}

	if a.RetryPolicy != nil {
		opts.RetryPolicy = a.RetryPolicy.ToTemporal(opts.RetryPolicy)
	}

	if a.DisableEagerExecution != nil {
		opts.DisableEagerExecution = *a.DisableEagerExecution
	}

	if a.Summary != "" {
		opts.Summary = a.Summary
	}

	if a.Priority != nil {
		opts.Priority = a.Priority.ToTemporal(opts.Priority)
	}

	return *opts
}

type ActivityPriority struct {
	PriorityKey    *int     `json:"priorityKey"`
	FairnessKey    string   `json:"fairnessKey"`
	FairnessWeight *float32 `json:"fairnessWeight"`
}

func (a *ActivityPriority) ToTemporal(priority temporal.Priority) temporal.Priority {
	if a.PriorityKey != nil {
		priority.PriorityKey = *a.PriorityKey
	}
	if a.FairnessKey != "" {
		priority.FairnessKey = a.FairnessKey
	}
	if a.FairnessWeight != nil {
		priority.FairnessWeight = *a.FairnessWeight
	}
	return priority
}

type RetryPolicy struct {
	InitialInterval        *model.Duration `json:"initialInterval"`
	BackoffCoefficient     *float64        `json:"backoffCoefficient"`
	MaximumInterval        *model.Duration `json:"maximumInterval"`
	MaximumAttempts        *int32          `json:"maximumAttempts"`
	NonRetryableErrorTypes []string        `json:"nonRetryableErrorTypes"`
}

func (r *RetryPolicy) ToTemporal(retry *temporal.RetryPolicy) *temporal.RetryPolicy {
	if retry == nil {
		retry = &temporal.RetryPolicy{}
	}

	if r.InitialInterval != nil {
		retry.InitialInterval = utils.ToDuration(r.InitialInterval)
	}
	if r.BackoffCoefficient != nil {
		retry.BackoffCoefficient = *r.BackoffCoefficient
	}
	if r.MaximumInterval != nil {
		retry.MaximumInterval = utils.ToDuration(r.MaximumInterval)
	}
	if r.MaximumAttempts != nil {
		retry.MaximumAttempts = *r.MaximumAttempts
	}
	if len(r.NonRetryableErrorTypes) > 0 {
		retry.NonRetryableErrorTypes = r.NonRetryableErrorTypes
	}

	return retry
}

// ************** //
// Static Methods //
// ************** //
func SetActivityOptions(ctx workflow.Context, wf *model.Workflow, task *model.TaskBase, taskName string) (workflow.Context, error) {
	logger := workflow.GetLogger(ctx)

	// Get any options already set
	ao := workflow.GetActivityOptions(ctx)

	// Set default values
	ao.Summary = taskName
	ao.RetryPolicy = defaultRetryPolicy
	ao.StartToCloseTimeout = defaultWorkflowTimeout

	// Convert the timeout
	if wf.Timeout != nil && wf.Timeout.Timeout != nil && wf.Timeout.Timeout.After != nil {
		ao.StartToCloseTimeout = utils.ToDuration(wf.Timeout.Timeout.After)
	}

	// Override any global activity options
	if a, ok := wf.Document.Metadata[MetadataActivityOptions]; ok {
		var opts ActivityOptions
		if err := utils.ToType(a, &opts); err != nil {
			return nil, fmt.Errorf("error decoding global activity options metadata: %w", err)
		}

		logger.Debug("Adding global activity options", "options", opts)
		ao = opts.ToTemporal(&ao)
	}

	// Override any task-specific activity options
	if a, ok := task.Metadata[MetadataActivityOptions]; ok {
		var opts ActivityOptions
		if err := utils.ToType(a, &opts); err != nil {
			return nil, fmt.Errorf("error decoding task activity options metadata: %w", err)
		}

		logger.Debug("Adding task activity options", "options", opts)
		ao = opts.ToTemporal(&ao)
	}

	logger.Debug("Setting activity options", "options", ao)

	// Create the new context with the options set
	return workflow.WithActivityOptions(ctx, ao), nil
}

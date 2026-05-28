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
	"encoding/json"
	"fmt"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const keyUntil = "until"

// NewWaitExtTaskBuilder constructs the builder for the Zigflow extended wait
// task, which the SDK has produced from a __zigflow_ext_wait task body. The
// builder evaluates any runtime expressions in the body at workflow execution
// time, then sleeps either until an absolute moment (until form) or for a
// computed duration (expression-bearing duration form).
func NewWaitExtTaskBuilder(
	temporalWorker worker.Worker,
	task *models.WaitExtTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*WaitExtTaskBuilder, error) {
	return &WaitExtTaskBuilder{
		builder: builder[*models.WaitExtTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type WaitExtTaskBuilder struct {
	builder[*models.WaitExtTask]
}

func (t *WaitExtTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, _ any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		// Clone the typed body to a plain map by round-tripping through
		// JSON. This avoids mutating the shared task definition.
		raw, err := json.Marshal(t.task.Wait)
		if err != nil {
			return nil, fmt.Errorf("error marshalling wait extension body: %w", err)
		}
		var cloned map[string]any
		if err := json.Unmarshal(raw, &cloned); err != nil {
			return nil, fmt.Errorf("error unmarshalling wait extension body: %w", err)
		}

		// Evaluate any runtime expressions against the workflow state.
		// No SideEffect wrapper: $data, $input, $context and $env are
		// already in workflow history and therefore deterministic.
		// Non-deterministic expressions are rejected up front by the
		// central workflow determinism pass (ValidateWorkflowDeterminism),
		// so anything reaching here is safe to evaluate directly.
		evaluated, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(cloned), nil, state)
		if err != nil {
			return nil, fmt.Errorf("error evaluating wait extension expressions: %w", err)
		}
		resolved, ok := evaluated.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("wait extension body must resolve to a map, got %T", evaluated)
		}

		if untilRaw, hasUntil := resolved[keyUntil]; hasUntil {
			untilStr, ok := untilRaw.(string)
			if !ok {
				return nil, fmt.Errorf("wait.until must resolve to a string, got %T", untilRaw)
			}
			return nil, t.sleepUntil(ctx, untilStr)
		}

		dur, err := utils.DurationFromMap(resolved)
		if err != nil {
			return nil, fmt.Errorf("error computing wait duration: %w", err)
		}

		logger.Debug("Sleeping", "duration", dur.String())
		if err := workflow.Sleep(ctx, dur); err != nil {
			if temporal.IsCanceledError(err) {
				return nil, nil
			}
			logger.Error("Error creating sleep instruction", "error", err)
			return nil, fmt.Errorf("error creating sleep: %w", err)
		}
		return nil, nil
	}, nil
}

func (t *WaitExtTaskBuilder) Validate() error {
	if t.task.Wait == nil {
		return fmt.Errorf("wait extension task %q has no body", t.GetTaskName())
	}
	until := t.task.Wait.Until
	if until == "" || model.IsStrictExpr(until) {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, until); err != nil {
		return fmt.Errorf("wait.until %q is not a valid RFC 3339 timestamp: %w", until, err)
	}
	return nil
}

// sleepUntil parses the resolved until value as RFC 3339, computes the delta
// against the deterministic workflow clock, and sleeps the timer. A past or
// zero delta is a no-op: the workflow continues immediately.
func (t *WaitExtTaskBuilder) sleepUntil(ctx workflow.Context, untilStr string) error {
	logger := workflow.GetLogger(ctx)

	untilTime, err := time.Parse(time.RFC3339, untilStr)
	if err != nil {
		return fmt.Errorf("wait.until %q is not a valid RFC 3339 timestamp: %w", untilStr, err)
	}

	delta := untilTime.Sub(workflow.Now(ctx))
	if delta <= 0 {
		logger.Debug("wait.until is in the past, continuing immediately", keyUntil, untilStr)
		return nil
	}

	logger.Debug("Sleeping until", keyUntil, untilStr, "duration", delta.String())
	if err := workflow.Sleep(ctx, delta); err != nil {
		if temporal.IsCanceledError(err) {
			return nil
		}
		logger.Error("Error creating sleep instruction", "error", err)
		return fmt.Errorf("error creating sleep: %w", err)
	}
	return nil
}

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
	"fmt"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewSetTaskBuilder(
	temporalWorker worker.Worker,
	task *model.SetTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*SetTaskBuilder, error) {
	return &SetTaskBuilder{
		doc:            doc,
		eventEmitter:   emitter,
		name:           taskName,
		task:           task,
		taskOpts:       taskOpts,
		temporalWorker: temporalWorker,
	}, nil
}

type SetTaskBuilder struct {
	builder[*model.SetTask]
}

func (t *SetTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (output any, err error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Parsing set data")

		result, err := t.parseAsSideEffect(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("error parsing set object :%w", err)
		}

		// A whole-field runtime expression can evaluate to a non-object, so
		// guard against a panic and return a clear workflow error instead.
		data, ok := result.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("set must evaluate to an object, got %T", result)
		}

		// Add the result to the state's data
		logger.Debug("Setting data to the state")
		state.AddData(data)

		return result, nil
	}, nil
}

// parseAsSideEffect parses the whole set object inside a single side effect,
// preventing Go from randomising the map's order
func (t *SetTaskBuilder) parseAsSideEffect(ctx workflow.Context, state *utils.State) (any, error) {
	type response struct {
		Err    string
		Result any
	}
	var val response

	err := workflow.SideEffect(ctx, func(ctx workflow.Context) any {
		result, err := utils.TraverseAndEvaluateObj(
			t.task.Set,
			nil,
			state,
			// Evaluation already runs inside this SideEffect, so pass a
			// pass-through wrapper to mark the expressions as replay-safe and
			// suppress the non-deterministic expression warning.
			func(fn func() (any, error)) (any, error) {
				return fn()
			},
		)
		if err != nil {
			return response{
				Err: fmt.Errorf("error parsing set object: %w", err).Error(),
			}
		}

		return response{
			Result: result,
		}
	}).Get(&val)
	if err != nil {
		return nil, fmt.Errorf("error running side effect: %w", err)
	}
	if val.Err != "" {
		return nil, fmt.Errorf("error running runtime expression: %s", val.Err)
	}

	return val.Result, err
}

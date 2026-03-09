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

	"github.com/rs/zerolog/log"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewCallActivityTaskBuilder(
	temporalWorker worker.Worker,
	task *model.CallFunction,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (*CallActivityTaskBuilder, error) {
	if task.Call != customCallFunctionActivity {
		return nil, fmt.Errorf("unsupported call task '%s' for activity builder", task.Call)
	}

	return &CallActivityTaskBuilder{
		builder: builder[*model.CallFunction]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type CallActivityTaskBuilder struct {
	builder[*model.CallFunction]

	// Store the parsed activity data
	activity *models.ActivityCallWith
}

func (t *CallActivityTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	log.Debug().Str("task", t.GetTaskName()).Msg("Converting call activity data")
	if err := t.convertToType(); err != nil {
		log.Error().Err(err).Msg("Error building call activity data")
		return nil, err
	}

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		args, err := t.parseArgs(state)
		if err != nil {
			logger.Error("Error parsing call activity arguments", "error", err)
			return nil, err
		}

		// Set the task queue
		opts := workflow.GetActivityOptions(ctx)
		opts.TaskQueue = t.activity.TaskQueue
		ctx = workflow.WithActivityOptions(ctx, opts)

		logger.Info("Executing Temporal activity", "activity", t.activity.Name, "task", t.GetTaskName())

		future := workflow.ExecuteActivity(ctx, t.activity.Name, args...)

		var res any
		if err := future.Get(ctx, &res); err != nil {
			if temporal.IsCanceledError(err) {
				logger.Debug("Activity cancelled", "activity", t.activity.Name)
				return nil, nil
			}
			logger.Error("Error executing activity", "activity", t.activity.Name, "error", err)
			return nil, fmt.Errorf("error executing activity %s: %w", t.activity.Name, err)
		}

		state.AddData(map[string]any{
			t.GetTaskName(): res,
		})

		return res, nil
	}, nil
}

func (t *CallActivityTaskBuilder) convertToType() error {
	payload, err := json.Marshal(t.task.With)
	if err != nil {
		return fmt.Errorf("error marshalling activity call arguments: %w", err)
	}

	var result models.ActivityCallWith
	if err := json.Unmarshal(payload, &result); err != nil {
		return fmt.Errorf("error unmarshalling activity call arguments: %w", err)
	}

	if result.Name == "" {
		return fmt.Errorf("call activity requires a name: %s", t.GetTaskName())
	}

	if result.TaskQueue == "" {
		return fmt.Errorf("activity task queue must be set: %s", t.GetTaskName())
	}

	t.activity = &result

	return nil
}

func (t *CallActivityTaskBuilder) parseArgs(state *utils.State) ([]any, error) {
	parsedArgs, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		"args": swUtil.DeepCloneValue(t.activity.Arguments),
	}), nil, state)
	if err != nil {
		return nil, err
	}

	// Return the parsed arguments
	return parsedArgs.(map[string]any)["args"].([]any), nil
}

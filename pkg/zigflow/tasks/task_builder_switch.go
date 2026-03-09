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

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewSwitchTaskBuilder(
	temporalWorker worker.Worker,
	task *model.SwitchTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (*SwitchTaskBuilder, error) {
	return &SwitchTaskBuilder{
		builder: builder[*model.SwitchTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type SwitchTaskBuilder struct {
	builder[*model.SwitchTask]
}

func (t *SwitchTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	hasDefault := false
	for i, switchItem := range t.task.Switch {
		for name, item := range switchItem {
			if item.When == nil {
				if hasDefault {
					return nil, fmt.Errorf("multiple switch statements without when: %s.%d.%s", t.GetTaskName(), i, name)
				}
				hasDefault = true
			}
		}
	}
	if !hasDefault {
		log.Warn().Str("task", t.GetTaskName()).Msg("No default switch task detected")
	}

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		for _, switchItem := range t.task.Switch {
			for name, item := range switchItem {
				logger.Debug("Checking if we should run this switch statement", "task", t.GetTaskName(), "condition", name)

				if shouldRun, err := utils.CheckIfStatement(item.When, state); err != nil {
					return nil, err
				} else if !shouldRun {
					logger.Debug("Skipping switch statement task", "task", t.GetTaskName(), "condition", name)
					continue
				}

				then := item.Then
				if then == nil || then.IsTermination() {
					logger.Debug("Skipping task as then is termination or not set")
					return nil, nil
				}

				logger.Info("Executing switch statement's task as a child workflow", "task", t.GetTaskName(), "condition", name)
				var res any
				if err := workflow.ExecuteChildWorkflow(ctx, then.Value, input, state).Get(ctx, &res); err != nil {
					logger.Error("Error executing child switch workflow", "task", t.GetTaskName(), "condition", name)
					return nil, err
				}

				// Stop it executing anything else
				return res, nil
			}
		}

		return nil, nil
	}, nil
}

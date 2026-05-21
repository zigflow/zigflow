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
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewSwitchTaskBuilder(
	temporalWorker worker.Worker,
	task *model.SwitchTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*SwitchTaskBuilder, error) {
	return &SwitchTaskBuilder{
		builder: builder[*model.SwitchTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
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
				if then == nil {
					// No flow directive for this case. Allow the enclosing
					// scope to continue executing the next task normally.
					logger.Debug("Matching switch case has no then; continuing", "task", t.GetTaskName(), "condition", name)
					return nil, nil
				}

				logger.Info("Switch case matched; emitting flow directive",
					"task", t.GetTaskName(), "condition", name, "directive", then.Value)
				return nil, flow.FromDirective(then)
			}
		}

		return nil, nil
	}, nil
}

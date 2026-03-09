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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewWaitTaskBuilder(
	temporalWorker worker.Worker,
	task *model.WaitTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (*WaitTaskBuilder, error) {
	return &WaitTaskBuilder{
		builder: builder[*model.WaitTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type WaitTaskBuilder struct {
	builder[*model.WaitTask]
}

func (t *WaitTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, _ any, _ *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		duration := utils.ToDuration(t.task.Wait)

		logger.Debug("Sleeping", "duration", duration.String())

		if err := workflow.Sleep(ctx, duration); err != nil {
			if temporal.IsCanceledError(err) {
				return nil, nil
			}

			logger.Error("Error creating sleep instruction", "error", err)
			return nil, fmt.Errorf("error creating sleep: %w", err)
		}

		return nil, nil
	}, nil
}

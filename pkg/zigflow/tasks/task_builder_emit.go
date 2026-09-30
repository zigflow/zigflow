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
	"strings"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewEmitTaskBuilder(
	temporalWorker worker.Worker,
	task *model.EmitTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*EmitTaskBuilder, error) {
	return &EmitTaskBuilder{
		doc:            doc,
		eventEmitter:   emitter,
		name:           taskName,
		task:           task,
		taskOpts:       taskOpts,
		temporalWorker: temporalWorker,
	}, nil
}

type EmitTaskBuilder struct {
	builder[*model.EmitTask]
}

func (t *EmitTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	eventType := t.task.Emit.Event.With.Type

	const prefix = "io.temporal.streams."
	topicName, ok := strings.CutPrefix(eventType, prefix)
	if !ok {
		return nil, fmt.Errorf("unsupported event type %q", eventType)
	}
	if topicName == "" {
		return nil, fmt.Errorf("temporal stream topic must not be empty")
	}

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		stream, err := state.GetStream(ctx)
		if err != nil {
			return nil, fmt.Errorf("error retrieving stream instance: %w", err)
		}

		logger.Debug("Emitting data via Temporal Workflow Streams", "topic", topicName)
		topic := stream.Topic(topicName)

		data := t.task.Emit.Event.With.Additional["data"]

		if err := topic.Publish(data); err != nil {
			return nil, err
		}

		return nil, nil
	}, nil
}

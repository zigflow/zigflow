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
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewCallHTTPTaskBuilder(
	temporalWorker worker.Worker,
	task *model.CallHTTP,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (*CallHTTPTaskBuilder, error) {
	return &CallHTTPTaskBuilder{
		builder: builder[*model.CallHTTP]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type CallHTTPTaskBuilder struct {
	builder[*model.CallHTTP]
}

func (t *CallHTTPTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		return t.executeActivity(ctx, (*activities.CallHTTP).CallHTTPActivity, input, state)
	}, nil
}

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
	taskOpts *TaskOpts,
) (*CallHTTPTaskBuilder, error) {
	return &CallHTTPTaskBuilder{
		builder: builder[*model.CallHTTP]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type CallHTTPTaskBuilder struct {
	builder[*model.CallHTTP]
}

// Per-task activity name "<workflowType>.<taskName>" so SDK metrics
// carry a distinct activity_type label per HTTP task. Falls back to the
// bare task name if the workflow has no name.
func (t *CallHTTPTaskBuilder) callHTTPActivityName() string {
	taskName := t.GetTaskName()
	if t.doc == nil || t.doc.Document.Name == "" {
		return taskName
	}
	return t.doc.Document.Name + "." + taskName
}

// callHTTPActivity is a package-level singleton whose method value is
// registered under per-task names. A bound method value is required so
// Temporal invokes the activity with the receiver supplied; an unbound
// method expression would pass the activity's first argument as the
// receiver, causing a nil pointer dereference at runtime.
var callHTTPActivity = &activities.CallHTTP{}

func (t *CallHTTPTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	activityName := t.callHTTPActivityName()
	registerActivityOnce(t.temporalWorker, callHTTPActivity.CallHTTPActivity, activityName)

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		return t.executeActivity(ctx, activityName, input, state)
	}, nil
}

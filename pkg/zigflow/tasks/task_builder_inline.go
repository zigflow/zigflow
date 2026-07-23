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
	"go.temporal.io/sdk/worker"
)

// newInlineDoBuilder constructs a DoTaskBuilder for a nested task list that
// executes inline within the current workflow rather than as a Temporal child
// workflow. Registration is disabled so no child workflow is created.
//
// The supplied task path is threaded onto the builder: direct construction
// bypasses NewTaskBuilder's path setter, so without this the nested tasks would
// register under collision-prone bare names when sibling scopes reuse a leaf
// name. Both TryTaskBuilder and ForTaskBuilder rely on this.
func newInlineDoBuilder(
	temporalWorker worker.Worker,
	taskList *model.TaskList,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
	taskPath []string,
) (*DoTaskBuilder, error) {
	builder, err := NewDoTaskBuilder(
		temporalWorker,
		&model.DoTask{Do: taskList},
		taskName,
		doc,
		emitter,
		taskOpts,
		DoTaskOpts{
			// The body is not a standalone Temporal workflow ...
			DisableRegisterWorkflow: true,
			// ... and it runs inline in the caller, so control-flow directives
			// (flow.ErrEnd) must propagate straight back to the try/for builder.
			InlineExecution: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating inline do task builder: %w", err)
	}

	builder.setTaskPath(taskPath)

	return builder, nil
}

// buildInlineTaskList builds a nested task list into a TemporalWorkflowFunc that
// runs inline in the current workflow. It is the shared path the try and for
// builders use to turn their bodies into ordinary functions executed directly,
// without registering a child workflow.
func buildInlineTaskList(
	temporalWorker worker.Worker,
	taskList *model.TaskList,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
	taskPath []string,
) (TemporalWorkflowFunc, error) {
	builder, err := newInlineDoBuilder(temporalWorker, taskList, taskName, doc, emitter, taskOpts, taskPath)
	if err != nil {
		return nil, err
	}

	fn, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("error building inline task list: %w", err)
	}

	return fn, nil
}

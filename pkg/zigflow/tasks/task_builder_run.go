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
	"slices"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewRunTaskBuilder(
	temporalWorker worker.Worker,
	task *model.RunTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (*RunTaskBuilder, error) {
	return &RunTaskBuilder{
		builder: builder[*model.RunTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type RunTaskBuilder struct {
	builder[*model.RunTask]
}

func (t *RunTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	if t.task.Run.Await == nil {
		// Default to true
		t.task.Run.Await = utils.Ptr(true)
	}

	var factory TemporalWorkflowFunc
	if t.task.Run.Container != nil {
		if t.task.Run.Container.Lifetime == nil {
			t.task.Run.Container.Lifetime = &model.ContainerLifetime{
				Cleanup: "always",
			}
		}

		if len(t.task.Run.Container.Ports) > 0 {
			return nil, fmt.Errorf("ports are not allowed on containers")
		}

		factory = t.runContainer
	} else if s := t.task.Run.Script; s != nil {
		if !slices.Contains([]string{"js", "python"}, s.Language) {
			return nil, fmt.Errorf("unknown script language '%s' for task: %s", s.Language, t.GetTaskName())
		}
		if !*t.task.Run.Await {
			return nil, fmt.Errorf("run scripts must be run with await: %s", t.GetTaskName())
		}
		if s.InlineCode == nil || *s.InlineCode == "" {
			return nil, fmt.Errorf("run script has no code defined: %s", t.GetTaskName())
		}
		factory = t.runScript
	} else if t.task.Run.Shell != nil {
		factory = t.runShell
	} else if t.task.Run.Workflow != nil {
		factory = t.runWorkflow
	} else {
		return nil, fmt.Errorf("unsupported run task: %s", t.GetTaskName())
	}

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Run await status", "await", *t.task.Run.Await, "task", t.GetTaskName())

		res, err := factory(ctx, input, state)
		if err != nil {
			return nil, err
		}

		// Add the result to the state's data
		logger.Debug("Setting data to the state", "key", t.name)
		state.AddData(map[string]any{
			t.name: res,
		})

		return res, nil
	}, nil
}

func (t *RunTaskBuilder) executeCommand(ctx workflow.Context, activityFn, input any, state *utils.State) (any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Executing a command", "task", t.GetTaskName())

	var res any
	if err := workflow.ExecuteActivity(ctx, activityFn, t.task, input, state).Get(ctx, &res); err != nil {
		if temporal.IsCanceledError(err) {
			return nil, nil
		}

		logger.Error("Error calling executing command task", "name", t.name, "error", err)
		return nil, fmt.Errorf("error calling executing command task: %w", err)
	}

	return res, nil
}

func (t *RunTaskBuilder) runContainer(ctx workflow.Context, input any, state *utils.State) (any, error) {
	return t.executeCommand(ctx, (*activities.Run).CallContainerActivity, input, state)
}

func (t *RunTaskBuilder) runScript(ctx workflow.Context, input any, state *utils.State) (any, error) {
	return t.executeCommand(ctx, (*activities.Run).CallScriptActivity, input, state)
}

func (t *RunTaskBuilder) runShell(ctx workflow.Context, input any, state *utils.State) (any, error) {
	return t.executeCommand(ctx, (*activities.Run).CallShellActivity, input, state)
}

func (t *RunTaskBuilder) runWorkflow(ctx workflow.Context, input any, state *utils.State) (any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Running a child workflow", "task", t.GetTaskName())

	await := *t.task.Run.Await

	opts := workflow.ChildWorkflowOptions{}
	if !await {
		opts.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_ABANDON
	}

	ctx = workflow.WithChildOptions(ctx, opts)

	future := workflow.ExecuteChildWorkflow(ctx, t.task.Run.Workflow.Name, input, state)

	if !await {
		logger.Warn("Not waiting for child workspace response", "task", t.GetTaskName())
		return nil, nil
	}

	var res any
	if err := future.Get(ctx, &res); err != nil {
		logger.Error("Error executiing child workflow", "error", err)
		return nil, fmt.Errorf("error executiing child workflow: %w", err)
	}
	logger.Debug("Child workflow completed", "task", t.GetTaskName())

	return res, nil
}

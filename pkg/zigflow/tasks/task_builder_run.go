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
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type RunTaskOpts struct {
	Namespace      string
	Runtime        activities.ContainerRuntime
	ServiceAccount string
}

func NewRunTaskBuilder(
	temporalWorker worker.Worker,
	task *model.RunTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*RunTaskBuilder, error) {
	return &RunTaskBuilder{
		builder: builder[*model.RunTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type RunTaskBuilder struct {
	builder[*model.RunTask]
}

func (t *RunTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	var factory TemporalWorkflowFunc
	switch {
	case t.task.Run.Container != nil:
		factory = t.runContainer
	case t.task.Run.Script != nil:
		factory = t.runScript
	case t.task.Run.Shell != nil:
		factory = t.runShell
	case t.task.Run.Workflow != nil:
		factory = t.runWorkflow
	default:
		return nil, fmt.Errorf("unsupported run task: %s", t.GetTaskName())
	}

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Run await status", "await", *t.task.Run.Await, "task", t.GetTaskName())

		res, err := factory(ctx, input, state)
		if err != nil {
			// Flow-control directives (notably an `end` propagated from
			// a `run.workflow` child) carry a meaningful payload that
			// the do-task pipeline needs to see. Preserve res so the
			// directive can be dispatched with the right output.
			if flow.IsControlError(err) {
				logger.Debug("Run task signalled a flow directive; preserving carried output",
					"task", t.GetTaskName(), "directive", err.Error())
				state.AddData(map[string]any{
					t.name: res,
				})
				return res, err
			}
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

func (t *RunTaskBuilder) Validate() error {
	if t.task.Run.Container != nil && len(t.task.Run.Container.Ports) > 0 {
		return fmt.Errorf("ports are not allowed on containers")
	}
	if s := t.task.Run.Script; s != nil {
		if !slices.Contains([]string{"js", constScriptLanguagePython}, s.Language) {
			return fmt.Errorf("unknown script language '%s' for task: %s", s.Language, t.GetTaskName())
		}
		if !*t.task.Run.Await {
			return fmt.Errorf("run scripts must be run with await: %s", t.GetTaskName())
		}
		if (s.InlineCode == nil || *s.InlineCode == "") && s.External == nil {
			return fmt.Errorf("run script has no inline or external code defined: %s", t.GetTaskName())
		}
		if s.InlineCode != nil && *s.InlineCode != "" && s.External != nil {
			return fmt.Errorf("run script must not set both inline code and external source: %s", t.GetTaskName())
		}
		if s.External != nil && s.External.Endpoint == nil {
			return fmt.Errorf("run script external source has no endpoint: %s", t.GetTaskName())
		}
	}
	return nil
}

func (t *RunTaskBuilder) PostLoad() error {
	// Default await to true: the returned closure and script validation in Build()
	// both dereference this field, so it must be non-nil before Build() runs.
	if t.task.Run.Await == nil {
		t.task.Run.Await = utils.Ptr(true)
	}

	// Default container lifetime: the container activity treats nil Lifetime the
	// same as Cleanup:"always", but setting it here makes the intent explicit.
	if t.task.Run.Container != nil && t.task.Run.Container.Lifetime == nil {
		t.task.Run.Container.Lifetime = &model.ContainerLifetime{Cleanup: "always"}
	}

	// Default run.workflow fields.
	if w := t.task.Run.Workflow; w != nil {
		if w.Namespace == "" {
			w.Namespace = constDefaultNamespace
		}
		if w.Version == "" {
			w.Version = "0.0.1"
		}
	}

	return nil
}

func (t *RunTaskBuilder) executeCommand(ctx workflow.Context, activityFn, input any, state *utils.State, additional ...any) (any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Executing a command", "task", t.GetTaskName())

	resolvedTask, err := evaluateTaskForActivity(t.task, state.Clone().AddWorkflowInfo(ctx))
	if err != nil {
		logger.Error("Error evaluating run task expressions", "task", t.GetTaskName(), "error", err)
		return nil, fmt.Errorf("error evaluating run task expressions: %w", err)
	}

	args := append([]any{
		resolvedTask, input, state,
	}, additional...)

	var res any
	if err := workflow.ExecuteActivity(ctx, activityFn, args...).Get(ctx, &res); err != nil {
		if temporal.IsCanceledError(err) {
			return nil, nil
		}

		logger.Error("Error calling executing command task", "name", t.name, "error", err)
		return nil, fmt.Errorf("error calling executing command task: %w", err)
	}

	return res, nil
}

func (t *RunTaskBuilder) runContainer(ctx workflow.Context, input any, state *utils.State) (any, error) {
	var namespace, serviceAccount string
	var runtime activities.ContainerRuntime

	if t.taskOpts != nil && t.taskOpts.Run != nil {
		namespace = t.taskOpts.Run.Namespace
		runtime = t.taskOpts.Run.Runtime
		serviceAccount = t.taskOpts.Run.ServiceAccount
	}

	return t.executeCommand(ctx, (*activities.Run).CallContainerActivity, input, state, namespace, runtime, serviceAccount)
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
		// A `then: end` directive inside the child workflow crosses the
		// child workflow boundary as a typed Temporal ApplicationError.
		// Treat it as a deliberate workflow termination rather than a
		// child-workflow failure: surface it as flow.ErrEnd and carry the
		// child's effective output upward so the surrounding do-task
		// pipeline can keep propagating end with the right payload.
		if payload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
			logger.Info("Run child workflow signalled end; propagating",
				"task", t.GetTaskName(), "workflow", t.task.Run.Workflow.Name, "carriedOutput", payload.Output)
			return payload.Output, flow.ErrEnd
		}

		logger.Error("Error executiing child workflow", "error", err)
		return nil, fmt.Errorf("error executiing child workflow: %w", err)
	}
	logger.Debug("Child workflow completed", "task", t.GetTaskName())

	return res, nil
}

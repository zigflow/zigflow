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
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewTryTaskBuilder(
	temporalWorker worker.Worker,
	task *model.TryTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*TryTaskBuilder, error) {
	return &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type TryTaskBuilder struct {
	builder[*model.TryTask]

	tryChildWorkflowName   string
	catchChildWorkflowName string
}

func (t *TryTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	for taskType, list := range t.getTasks() {
		name, builder, err := t.createBuilder(taskType, list)
		if err != nil {
			return nil, fmt.Errorf("erroring registering %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}

		if _, err = builder.Build(); err != nil {
			log.Error().Str("task", t.GetTaskName()).Str("taskType", taskType).Msg("Error building for workflow")
			return nil, fmt.Errorf("error building for workflow: %w", err)
		}

		if taskType == "try" {
			t.tryChildWorkflowName = name
		} else {
			t.catchChildWorkflowName = name
		}
	}

	return t.exec()
}

func (t *TryTaskBuilder) PostLoad() error {
	for taskType, list := range t.getTasks() {
		_, builder, err := t.createBuilder(taskType, list)
		if err != nil {
			return fmt.Errorf("erroring registering %s post load tasks for %s: %w", taskType, t.GetTaskName(), err)
		}

		if err = builder.PostLoad(); err != nil {
			log.Error().Str("task", t.GetTaskName()).Str("taskType", taskType).Msg("Error building for workflow")
			return fmt.Errorf("error building for post load workflow: %w", err)
		}
	}

	return nil
}

func (t *TryTaskBuilder) Validate() error {
	for taskType, list := range t.getTasks() {
		_, builder, err := t.createBuilder(taskType, list)
		if err != nil {
			return fmt.Errorf("error registering %s validate tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
		if err := builder.Validate(); err != nil {
			return fmt.Errorf("error validating %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
	}
	return nil
}

func (t *TryTaskBuilder) buildCatchError(err error) map[string]any {
	out := map[string]any{}

	if childErr, ok := errors.AsType[*temporal.ChildWorkflowExecutionError](err); ok {
		out["childWorkflow"] = map[string]any{
			"workflowType":     childErr.WorkflowType(),
			"workflowID":       childErr.WorkflowID(),
			"runID":            childErr.RunID(),
			"initiatedEventID": childErr.InitiatedEventID(),
			"startedEventID":   childErr.StartedEventID(),
		}
	}

	if actErr, ok := errors.AsType[*temporal.ActivityError](err); ok {
		retryStateName := actErr.RetryState().String()
		out["activity"] = map[string]any{
			"type":             actErr.ActivityType().Name,
			"activityID":       actErr.ActivityID(),
			"identity":         actErr.Identity(),
			"scheduledEventID": actErr.ScheduledEventID(),
			"startedEventID":   actErr.StartedEventID(),
			"retryState":       retryStateName,
		}
	}

	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok {
		out["type"] = appErr.Type()
		out["message"] = appErr.Message() // cleaner than Error()
		out["nonRetryable"] = appErr.NonRetryable()

		if d := appErr.NextRetryDelay(); d > 0 {
			out["nextRetryDelay"] = d.String()
		}

		if cat := appErr.Category(); cat != temporal.ApplicationErrorCategoryUnspecified {
			switch cat {
			case temporal.ApplicationErrorCategoryBenign:
				out["category"] = "benign"
			default:
				out["category"] = "unknown"
			}
		}

		// Unwrap one level to get the immediate cause message
		if cause := errors.Unwrap(appErr); cause != nil {
			out["cause"] = cause.Error()
		}

		if appErr.HasDetails() {
			var details any
			if derr := appErr.Details(&details); derr == nil {
				out["details"] = details
			}
		}
	}

	if timeoutErr, ok := errors.AsType[*temporal.TimeoutError](err); ok {
		out["errorKind"] = "timeout"
		out["timeoutType"] = timeoutErr.TimeoutType().String()
	}

	if panicErr, ok := errors.AsType[*temporal.PanicError](err); ok {
		out["errorKind"] = "panic"
		out["stackTrace"] = panicErr.StackTrace()
	}

	if _, ok := errors.AsType[*temporal.CanceledError](err); ok {
		out["errorKind"] = "canceled"
	}

	return out
}

func (t *TryTaskBuilder) exec() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (output any, err error) {
		logger := workflow.GetLogger(ctx)

		opts := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s_try", workflow.GetInfo(ctx).WorkflowExecution.ID),
		}
		childCtx := workflow.WithChildOptions(ctx, opts)

		var res map[string]any
		if err := workflow.ExecuteChildWorkflow(childCtx, t.tryChildWorkflowName, state.Input, state).Get(ctx, &res); err != nil {
			// A `then: end` directive inside the try body crosses the
			// child workflow boundary as a typed Temporal ApplicationError.
			// That is a deliberate workflow termination, not a failure to
			// be caught: surface it as flow.ErrEnd so the do-task pipeline
			// can keep propagating end upward, and preserve the carried
			// output from the child so the root completion reflects it.
			// Crucially, this skips the catch handler.
			if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
				logger.Info("Try child workflow signalled end; propagating without running catch",
					"tryWorkflow", t.tryChildWorkflowName, "carriedOutput", endPayload.Output)
				return endPayload.Output, flow.ErrEnd
			}

			logger.Warn("Workflow failed, catching the error", "tryWorkflow", t.tryChildWorkflowName, "catchWorkflow", t.catchChildWorkflowName)

			// Expose the caught error to the catch tasks so they can
			// interrogate it. The Serverless Workflow spec names this
			// variable via `catch.as`, defaulting to "error", and we expose
			// it under $data so it reads as `$data.error` (or `$data.<as>`).
			//
			// Clone the state first so the injected error only lives on the
			// catch state and never leaks back into the parent state visible
			// to later tasks. Zigflow's explicit state propagation model means
			// the error is only carried forward if the catch tasks output it.
			errKey := "error"
			if as := t.task.Catch.As; as != "" {
				errKey = as
			}
			catchState := state.Clone()
			catchState.AddData(map[string]any{
				errKey: t.buildCatchError(err),
			})

			// The try workflow has failed - let's run the catch workflow
			opts := workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("%s_catch", workflow.GetInfo(ctx).WorkflowExecution.ID),
			}

			childCtx := workflow.WithChildOptions(ctx, opts)

			if err := workflow.ExecuteChildWorkflow(childCtx, t.catchChildWorkflowName, catchState.Input, catchState).Get(ctx, &res); err != nil {
				// The catch handler itself may emit `then: end`. Propagate
				// that as flow.ErrEnd rather than wrapping it as a generic
				// catch-workflow failure.
				if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
					logger.Info("Catch child workflow signalled end; propagating",
						"catchWorkflow", t.catchChildWorkflowName, "carriedOutput", endPayload.Output)
					return endPayload.Output, flow.ErrEnd
				}
				// Everything has failed
				logger.Error("Error calling try workflow", "error", err)
				return nil, fmt.Errorf("error calling catch workflow: %w", err)
			}
		}

		return res, nil
	}, nil
}

func (t *TryTaskBuilder) getTasks() map[string]*model.TaskList {
	return map[string]*model.TaskList{
		"try":   t.task.Try,
		"catch": t.task.Catch.Do,
	}
}

func (t *TryTaskBuilder) createBuilder(
	taskType string, list *model.TaskList,
) (childWorkflowName string, builder TaskBuilder, err error) {
	l := log.With().Str("task", t.GetTaskName()).Str("taskType", taskType).Logger()

	if len(*list) == 0 {
		err = fmt.Errorf("no tasks detected for %s in %s", taskType, t.GetTaskName())
		return
	}

	childWorkflowName = utils.GenerateChildWorkflowName(taskType, t.GetTaskName())

	b, err := NewTaskBuilder(childWorkflowName, &model.DoTask{Do: list}, t.temporalWorker, t.doc, t.eventEmitter, t.taskOpts)
	if err != nil {
		l.Error().Msg("Error creating the for task builder")
		err = fmt.Errorf("error creating the for task builder: %w", err)
		return
	}

	builder = b

	return
}

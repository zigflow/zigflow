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
	"reflect"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog/log"
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
}

func (t *TryTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	bodies := make(map[string]TemporalWorkflowFunc, len(t.getTasks()))
	for taskType, list := range t.getTasks() {
		builder, err := t.createBuilder(taskType, list)
		if err != nil {
			return nil, fmt.Errorf("error creating %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}

		bodies[taskType], err = builder.buildInline()
		if err != nil {
			log.Error().Str("task", t.GetTaskName()).Str("taskType", taskType).Msg("Error building inline tasks")
			return nil, fmt.Errorf("error building inline %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
	}

	return t.exec(bodies[tryBodyPathSegment], bodies[catchBodyPathSegment])
}

func (t *TryTaskBuilder) PostLoad() error {
	for taskType, list := range t.getTasks() {
		builder, err := t.createBuilder(taskType, list)
		if err != nil {
			return fmt.Errorf("error creating %s post load tasks for %s: %w", taskType, t.GetTaskName(), err)
		}

		if err = builder.PostLoad(); err != nil {
			log.Error().Str("task", t.GetTaskName()).Str("taskType", taskType).Msg("Error running post load for inline tasks")
			return fmt.Errorf("error running post load for inline %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
	}

	return nil
}

func (t *TryTaskBuilder) Validate() error {
	for taskType, list := range t.getTasks() {
		builder, err := t.createBuilder(taskType, list)
		if err != nil {
			return fmt.Errorf("error creating %s validate tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
		if err := builder.Validate(); err != nil {
			return fmt.Errorf("error validating %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
	}
	return nil
}

func (t *TryTaskBuilder) buildCatchError(err error) map[string]any {
	out := map[string]any{
		"message":      err.Error(),
		"nonRetryable": false,
	}

	errType := reflect.TypeOf(err)
	for errType.Kind() == reflect.Pointer {
		errType = errType.Elem()
	}
	if errType.Name() != "" {
		out["type"] = errType.Name()
	}

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

func (t *TryTaskBuilder) exec(tryFn, catchFn TemporalWorkflowFunc) (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (output any, err error) {
		logger := workflow.GetLogger(ctx)

		tryState := state.Clone()
		res, err := tryFn(ctx, tryState.Input, tryState)
		if err != nil {
			if errors.Is(err, flow.ErrEnd) {
				logger.Info("Try tasks signalled end; propagating without running catch")
				return res, err
			}

			logger.Warn("Try tasks failed, catching the error")

			// Expose the caught error to the catch tasks so they can
			// interrogate it. The Open Workflow Specification names this
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

			res, err = catchFn(ctx, catchState.Input, catchState)
			if err != nil {
				if errors.Is(err, flow.ErrEnd) {
					logger.Info("Catch tasks signalled end; propagating")
					return res, err
				}
				logger.Error("Catch tasks failed", "error", err)
				return nil, err
			}
		}

		return res, nil
	}, nil
}

func (t *TryTaskBuilder) getTasks() map[string]*model.TaskList {
	return map[string]*model.TaskList{
		tryBodyPathSegment:   t.task.Try,
		catchBodyPathSegment: t.task.Catch.Do,
	}
}

func (t *TryTaskBuilder) createBuilder(
	taskType string, list *model.TaskList,
) (*DoTaskBuilder, error) {
	l := log.With().Str("task", t.GetTaskName()).Str("taskType", taskType).Logger()

	if len(*list) == 0 {
		return nil, fmt.Errorf("no tasks detected for %s in %s", taskType, t.GetTaskName())
	}

	b, err := NewDoTaskBuilder(
		t.temporalWorker,
		&model.DoTask{Do: list},
		fmt.Sprintf("%s.%s", t.GetTaskName(), taskType),
		t.doc,
		t.eventEmitter,
		t.taskOpts,
	)
	if err != nil {
		l.Error().Msg("Error creating the try task builder")
		return nil, fmt.Errorf("error creating the try task builder: %w", err)
	}

	b.setTaskPath(t.childTaskPath(taskType))

	return b, nil
}

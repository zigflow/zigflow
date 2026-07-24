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
	tryFn, err := t.buildBody(tryBodyPathSegment, t.task.Try)
	if err != nil {
		log.Error().Str("task", t.GetTaskName()).Str("taskType", tryBodyPathSegment).Msg("Error building try body")
		return nil, err
	}

	catchFn, err := t.buildBody(catchBodyPathSegment, t.task.Catch.Do)
	if err != nil {
		log.Error().Str("task", t.GetTaskName()).Str("taskType", catchBodyPathSegment).Msg("Error building catch body")
		return nil, err
	}

	return t.exec(tryFn, catchFn)
}

func (t *TryTaskBuilder) PostLoad() error {
	for taskType, taskList := range t.getTasks() {
		builder, err := t.bodyBuilder(taskType, taskList)
		if err != nil {
			return fmt.Errorf("error registering %s post load tasks for %s: %w", taskType, t.GetTaskName(), err)
		}

		if err = builder.PostLoad(); err != nil {
			log.Error().Str("task", t.GetTaskName()).Str("taskType", taskType).Msg("Error post-loading try body")
			return fmt.Errorf("error building for post load workflow: %w", err)
		}
	}

	return nil
}

func (t *TryTaskBuilder) Validate() error {
	for taskType, taskList := range t.getTasks() {
		builder, err := t.bodyBuilder(taskType, taskList)
		if err != nil {
			return err
		}
		if err := builder.Validate(); err != nil {
			return fmt.Errorf("error validating %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}
	}

	return nil
}

func (t *TryTaskBuilder) buildCatchError(err error) map[string]any {
	out := map[string]any{}

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

	// The try body now runs inline, so a plain Go error (one that is not a
	// typed Temporal error) is a common shape. Rather than exposing an empty
	// map to the catch tasks, fall back to surfacing at least the message so
	// `$data.error` always carries something interrogable.
	if len(out) == 0 && err != nil {
		out["message"] = err.Error()
	}

	return out
}

// bodyBuilder constructs the inline DoTaskBuilder for the try or catch body,
// threading the matching path segment ("try" / "catch") so nested tasks that
// reuse a leaf name across the two bodies still register distinct per-task
// activity names.
func (t *TryTaskBuilder) bodyBuilder(taskType string, task *model.TaskList) (*DoTaskBuilder, error) {
	return newInlineDoBuilder(t.temporalWorker, task, "", t.doc, t.eventEmitter, t.taskOpts, t.childTaskPath(taskType))
}

// buildBody builds the try or catch body into an inline TemporalWorkflowFunc.
func (t *TryTaskBuilder) buildBody(taskType string, task *model.TaskList) (TemporalWorkflowFunc, error) {
	return buildInlineTaskList(t.temporalWorker, task, "", t.doc, t.eventEmitter, t.taskOpts, t.childTaskPath(taskType))
}

func (t *TryTaskBuilder) exec(tryFn, catchFn TemporalWorkflowFunc) (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Info("Starting try task")

		output, err := tryFn(ctx, input, state)
		if err != nil {
			logger.Warn("Try body failed, catching the error", "error", err)

			// A `then: end` directive inside the try body is a deliberate
			// workflow termination, not a failure to be caught: surface it as
			// flow.ErrEnd so the do-task pipeline can keep propagating end
			// upward, preserving the carried output so the root completion
			// reflects it. Crucially, this skips the catch handler.
			//
			// The try body runs inline, so it returns flow.ErrEnd directly
			// alongside its output. The DecodeEndApplicationError branch is
			// retained for backwards compatibility with an encoded end error
			// that may still arrive from a Temporal boundary.
			if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
				logger.Info("Try body signalled end; propagating without running catch", "carriedOutput", endPayload.Output)
				return endPayload.Output, flow.ErrEnd
			}
			if errors.Is(err, flow.ErrEnd) {
				logger.Info("Try body signalled end; propagating without running catch", "carriedOutput", output)
				return output, flow.ErrEnd
			}

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

			output, err = catchFn(ctx, catchState.Input, catchState)
			if err != nil {
				// Everything has failed

				// The catch handler itself may emit `then: end`. Propagate
				// that as flow.ErrEnd rather than wrapping it as a generic
				// catch failure. As with the try body, the inline catch
				// returns flow.ErrEnd directly; the decode branch stays for
				// backwards compatibility with an encoded end error.
				if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
					logger.Info("Catch body signalled end; propagating", "carriedOutput", endPayload.Output)
					return endPayload.Output, flow.ErrEnd
				}
				if errors.Is(err, flow.ErrEnd) {
					logger.Info("Catch body signalled end; propagating", "carriedOutput", output)
					return output, flow.ErrEnd
				}

				logger.Error("Error running catch tasks", "error", err)
				return nil, fmt.Errorf("error running catch tasks: %w", err)
			}
		}

		return output, nil
	}, nil
}

func (t *TryTaskBuilder) getTasks() map[string]*model.TaskList {
	return map[string]*model.TaskList{
		tryBodyPathSegment:   t.task.Try,
		catchBodyPathSegment: t.task.Catch.Do,
	}
}

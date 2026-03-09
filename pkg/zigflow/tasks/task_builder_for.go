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
	"context"
	"errors"
	"fmt"

	ceSDK "github.com/cloudevents/sdk-go/v2"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewForTaskBuilder(
	temporalWorker worker.Worker,
	task *model.ForTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (*ForTaskBuilder, error) {
	return &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

var errForkIterationStop = fmt.Errorf("fork iteration stop")

type ForTaskBuilder struct {
	builder[*model.ForTask]

	childWorkflowName string
}

func (t *ForTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	builder, err := t.createBuilder()
	if err != nil {
		return nil, err
	}
	if builder == nil {
		return nil, nil
	}

	if _, err := builder.Build(); err != nil {
		log.Error().Str("task", t.childWorkflowName).Err(err).Msg("Error building for workflow")
		return nil, fmt.Errorf("error building for workflow: %w", err)
	}

	return t.exec()
}

func (t *ForTaskBuilder) PostLoad() error {
	builder, err := t.createBuilder()
	if err != nil {
		return err
	}
	if builder == nil {
		return nil
	}

	if err := builder.PostLoad(); err != nil {
		log.Error().Str("task", t.childWorkflowName).Err(err).Msg("Error building for workflow postload")
		return fmt.Errorf("error building for workflow postload: %w", err)
	}

	return nil
}

// addIterationResult adds the latest iteration to the data - this will be overidden
// with each iteration so should only be relied upon inside the iterator
func (t *ForTaskBuilder) addIterationResult(ctx workflow.Context, state *utils.State, response any) {
	logger := workflow.GetLogger(ctx)

	cctx := context.Background()
	info := workflow.GetInfo(ctx)
	workflowID := info.WorkflowExecution.ID

	taskName := t.GetTaskName()

	logger.Debug("Adding iteration result to data object")
	state.AddData(map[string]any{
		taskName: response,
	})

	t.eventEmitter.Emit(cctx, "iteration.completed", func(e *ceSDK.Event) {
		e.SetID(workflowID)
		e.SetSubject(taskName)
		_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
			"state": state,
			"while": t.task.While,
		})
	})
}

func (t *ForTaskBuilder) createBuilder() (TaskBuilder, error) {
	if len(*t.task.Do) == 0 {
		log.Warn().Str("task", t.GetTaskName()).Msg("No do tasks detected in for task")
		return nil, nil
	}

	// Register the ForTask's Do as a child workflow
	t.childWorkflowName = utils.GenerateChildWorkflowName("for", t.GetTaskName())

	builder, err := NewTaskBuilder(t.childWorkflowName, &model.DoTask{Do: t.task.Do}, t.temporalWorker, t.doc, t.eventEmitter)
	if err != nil {
		log.Error().Str("task", t.childWorkflowName).Err(err).Msg("Error creating the for task builder")
		return nil, fmt.Errorf("error creating the for task builder: %w", err)
	}

	return builder, nil
}

func (t *ForTaskBuilder) exec() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		data, err := utils.EvaluateString(t.task.For.In, nil, state)
		if err != nil {
			logger.Error("Error parsing for task data list", "data", t.task.For.In, "task", t.GetTaskName())
			return nil, fmt.Errorf("error parsing for task data list: %w", err)
		}

		switch v := data.(type) {
		case map[string]any:
			logger.Debug("Iterating data as object", "task", t.GetTaskName())
			output := map[string]any{}
			for key, value := range v {
				res, err := t.iterator(ctx, key, value, state.Clone().ClearOutput())
				if err != nil {
					if errors.Is(err, errForkIterationStop) {
						break
					}
					return nil, err
				}

				t.addIterationResult(ctx, state, res)

				output[key] = res
			}

			return output, nil
		case []any:
			logger.Debug("Iterating data as array", "task", t.GetTaskName())
			output := make([]any, 0)
			for i, value := range v {
				res, err := t.iterator(ctx, i, value, state.Clone().ClearOutput())
				if err != nil {
					if errors.Is(err, errForkIterationStop) {
						break
					}
					return nil, err
				}

				t.addIterationResult(ctx, state, res)

				output = append(output, res)
			}

			return output, nil
		case int:
			logger.Debug("Iterating data as a number", "task", t.GetTaskName())
			output := make([]any, 0)
			for i := range v {
				res, err := t.iterator(ctx, i, i, state.Clone().ClearOutput())
				if err != nil {
					if errors.Is(err, errForkIterationStop) {
						break
					}
					return nil, err
				}

				t.addIterationResult(ctx, state, res)

				output = append(output, res)
			}

			return output, nil
		default:
			logger.Error("For task data is not iterable", "task", t.GetTaskName())
			return nil, fmt.Errorf("for task data is not iterable")
		}
	}, nil
}

func (t *ForTaskBuilder) iterator(ctx workflow.Context, key, value any, state *utils.State) (any, error) {
	logger := workflow.GetLogger(ctx)

	keyVar := t.task.For.At
	if keyVar == "" {
		keyVar = "index"
	}
	valueVar := t.task.For.Each
	if valueVar == "" {
		valueVar = "item"
	}

	state.AddData(map[string]any{
		keyVar:   key,
		valueVar: value,
	})

	// Check if this iteration should be run according to the while test
	if shouldRun, err := t.checkWhile(ctx, state); err != nil {
		logger.Error("Error checking for while", "error", err, "key", key, "task", t.GetTaskName())
		return nil, fmt.Errorf("error checking for while: %w", err)
	} else if !shouldRun {
		logger.Debug("For while responded false - stopping iteration", "key", key, "task", t.GetTaskName())
		return nil, errForkIterationStop
	}

	// Run the tasks
	opts := workflow.ChildWorkflowOptions{
		// key may be an integer or a string - use %v to let Go figure out how to represent it
		WorkflowID: fmt.Sprintf("%s_for_%v", workflow.GetInfo(ctx).WorkflowExecution.ID, key),
	}
	childCtx := workflow.WithChildOptions(ctx, opts)

	logger.Info("Triggering forked child workflow", "name", t.childWorkflowName)

	var res any
	if err := workflow.ExecuteChildWorkflow(childCtx, t.childWorkflowName, state.Input, state).Get(ctx, &res); err != nil {
		logger.Error("Error calling for workflow", "error", err, "workflow", t.childWorkflowName)
		return nil, fmt.Errorf("error calling for workflow: %w", err)
	}

	return res, nil
}

// checkWhile decides if we should stop the iteration
func (t *ForTaskBuilder) checkWhile(ctx workflow.Context, state *utils.State) (res bool, err error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Checking the while response", "value", t.task.While, "task", t.GetTaskName())

	if t.task.While == "" {
		res = true
		return
	}

	whileRes, err := utils.EvaluateString(t.task.While, nil, state)
	if err != nil {
		logger.Error("Error parsing for task while", "data", t.task.While, "task", t.GetTaskName())
		err = fmt.Errorf("error parsing for task data list: %w", err)
		return
	}

	if v, ok := whileRes.(bool); ok {
		logger.Debug("Task while has resolved", "response", v)
		res = v
		return
	}

	logger.Warn("Task while has resolved to a non-boolean - responding with false", "response", whileRes)

	return
}

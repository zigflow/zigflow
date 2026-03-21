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

// forChildResult is returned by the for-loop's child workflow and carries
// intra-loop state only. Neither field is promoted to the parent workflow state.
//
// Output is the per-iteration result. It is stored in workingState.Output so
// that the next iteration's while condition can read $output. The loop's final
// aggregated result is returned from exec() and handled by the surrounding task
// pipeline, not taken from this field.
//
// Context carries any value written by an export directive inside the iteration.
// It is stored in workingState.Context so that the next iteration can read
// $context. It is NOT copied to the parent state after the loop completes:
// workingState.Context is loop-private, and copying it would cause inner-loop
// exports to leak into the surrounding workflow context.
type forChildResult struct {
	Output  any `json:"output"`
	Context any `json:"context"`
}

type ForTaskBuilder struct {
	builder[*model.ForTask]

	childWorkflowName string
}

func (t *ForTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	if len(*t.task.Do) == 0 {
		log.Warn().Str("task", t.GetTaskName()).Msg("No do tasks detected in for task")
		return nil, nil
	}

	t.childWorkflowName = utils.GenerateChildWorkflowName("for", t.GetTaskName())

	// Build the inner DoTask with registration disabled so we can register our
	// own wrapper that returns forChildResult instead of bare state.Output.
	innerBuilder, err := NewDoTaskBuilder(
		t.temporalWorker,
		&model.DoTask{Do: t.task.Do},
		t.childWorkflowName,
		t.doc,
		t.eventEmitter,
		DoTaskOpts{DisableRegisterWorkflow: true},
	)
	if err != nil {
		log.Error().Str("task", t.childWorkflowName).Err(err).Msg("Error creating the for task builder")
		return nil, fmt.Errorf("error creating the for task builder: %w", err)
	}

	innerFn, err := innerBuilder.Build()
	if err != nil {
		log.Error().Str("task", t.childWorkflowName).Err(err).Msg("Error building for workflow")
		return nil, fmt.Errorf("error building for workflow: %w", err)
	}

	// Register a wrapper child workflow that returns the iteration output and
	// exported context so the loop can propagate inter-iteration state.
	t.temporalWorker.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input any, state *utils.State) (forChildResult, error) {
			output, err := innerFn(ctx, input, state)
			if err != nil {
				return forChildResult{}, err
			}
			return forChildResult{
				Output:  output,
				Context: state.Context,
			}, nil
		},
		workflow.RegisterOptions{Name: t.childWorkflowName},
	)

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

// addIterationResult adds the latest iteration to the data - this will be overridden
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

		// workingState is an isolated copy that accumulates inter-iteration state
		// (Context and Output) without mutating the parent state mid-loop.
		// Loop-local variables such as index/item are placed on a further per-iteration
		// clone inside iterator() and therefore never appear in workingState or state.
		workingState := state.Clone()
		workingState.Output = nil

		output, err := t.iterate(ctx, workingState, data)
		if err != nil {
			// Parent state is not modified on error: state.Output is only set
			// after a clean exit so retries and catch handlers see the original state.
			return nil, err
		}

		// state.Output is set to the aggregated result (array or object) so that
		// output: expressions on the for task behave consistently with other tasks.
		//
		// workingState.Context is intentionally NOT copied to state.Context.
		// It is loop-private state used only to carry exported values from one
		// iteration to the next. Copying it to the parent would cause inner-loop
		// export values to leak into the surrounding workflow context and would
		// overwrite any context already established by earlier tasks.
		// If the caller needs to surface a value from inside the loop, they should
		// use an output: expression on the for task itself.
		state.Output = output
		return output, nil
	}, nil
}

// iterate runs the loop body for each element in data, accumulating results.
// workingState is updated after each successful iteration: Context carries any
// value written by an export directive, Output carries the iteration result so
// that the next iteration's while condition can read $output.
func (t *ForTaskBuilder) iterate(ctx workflow.Context, workingState *utils.State, data any) (any, error) {
	logger := workflow.GetLogger(ctx)

	switch v := data.(type) {
	case map[string]any:
		logger.Debug("Iterating data as object", "task", t.GetTaskName())
		output := map[string]any{}
		for key, value := range v {
			res, err := t.iterator(ctx, key, value, workingState)
			if err != nil {
				if errors.Is(err, errForkIterationStop) {
					break
				}
				return nil, err
			}
			t.addIterationResult(ctx, workingState, res)
			output[key] = res
		}
		return output, nil
	case []any:
		logger.Debug("Iterating data as array", "task", t.GetTaskName())
		output := make([]any, 0)
		for i, value := range v {
			res, err := t.iterator(ctx, i, value, workingState)
			if err != nil {
				if errors.Is(err, errForkIterationStop) {
					break
				}
				return nil, err
			}
			t.addIterationResult(ctx, workingState, res)
			output = append(output, res)
		}
		return output, nil
	case int:
		logger.Debug("Iterating data as a number", "task", t.GetTaskName())
		output := make([]any, 0)
		for i := range v {
			res, err := t.iterator(ctx, i, i, workingState)
			if err != nil {
				if errors.Is(err, errForkIterationStop) {
					break
				}
				return nil, err
			}
			t.addIterationResult(ctx, workingState, res)
			output = append(output, res)
		}
		return output, nil
	default:
		logger.Error("For task data is not iterable", "task", t.GetTaskName())
		return nil, fmt.Errorf("for task data is not iterable")
	}
}

// iterator runs one iteration of the for loop.
//
// workingState carries accumulated cross-iteration state:
// - Context from export directives
// - Output from the previous iteration for while evaluation
// - the last iteration result under the loop task name via addIterationResult
//
// Instead, iterator creates an iterState clone that includes the loop-local
// variables and is passed to the child workflow. Only Context and Output are
// propagated back from the child into workingState.
func (t *ForTaskBuilder) iterator(ctx workflow.Context, key, value any, workingState *utils.State) (any, error) {
	logger := workflow.GetLogger(ctx)

	keyVar := t.task.For.At
	if keyVar == "" {
		keyVar = "index"
	}
	valueVar := t.task.For.Each
	if valueVar == "" {
		valueVar = "item"
	}

	// Build a per-iteration state that adds the loop-local variables.
	// Using a clone of workingState means:
	//   - $context and $output from the previous iteration are visible here
	//   - keyVar/valueVar do not pollute workingState or the parent state
	iterState := workingState.Clone()
	iterState.AddData(map[string]any{
		keyVar:   key,
		valueVar: value,
	})

	// Evaluate while against iterState so that:
	//   - $output reflects the previous iteration's output (from workingState)
	//   - $data.keyVar / $data.valueVar reflect the current iteration's variables
	if shouldRun, err := t.checkWhile(ctx, iterState); err != nil {
		logger.Error("Error checking for while", "error", err, "key", key, "task", t.GetTaskName())
		return nil, fmt.Errorf("error checking for while: %w", err)
	} else if !shouldRun {
		logger.Debug("For while responded false - stopping iteration", "key", key, "task", t.GetTaskName())
		return nil, errForkIterationStop
	}

	// Clear output so the child starts with a clean slate.
	// Context is deliberately preserved in iterState so that exports from the
	// previous iteration are visible to tasks inside this iteration via $context.
	iterState.Output = nil

	// Run the tasks
	opts := workflow.ChildWorkflowOptions{
		// key may be an integer or a string - use %v to let Go figure out how to represent it
		WorkflowID: fmt.Sprintf("%s_for_%v", workflow.GetInfo(ctx).WorkflowExecution.ID, key),
	}
	childCtx := workflow.WithChildOptions(ctx, opts)

	logger.Info("Triggering forked child workflow", "name", t.childWorkflowName)

	var res forChildResult
	if err := workflow.ExecuteChildWorkflow(childCtx, t.childWorkflowName, iterState.Input, iterState).Get(ctx, &res); err != nil {
		logger.Error("Error calling for workflow", "error", err, "workflow", t.childWorkflowName)
		return nil, fmt.Errorf("error calling for workflow: %w", err)
	}

	// Propagate only Context and Output back into workingState so that the
	// next iteration's while check and $context references see current values.
	// Data mutations made inside the child workflow are intentionally discarded:
	// they are child-internal. The only Data update that crosses iteration
	// boundaries is the one made by addIterationResult during loop iteration.
	workingState.Context = res.Context
	workingState.Output = res.Output

	return res.Output, nil
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

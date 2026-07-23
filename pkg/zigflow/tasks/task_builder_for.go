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
	"math"

	ceSDK "github.com/cloudevents/sdk-go/v2"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog/log"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewForTaskBuilder(
	temporalWorker worker.Worker,
	task *model.ForTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*ForTaskBuilder, error) {
	return &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

// errForkIterationStop is an internal signal returned by iterator() when the
// while condition evaluates false. It is not a failure: iterate() interprets
// it as "stop the loop quietly" and never surfaces it to the caller.
var errForkIterationStop = fmt.Errorf("fork iteration stop")

type ForTaskBuilder struct {
	builder[*model.ForTask]
}

func (t *ForTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	// Build the for body once as an ordinary inline function. It executes in
	// the current workflow for every iteration rather than as a child workflow.
	bodyFn, err := buildInlineTaskList(t.temporalWorker, t.task.Do, "", t.doc, t.eventEmitter, t.taskOpts, t.taskPath)
	if err != nil {
		log.Error().Str("task", t.GetTaskName()).Err(err).Msg("Error building for body")
		return nil, fmt.Errorf("error building for body: %w", err)
	}

	return t.exec(bodyFn)
}

func (t *ForTaskBuilder) PostLoad() error {
	builder, err := t.bodyBuilder()
	if err != nil {
		return err
	}

	if err := builder.PostLoad(); err != nil {
		log.Error().Str("task", t.GetTaskName()).Err(err).Msg("Error post-loading for body")
		return fmt.Errorf("error building for workflow postload: %w", err)
	}

	return nil
}

func (t *ForTaskBuilder) Validate() error {
	builder, err := t.bodyBuilder()
	if err != nil {
		return err
	}

	if err := builder.Validate(); err != nil {
		return fmt.Errorf("error validating for workflow: %w", err)
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
			constKeyState: state,
			"while":       t.task.While,
		})
	})
}

// bodyBuilder constructs the inline DoTaskBuilder for the for body without
// registering a workflow. It backs PostLoad and Validate so their configuration
// cannot drift from Build's.
func (t *ForTaskBuilder) bodyBuilder() (*DoTaskBuilder, error) {
	return newInlineDoBuilder(t.temporalWorker, t.task.Do, "", t.doc, t.eventEmitter, t.taskOpts, t.taskPath)
}

func (t *ForTaskBuilder) exec(bodyFn TemporalWorkflowFunc) (TemporalWorkflowFunc, error) {
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

		output, err := t.iterate(ctx, workingState, data, bodyFn)
		if err != nil {
			// flow.ErrEnd is special: an iteration deliberately requested
			// that the workflow terminate, and the iterator has already
			// surfaced its effective output. Propagate that output to the
			// do-task layer so subsequent processTaskOutput / Export logic
			// reflect the iteration's work rather than losing it.
			if errors.Is(err, flow.ErrEnd) {
				state.Output = output
				return output, err
			}
			// For other errors the parent state is not modified: state.Output
			// is only set after a clean exit so retries and catch handlers
			// see the original state.
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
func (t *ForTaskBuilder) iterate(ctx workflow.Context, workingState *utils.State, data any, bodyFn TemporalWorkflowFunc) (any, error) {
	logger := workflow.GetLogger(ctx)

	switch v := data.(type) {
	case map[string]any:
		logger.Debug("Iterating data as object", "task", t.GetTaskName())
		output := map[string]any{}
		for key, value := range v {
			res, err := t.iterator(ctx, key, value, workingState, bodyFn)
			if done, endRes, endErr := t.classifyIterationOutcome(res, err); done {
				if endErr != nil {
					return endRes, endErr
				}
				break
			}
			t.addIterationResult(ctx, workingState, res)
			output[key] = res
		}
		return output, nil
	case []any:
		logger.Debug("Iterating data as array", "task", t.GetTaskName())
		output := make([]any, 0)
		for i, value := range v {
			res, err := t.iterator(ctx, i, value, workingState, bodyFn)
			if done, endRes, endErr := t.classifyIterationOutcome(res, err); done {
				if endErr != nil {
					return endRes, endErr
				}
				break
			}
			t.addIterationResult(ctx, workingState, res)
			output = append(output, res)
		}
		return output, nil
	}

	count, ok, err := t.iterationCount(data)
	if err != nil {
		return nil, err
	}
	if !ok {
		logger.Error("For task data is not iterable", "task", t.GetTaskName())
		return nil, fmt.Errorf("for task data is not iterable")
	}

	logger.Debug("Iterating data as a number", "task", t.GetTaskName())
	output := make([]any, 0)
	for i := range count {
		res, err := t.iterator(ctx, i, i, workingState, bodyFn)
		if done, endRes, endErr := t.classifyIterationOutcome(res, err); done {
			if endErr != nil {
				return endRes, endErr
			}
			break
		}
		t.addIterationResult(ctx, workingState, res)
		output = append(output, res)
	}
	return output, nil
}

// classifyIterationOutcome interprets the per-iteration (res, err) pair.
//
//	(false, _, _)     => no error, keep accumulating.
//	(true, _, nil)    => an internal stop signal (e.g. while-false);
//	                     caller should break out of the loop quietly.
//	(true, res, end)  => the iteration body emitted flow.ErrEnd and the
//	                     iteration's effective output must be propagated
//	                     to the for-task layer so it can survive the
//	                     workflow termination.
//	(true, nil, err)  => a genuine failure that must propagate to the
//	                     caller; the loop aborts.
func (t *ForTaskBuilder) classifyIterationOutcome(res any, err error) (done bool, output any, propagate error) {
	if err == nil {
		return false, nil, nil
	}
	if errors.Is(err, errForkIterationStop) {
		return true, nil, nil
	}
	if errors.Is(err, flow.ErrEnd) {
		return true, res, err
	}
	return true, nil, err
}

// iterationCount resolves a numeric for.in value into a concrete iteration
// count. Variables decoded from JSON arrive as float64 even when authored as
// integer literals, so float64 values that represent whole numbers are
// accepted alongside native int. Fractional float64 values are rejected
// rather than silently truncated so a surprising loop count can never occur.
// NaN, infinite, and out-of-int-range float64 values are rejected explicitly
// so that float-to-int conversion cannot produce an undefined or platform
// dependent result. ok reports whether data was numeric at all; non-numeric
// values are the caller's responsibility to handle.
func (t *ForTaskBuilder) iterationCount(data any) (count int, ok bool, err error) {
	switch n := data.(type) {
	case int:
		return n, true, nil
	case float64:
		if math.IsNaN(n) {
			return 0, true, fmt.Errorf("for task numeric iteration value cannot be NaN")
		}
		if math.IsInf(n, 0) {
			return 0, true, fmt.Errorf(
				"for task numeric iteration value cannot be infinite: %v",
				n,
			)
		}
		if n > math.MaxInt || n < math.MinInt {
			return 0, true, fmt.Errorf(
				"for task numeric iteration value out of int range: %v",
				n,
			)
		}
		if n != math.Trunc(n) {
			return 0, true, fmt.Errorf(
				"for task numeric iteration value must be a whole number: %v",
				n,
			)
		}
		return int(n), true, nil
	}
	return 0, false, nil
}

// iterator runs one iteration of the for loop by invoking the inline body
// function in the current workflow.
//
// workingState carries accumulated cross-iteration state:
// - Context from export directives
// - Output from the previous iteration for while evaluation
// - the last iteration result under the loop task name via addIterationResult
//
// iterator creates an iterState clone that includes the loop-local variables
// and passes it to bodyFn. Only Context and Output are propagated back into
// workingState; the body's other Data mutations stay isolated to the iteration.
func (t *ForTaskBuilder) iterator(
	ctx workflow.Context, key, value any, workingState *utils.State, bodyFn TemporalWorkflowFunc,
) (any, error) {
	logger := workflow.GetLogger(ctx)

	keyVar := t.task.For.At
	if keyVar == "" {
		keyVar = "index"
	}
	valueVar := t.task.For.Each
	if valueVar == "" {
		valueVar = constDefaultItemVar
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

	// Clear output so the iteration body starts with a clean slate.
	// Context is deliberately preserved in iterState so that exports from the
	// previous iteration are visible to tasks inside this iteration via $context.
	iterState.Output = nil

	logger.Debug("Running inline for iteration body", "key", key, "task", t.GetTaskName())

	output, err := bodyFn(ctx, iterState.Input, iterState)
	if err != nil {
		// A `then: end` directive inside the iteration body deliberately
		// terminates the workflow. The inline body surfaces flow.ErrEnd
		// directly, carrying its effective output. Surface it as flow.ErrEnd
		// to the enclosing scope rather than reporting a generic failure, and
		// update workingState.Output so the for-task returns the iteration's
		// effective output rather than losing it.
		//
		// The DecodeEndApplicationError branch is retained only for
		// backwards compatibility with an encoded end error that could still
		// arrive from a Temporal boundary. It is checked first so its carried
		// payload is not lost; it is not the primary inline path.
		if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
			logger.Info("For iteration signalled end (encoded); propagating to caller", "carriedOutput", endPayload.Output)
			workingState.Output = endPayload.Output
			return endPayload.Output, flow.ErrEnd
		}
		if errors.Is(err, flow.ErrEnd) {
			logger.Info("For iteration signalled end; propagating to caller", "carriedOutput", output)
			workingState.Output = output
			return output, flow.ErrEnd
		}
		logger.Error("Error running for iteration tasks", "error", err, "key", key, "task", t.GetTaskName())
		return nil, fmt.Errorf("error running for iteration tasks: %w", err)
	}

	// Propagate only Context and Output back into workingState so that the
	// next iteration's while check and $context references see current values.
	// Data mutations made inside the body are intentionally discarded: they
	// are iteration-internal. The only Data update that crosses iteration
	// boundaries is the one made by addIterationResult during loop iteration.
	workingState.Context = iterState.Context
	workingState.Output = output

	return output, nil
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

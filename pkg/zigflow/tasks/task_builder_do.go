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
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type DoTaskOpts struct {
	DisableRegisterWorkflow bool
	Envvars                 map[string]any
	MaxHistoryLength        int
	Telemetry               *telemetry.Telemetry
	Validator               *utils.Validator
}

func NewDoTaskBuilder(
	temporalWorker worker.Worker,
	task *model.DoTask,
	workflowName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
	opts ...DoTaskOpts,
) (*DoTaskBuilder, error) {
	var doOpts DoTaskOpts
	if len(opts) == 1 {
		doOpts = opts[0]
	}
	if doOpts.Envvars == nil {
		doOpts.Envvars = map[string]any{}
	}

	return &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           workflowName,
			neverSkipCAN:   true,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
		opts: doOpts,
	}, nil
}

type DoTaskBuilder struct {
	builder[*model.DoTask]
	opts DoTaskOpts
}

type workflowFunc struct {
	TaskBuilder

	Func TemporalWorkflowFunc
	Name string
}

func (t *DoTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	tasks := make([]workflowFunc, 0)

	var hasNoDo bool
	for _, task := range *t.task.Do {
		l := log.With().Str("task", task.Key).Str("workflow", t.GetTaskName()).Logger()

		addTasks := true
		if do := task.AsDoTask(); do == nil {
			l.Debug().Msg("No do task detected")
			hasNoDo = true
		} else if hasNoDo {
			l.Debug().Msg("Nested do task detected - ignoring")
			addTasks = false
		}

		// Build a task builder
		l.Debug().Msg("Creating task builder")
		builder, err := NewTaskBuilder(task.Key, task.Task, t.temporalWorker, t.doc, t.eventEmitter, t.taskOpts)
		if err != nil {
			return nil, fmt.Errorf("error creating task builder: %w", err)
		}

		// Build the task and store it for use
		l.Debug().Msg("Building task")
		fn, err := builder.Build()
		if err != nil {
			return nil, fmt.Errorf("error building task: %w", err)
		}
		if fn != nil && addTasks {
			l.Debug().Msg("Adding task to workflow")
			tasks = append(tasks, workflowFunc{
				Func:        fn,
				Name:        builder.GetTaskName(),
				TaskBuilder: builder,
			})
		}
	}

	// Execute the workflow
	wf := t.workflowExecutor(tasks)

	if !t.opts.DisableRegisterWorkflow {
		if hasNoDo {
			log.Debug().Str("name", t.GetTaskName()).Msg("Registering workflow")
			t.temporalWorker.RegisterWorkflowWithOptions(wf, workflow.RegisterOptions{
				Name: t.GetTaskName(),
			})
		}
	}

	return wf, nil
}

func (t *DoTaskBuilder) PostLoad() error {
	for _, task := range *t.task.Do {
		l := log.With().Str("task", task.Key).Logger()

		// Build a task builder
		l.Debug().Msg("Creating prep task builder")
		builder, err := NewTaskBuilder(task.Key, task.Task, t.temporalWorker, t.doc, t.eventEmitter, t.taskOpts)
		if err != nil {
			return fmt.Errorf("error creating task prep builder: %w", err)
		}

		// Run the postload task
		l.Debug().Msg("Run post load task")
		if err := builder.PostLoad(); err != nil {
			return fmt.Errorf("error running task post load: %w", err)
		}
	}
	return nil
}

// validateInput validates the input if it exists
func (t *DoTaskBuilder) validateInput(ctx workflow.Context, inputDef *model.Input, state *utils.State) error {
	logger := workflow.GetLogger(ctx)

	if inputDef != nil {
		logger.Debug("Validating input against schema")
		if err := swUtil.ValidateSchema(state.Input, inputDef.Schema, t.GetTaskName()); err != nil {
			logger.Error("Input failed data validation", "error", err)

			return temporal.NewNonRetryableApplicationError(
				"Workflow input did not meet JSON schema specification",
				"Validation",
				err,
				// There is additional detail useful in here
				err.(*model.Error),
			)
		}
	}

	return nil
}

// workflowExecutor executes the workflow by iterating through the tasks in order
func (t *DoTaskBuilder) workflowExecutor(tasks []workflowFunc) TemporalWorkflowFunc {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Info("Running workflow", "workflow", t.GetTaskName())

		// Increment number of workflow runs - a new Do task, even if a child workflow,
		// counts as a new workflow run. This is designed to be an indication of usage
		if t.opts.Telemetry != nil && !workflow.IsReplaying(ctx) {
			t.opts.Telemetry.IncrementRun()
		}

		if state == nil {
			logger.Debug("Creating new state instance")
			state = utils.NewState().AddWorkflowInfo(ctx)
			state.Env = t.opts.Envvars
			state.Input = input

			// Validate input for the whole document
			logger.Debug("Validating input against document")
			if err := t.validateInput(ctx, t.doc.Input, state); err != nil {
				logger.Debug("Document input validation error", "error", err)
				return nil, err
			}
		}

		t.eventEmitter.Emit(context.Background(), "workflow.started", func(e *ceSDK.Event) {
			e.SetID(workflow.GetInfo(ctx).WorkflowExecution.ID)
			_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
				constKeyInput: input,
				constKeyState: state,
			})
		})

		// Iterate through the tasks to create the workflow. An `end` flow
		// directive propagates outward as flow.ErrEnd. At the true root
		// workflow boundary this is a clean termination, not a failure.
		// At nested child workflow boundaries (e.g. redirect targets,
		// for-loop iteration bodies) we re-emit ErrEnd as a serialisable
		// Temporal ApplicationError so the parent workflow can detect it
		// and continue propagating "end" upward until it reaches the
		// root. Without this, ErrEnd would be silently swallowed by every
		// child workflow boundary and `then: end` could only end the
		// innermost scope rather than the overall workflow.
		//
		// state.Output is included in the end signal so the child's
		// effective output survives the boundary; the parent's
		// executeRedirect decodes it and applies the originating task's
		// output/export directives to it before propagating end further
		// upward.
		if err := t.iterateTasks(ctx, tasks, input, state); err != nil {
			if !errors.Is(err, flow.ErrEnd) {
				return nil, err
			}
			if !isRootWorkflow(ctx) {
				return nil, flow.NewEndApplicationError(state.Output)
			}
		}

		t.eventEmitter.Emit(context.Background(), "workflow.completed", func(e *ceSDK.Event) {
			e.SetID(workflow.GetInfo(ctx).WorkflowExecution.ID)
			_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
				"output": state.Output,
			})
		})

		return state.Output, nil
	}
}

func (t *DoTaskBuilder) continueAsNew(
	ctx workflow.Context, wfn, taskID string, input any, state *utils.State,
) error {
	logger := workflow.GetLogger(ctx)

	err := workflow.Await(ctx, func() bool {
		return workflow.AllHandlersFinished(ctx)
	})
	if err != nil {
		logger.Error("Failed to wait for handers to finish", "error", err)
		return fmt.Errorf("failed to wait for handlers to finish: %w", err)
	}

	logger.Info("Continuing as new", "taskId", taskID)
	state.CANStartFrom = utils.Ptr(taskID)
	return workflow.NewContinueAsNewError(ctx, wfn, input, state)
}

func (t *DoTaskBuilder) iterateTasks(
	ctx workflow.Context, tasks []workflowFunc, input any, state *utils.State,
) error {
	var nextTargetName *string
	logger := workflow.GetLogger(ctx)

	for i, task := range tasks {
		taskID := fmt.Sprintf("%s-%d", task.GetTaskName(), i)
		if t.shouldContinueAsNew(ctx) {
			logger.Debug("Task continue-as-new", "taskID", taskID, "workflow", t.name)
			return t.continueAsNew(ctx, t.name, taskID, input, state)
		}
		if t.shouldSkip(taskID, task, state) {
			logger.Debug("Skipping complete continue-as-new task", "taskID", taskID, "workflow", t.name)
			continue
		}

		taskBase := task.GetTask().GetBase()

		state.AddData(map[string]any{
			"task": map[string]any{
				"name": task.GetTaskName(),
			},
		})

		if nextTargetName != nil {
			logger.Debug("Check if a next task is set and it's this one", "task", task.Name)
			if task.Name == *nextTargetName {
				logger.Debug("Task is next one to be run from flow directive", "task", task.Name)
				// We've found the desired task - reset
				nextTargetName = nil
			} else {
				// Not the task - skip
				logger.Debug("Skipping task as not one set as next target", "task", task.Name, "nextTask", *nextTargetName)
				continue
			}
		}

		logger.Debug("Check if task should be run", "task", task.Name)
		if toRun, err := task.ShouldRun(state); err != nil {
			logger.Error("Error checking if statement", "error", err, "name", task.Name)
			return err
		} else if !toRun {
			logger.Debug("Skipping task as if statement resolve as false", "name", task.Name)
			continue
		}

		// Check input for the task
		logger.Debug("Validating input against task", "name", task.Name)
		if err := t.validateInput(ctx, taskBase.Input, state); err != nil {
			logger.Debug("Task input validation error", "error", err)
			return err
		}

		logger.Debug("Parse metadata", "name", task.Name)
		if err := task.ParseMetadata(ctx, state); err != nil {
			logger.Error("Error parsing metadata", "error", err)
			return err
		}

		if wCtx, err := metadata.SetActivityOptions(ctx, t.doc, taskBase, task.Name); err != nil {
			return err
		} else {
			ctx = wCtx
		}

		stop, err := t.runTaskAndHandleFlow(ctx, task, taskBase, input, state, &nextTargetName)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	if nextTargetName != nil {
		logger.Error("Next target specified but not found", "targetTask", nextTargetName)
		return fmt.Errorf("next target specified but not found: %s", *nextTargetName)
	}

	return nil
}

// runTaskAndHandleFlow drives a single task through the standard
// pipeline:
//
//  1. Execute the task and capture its output and any flow directive it
//     emitted (e.g. a switch returning flow.ErrContinue).
//  2. Process output and export, then emit task.completed.
//  3. Dispatch the effective flow directive.
//
// The effective directive is whichever the task itself emitted, or
// flow.FromDirective(taskBase.Then) if the task had no opinion. This is
// how task-level `then:` lines feed through the same control-flow
// machinery as switch cases: there is exactly one dispatch path for
// continue, exit, end and named redirects.
//
// Switch-emitted redirects (where the task signals flow.RedirectError as
// its result) are the one exception to the "process output/export now"
// rule: executeRedirect runs the named child workflow and the switch
// task's output/export expressions should apply to the redirect target's
// result rather than to the switch's nil placeholder output. For that
// case we defer output/export/completed until after executeRedirect.
func (t *DoTaskBuilder) runTaskAndHandleFlow(
	ctx workflow.Context,
	task workflowFunc,
	taskBase *model.TaskBase,
	input any,
	state *utils.State,
	nextTargetName **string,
) (stop bool, err error) {
	res, rerr := t.runTask(ctx, task, input, state)
	if rerr != nil {
		return false, rerr
	}

	// A cancelled task has already had task.cancelled emitted by
	// runTask. It must not be treated as successful completion: no
	// output processing, no export processing, no task.completed event,
	// and state.Output is preserved so the previous task's output stays
	// visible. Iteration continues normally to the next task because
	// cancellation is not modelled as a flow directive.
	if res.Cancelled {
		return false, nil
	}

	// Decide which directive (if any) to dispatch. A directive emitted
	// by the task itself (e.g. switch) takes precedence over taskBase.Then.
	directive, fromTaskLevel := effectiveDirective(res.Directive, taskBase)

	// Switch-emitted redirects own their own output/export pipeline via
	// executeRedirect; for everything else process now so taskBase.Output,
	// taskBase.Export and the task.completed event fire even when the task
	// is purely control-flow.
	//
	// controlOnly tells processTaskOutput to leave state.Output alone if the
	// task did not define an output directive: a routing task such as a
	// switch did not produce a payload, so the previous task's output should
	// remain visible to downstream tasks.
	if !isSwitchRedirect(res.Directive) {
		controlOnly := res.Directive != nil
		effectiveOutput, err := t.processTaskOutput(task, res.Output, state, controlOnly)
		if err != nil {
			return false, fmt.Errorf("error processing task output: %w", err)
		}
		if err := t.processTaskExport(task, res.Output, state); err != nil {
			return false, fmt.Errorf("error processing task export: %w", err)
		}
		// task.completed reports the effective output (i.e. the value
		// recorded on state.Output after taskBase.Output processing) so
		// event consumers see the transformed payload, not the raw task
		// result. For control-only tasks with no output directive this
		// is nil, matching the prior emission contract for those cases.
		t.emitTaskCompleted(ctx, task, input, effectiveOutput, state)
	}

	if directive == nil {
		return false, nil
	}

	return t.dispatchFlowDirective(ctx, task, directive, input, state, nextTargetName, fromTaskLevel)
}

// effectiveDirective picks the directive to dispatch.
// Task-emitted directives take precedence: a switch that signalled
// flow.ErrEnd should not be silently overridden by an unrelated
// taskBase.Then on the underlying TaskBase.
func effectiveDirective(taskDirective error, taskBase *model.TaskBase) (directive error, fromTaskLevel bool) {
	if taskDirective != nil {
		return taskDirective, false
	}
	if taskBase != nil && taskBase.Then != nil {
		return flow.FromDirective(taskBase.Then), true
	}
	return nil, false
}

// isSwitchRedirect reports whether the task-emitted directive is a
// RedirectError originating from a task body (typically a switch). Such
// redirects are dispatched via executeRedirect, which handles output,
// export and the completion event using the redirect target's result.
func isSwitchRedirect(taskDirective error) bool {
	if taskDirective == nil {
		return false
	}
	var redirect flow.RedirectError
	return errors.As(taskDirective, &redirect)
}

// dispatchFlowDirective is the single point that maps a flow directive
// to its execution effect. It is used for both task-emitted directives
// (e.g. switch) and task-level taskBase.Then directives so they share
// identical semantics for continue, exit and end.
//
// For named redirects the dispatch depends on origin:
//   - task-emitted (switch) RedirectError runs the named workflow as a
//     Temporal child workflow via executeRedirect.
//   - task-level (taskBase.Then) named target sets nextTargetName so
//     iterateTasks skips forward to the sibling task in the current
//     scope. This preserves the existing same-scope redirect behaviour
//     covered by TestDoTaskBuilderIterateTasksFlowControl.
func (t *DoTaskBuilder) dispatchFlowDirective(
	ctx workflow.Context,
	task workflowFunc,
	directive error,
	input any,
	state *utils.State,
	nextTargetName **string,
	fromTaskLevel bool,
) (stop bool, err error) {
	switch {
	case errors.Is(directive, flow.ErrContinue):
		return false, nil
	case errors.Is(directive, flow.ErrExit):
		return true, nil
	case errors.Is(directive, flow.ErrEnd):
		return false, flow.ErrEnd
	}

	var redirect flow.RedirectError
	if errors.As(directive, &redirect) {
		if fromTaskLevel {
			target := redirect.Target
			*nextTargetName = &target
			return false, nil
		}
		redirErr := t.executeRedirect(ctx, task, redirect.Target, input, state)
		// executeRedirect has already updated state.Output via the
		// originating task's output/export, both on a clean redirect
		// and on an end-propagation. Emit task.completed so consumers
		// see the switch task close with the effective output, even
		// when the redirect target signalled end. A genuine failure
		// from the redirect target skips the completion event because
		// the task did not complete cleanly.
		if redirErr == nil || errors.Is(redirErr, flow.ErrEnd) {
			t.emitTaskCompleted(ctx, task, input, state.Output, state)
		}
		return false, redirErr
	}

	return false, directive
}

// executeRedirect dispatches a named flow directive target. Zigflow's
// execution model represents redirect targets as registered child
// workflows, so we execute the target by name and feed the result
// through the originating task's output and export processing.
//
// If the child workflow signalled `then: end` it terminates by returning
// the Temporal ApplicationError minted by flow.NewEndApplicationError.
// We decode that error, extract the child's effective output from the
// attached payload, and feed it through the originating task's
// output/export directives so the redirect's contribution to
// state.Output is preserved even though the child terminated. We then
// return flow.ErrEnd so the surrounding dispatch continues to propagate
// "end" upward toward the root workflow.
func (t *DoTaskBuilder) executeRedirect(
	ctx workflow.Context, task workflowFunc, target string, input any, state *utils.State,
) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Redirecting to named task as child workflow",
		"task", task.Name, "target", target)

	var res any
	err := workflow.ExecuteChildWorkflow(ctx, target, input, state).Get(ctx, &res)
	if err != nil {
		if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
			logger.Info("Redirect target signalled end; propagating to caller",
				"target", target, "carriedOutput", endPayload.Output)
			// Treat the carried output as the child's effective result
			// so the originating task's output/export still apply and
			// state.Output reflects the work the child did before it
			// signalled end.
			if perr := t.applyRedirectResult(task, endPayload.Output, state); perr != nil {
				return perr
			}
			return flow.ErrEnd
		}
		logger.Error("Error executing redirect target", "target", target, "error", err)
		return err
	}

	return t.applyRedirectResult(task, res, state)
}

// applyRedirectResult feeds a redirect target's result through the
// originating task's output and export directives. controlOnly is
// always false here: a redirect target produced an actual result
// (either returned cleanly or surfaced through an end-signal payload)
// and the default state.Output assignment is what callers expect.
func (t *DoTaskBuilder) applyRedirectResult(task workflowFunc, res any, state *utils.State) error {
	if _, err := t.processTaskOutput(task, res, state, false); err != nil {
		return fmt.Errorf("error processing task output: %w", err)
	}
	if err := t.processTaskExport(task, res, state); err != nil {
		return fmt.Errorf("error processing task export: %w", err)
	}
	return nil
}

// isRootWorkflow reports whether ctx belongs to the topmost workflow in
// the current execution. Child workflows started via
// workflow.ExecuteChildWorkflow (redirect targets, for-loop iteration
// bodies, etc.) have ParentWorkflowExecution set; the original workflow
// started by the client does not.
func isRootWorkflow(ctx workflow.Context) bool {
	return workflow.GetInfo(ctx).ParentWorkflowExecution == nil
}

// processTaskExport evaluates the task's optional export directive
// against taskOutput and stores the result in state.Context. If the
// task did not define an export directive nothing is updated, which
// matches the behaviour of a task that simply did not export.
func (t *DoTaskBuilder) processTaskExport(task workflowFunc, taskOutput any, state *utils.State) error {
	taskBase := task.GetTask().GetBase()

	if taskBase.Export == nil {
		return nil
	}

	export, err := utils.TraverseAndEvaluateObj(taskBase.Export.As, taskOutput, state)
	if err != nil {
		return err
	}

	if err := swUtil.ValidateSchema(export, taskBase.Export.Schema, task.Name); err != nil {
		return err
	}

	state.Context = export

	return nil
}

// processTaskOutput evaluates the task's optional output directive
// against taskOutput and stores the result in state.Output. When no
// output directive is set the default behaviour is to record taskOutput
// as state.Output, so each subsequent task observes the previous one's
// value.
//
// The returned value is the effective output for this task: the value
// recorded on state.Output. Callers use it as the payload for the
// task.completed event so consumers see the transformed output rather
// than the raw task result.
//
// controlOnly is set for tasks that returned a flow-control directive
// (e.g. a switch). Such tasks do not produce a meaningful payload of
// their own, so without an explicit output directive we must leave
// state.Output alone rather than wiping the prior task's output with
// the nil placeholder a control task returns. An explicit output
// expression still fires so users can shape state.Output from $context
// even on a routing task; in that case the effective output is the
// evaluated expression. With no directive and controlOnly=true the
// effective output is nil because the task contributed nothing.
func (t *DoTaskBuilder) processTaskOutput(task workflowFunc, taskOutput any, state *utils.State, controlOnly bool) (any, error) {
	taskBase := task.GetTask().GetBase()

	if taskBase.Output == nil {
		if controlOnly {
			return nil, nil
		}
		state.Output = taskOutput
		return taskOutput, nil
	}

	output, err := utils.TraverseAndEvaluateObj(taskBase.Output.As, taskOutput, state)
	if err != nil {
		return nil, err
	}

	if err := swUtil.ValidateSchema(output, taskBase.Output.Schema, task.Name); err != nil {
		return nil, err
	}

	state.Output = output

	return output, nil
}

// taskRunResult is the structured outcome of running a single task.
//
// Exactly one of the fields is meaningful per result:
//   - Cancelled: Temporal cancelled the task. The cancellation event has
//     already been emitted; the caller must skip the completion
//     pipeline (no output, no export, no task.completed) so a cancelled
//     task is never indistinguishable from a successful one.
//   - Directive: a flow-control directive emitted by the task (switch
//     continue/exit/end or a RedirectError). The caller still runs the
//     completion pipeline because flow-directive-emitting tasks are
//     real, successful task executions that happen to also route.
//   - Output: a regular successful result. The caller processes
//     output/export and emits task.completed normally.
type taskRunResult struct {
	Output    any
	Directive error
	Cancelled bool
}

// runTask executes the task function and reports its outcome. It emits
// task.started, task.retried, task.cancelled and task.faulted events
// itself but deliberately does NOT process output/export or emit
// task.completed; the caller (runTaskAndHandleFlow) handles those so
// the same code path covers both regular and flow-directive-emitting
// tasks.
//
// The returned err is a genuine task failure: real Go-level errors
// that should be surfaced as workflow failures. Cancellation and flow
// directives are reported through taskRunResult instead so the caller
// can branch on them without conflating them with failures.
func (t *DoTaskBuilder) runTask(
	ctx workflow.Context, task workflowFunc, input any, state *utils.State,
) (taskRunResult, error) {
	logger := workflow.GetLogger(ctx)

	cctx := context.Background()
	info := workflow.GetInfo(ctx)
	workflowID := info.WorkflowExecution.ID

	t.eventEmitter.Emit(cctx, "task.started", func(e *ceSDK.Event) {
		e.SetID(workflowID)
		e.SetSubject(task.Name)
		_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
			"attempt":     info.Attempt,
			constKeyInput: input,
			constKeyState: state,
		})
	})

	if a := info.Attempt; a > 1 {
		t.eventEmitter.Emit(cctx, "task.retried", func(e *ceSDK.Event) {
			e.SetID(workflowID)
			e.SetSubject(task.Name)
			_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
				"attempt": a,
			})
		})
	}

	logger.Info("Running task", "name", task.Name)
	res, fnErr := task.Func(ctx, input, state)
	if fnErr != nil {
		if temporal.IsCanceledError(fnErr) {
			logger.Debug("Task cancelled", "name", task.Name)
			t.eventEmitter.Emit(cctx, "task.cancelled", func(e *ceSDK.Event) {
				e.SetID(workflowID)
				e.SetSubject(task.Name)
			})
			// Cancellation is reported explicitly so the caller can
			// skip the completion pipeline. Returning {} with a real
			// error here would be wrong because cancellation isn't a
			// task failure; returning success would be wrong because
			// the task did not actually complete.
			return taskRunResult{Cancelled: true}, nil
		}

		// Flow control errors are internal signals, not user-facing
		// failures. Hand them back to the caller without emitting a
		// fault event so the caller can run the normal completion
		// pipeline before dispatching the directive.
		if flow.IsControlError(fnErr) {
			logger.Debug("Task signalled a flow directive", "name", task.Name, "directive", fnErr.Error())
			return taskRunResult{Output: res, Directive: fnErr}, nil
		}

		t.eventEmitter.Emit(cctx, "task.faulted", func(e *ceSDK.Event) {
			e.SetID(workflowID)
			e.SetSubject(task.Name)
			_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
				"error": fnErr.Error(),
			})
		})

		logger.Error("Error running task", "name", task.Name, "error", fnErr)
		return taskRunResult{}, fnErr
	}

	return taskRunResult{Output: res}, nil
}

// emitTaskCompleted fires the task.completed cloudevent. It is invoked
// after output/export processing so the emitted "output" reflects the
// value used to update state.Output (for redirects this is the redirect
// target's result, not the switch task's nil placeholder).
func (t *DoTaskBuilder) emitTaskCompleted(
	ctx workflow.Context,
	task workflowFunc,
	input, output any,
	state *utils.State,
) {
	info := workflow.GetInfo(ctx)
	workflowID := info.WorkflowExecution.ID

	t.eventEmitter.Emit(context.Background(), "task.completed", func(e *ceSDK.Event) {
		e.SetID(workflowID)
		e.SetSubject(task.Name)
		_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
			constKeyInput: input,
			"output":      output,
			constKeyState: state,
		})
	})
}

func (t *DoTaskBuilder) shouldContinueAsNew(ctx workflow.Context) bool {
	logger := workflow.GetLogger(ctx)
	info := workflow.GetInfo(ctx)

	currentHistoryLength := info.GetCurrentHistoryLength()
	isSuggested := info.GetContinueAsNewSuggested()

	logger.Debug(
		"Checking continue-as-new state",
		"suggested", isSuggested,
		"maxHistoryOverride", t.opts.MaxHistoryLength,
		"currentHistoryLength", currentHistoryLength,
	)

	// Temporal is suggesting CAN
	if isSuggested {
		return true
	}

	// We've overridden for testing purposes
	if t.opts.MaxHistoryLength > 0 && currentHistoryLength > t.opts.MaxHistoryLength {
		return true
	}

	return false
}

func (t *DoTaskBuilder) shouldSkip(taskID string, task workflowFunc, state *utils.State) bool {
	if task.NeverSkipCAN() {
		// Task should never be skipped - eg a query listener which needs to be reinitialised
		return false
	}
	if targetID := state.CANStartFrom; targetID != nil {
		// We've reached the task we stopped on - continue from here
		shouldSkip := *targetID != taskID

		if !shouldSkip {
			// Matches - stop skipping skip anything else
			state.CANStartFrom = nil
		}

		return shouldSkip
	}

	return false
}

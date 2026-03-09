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
	"fmt"

	ceSDK "github.com/cloudevents/sdk-go/v2"
	"github.com/rs/zerolog/log"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/utils"
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
		builder, err := NewTaskBuilder(task.Key, task.Task, t.temporalWorker, t.doc, t.eventEmitter)
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
		builder, err := NewTaskBuilder(task.Key, task.Task, t.temporalWorker, t.doc, t.eventEmitter)
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
				"input": input,
				"state": state,
			})
		})

		// Iterate through the tasks to create the workflow
		if err := t.iterateTasks(ctx, tasks, input, state); err != nil {
			return nil, err
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

		if err := t.runTask(ctx, task, input, state); err != nil {
			return err
		}

		next, terminate := t.handleFlowDirective(ctx, taskBase)
		if terminate {
			break
		}
		if next != nil {
			nextTargetName = next
			continue
		}
	}

	if nextTargetName != nil {
		logger.Error("Next target specified but not found", "targetTask", nextTargetName)
		return fmt.Errorf("next target specified but not found: %s", *nextTargetName)
	}

	return nil
}

func (t *DoTaskBuilder) handleFlowDirective(
	ctx workflow.Context, taskBase *model.TaskBase,
) (next *string, terminate bool) {
	logger := workflow.GetLogger(ctx)

	if then := taskBase.Then; then != nil {
		flowDirective := then.Value
		if then.IsTermination() {
			logger.Debug("Workflow to be terminated", "flow", flowDirective)
			terminate = true
		} else if !then.IsEnum() {
			logger.Debug("Next task targeted", "nextTask", flowDirective)
			next = &flowDirective
		}
	}

	return
}

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

func (t *DoTaskBuilder) processTaskOutput(task workflowFunc, taskOutput any, state *utils.State) error {
	taskBase := task.GetTask().GetBase()

	if taskBase.Output == nil {
		state.Output = taskOutput
		return nil
	}

	output, err := utils.TraverseAndEvaluateObj(taskBase.Output.As, taskOutput, state)
	if err != nil {
		return err
	}

	if err := swUtil.ValidateSchema(output, taskBase.Output.Schema, task.Name); err != nil {
		return err
	}

	state.Output = output

	return nil
}

func (t *DoTaskBuilder) runTask(ctx workflow.Context, task workflowFunc, input any, state *utils.State) error {
	logger := workflow.GetLogger(ctx)

	cctx := context.Background()
	info := workflow.GetInfo(ctx)
	workflowID := info.WorkflowExecution.ID

	t.eventEmitter.Emit(cctx, "task.started", func(e *ceSDK.Event) {
		e.SetID(workflowID)
		e.SetSubject(task.Name)
		_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
			"attempt": info.Attempt,
			"input":   input,
			"state":   state,
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
	output, err := task.Func(ctx, input, state)
	if err != nil {
		if temporal.IsCanceledError(err) {
			logger.Debug("Task cancelled", "name", task.Name)
			t.eventEmitter.Emit(cctx, "task.cancelled", func(e *ceSDK.Event) {
				e.SetID(workflowID)
				e.SetSubject(task.Name)
			})
			return nil
		}

		t.eventEmitter.Emit(cctx, "task.faulted", func(e *ceSDK.Event) {
			e.SetID(workflowID)
			e.SetSubject(task.Name)
			_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
				"error": err.Error(),
			})
		})

		logger.Error("Error running task", "name", task.Name, "error", err)
		return err
	}

	// Set the output
	if err := t.processTaskOutput(task, output, state); err != nil {
		return fmt.Errorf("error processing task output: %w", err)
	}

	// Set the export
	if err := t.processTaskExport(task, output, state); err != nil {
		return fmt.Errorf("error processing task export: %w", err)
	}

	t.eventEmitter.Emit(cctx, "task.completed", func(e *ceSDK.Event) {
		e.SetID(workflowID)
		e.SetSubject(task.Name)
		_ = e.SetData(ceSDK.ApplicationJSON, map[string]any{
			"input":  input,
			"output": output,
			"state":  state,
		})
	})

	return nil
}

func (t *DoTaskBuilder) shouldContinueAsNew(ctx workflow.Context) bool {
	logger := workflow.GetLogger(ctx)
	info := workflow.GetInfo(ctx)

	currentHistoryLength := info.GetCurrentHistoryLength()
	isSuggested := info.GetContinueAsNewSuggested()

	logger.Debug("Checking continue-as-new state",
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

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
	"maps"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewForkTaskBuilder(
	temporalWorker worker.Worker,
	task *model.ForkTask,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*ForkTaskBuilder, error) {
	return &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type ForkTaskBuilder struct {
	builder[*model.ForkTask]
}

type forkedTask struct {
	task              *model.TaskItem
	childWorkflowName string
	taskName          string
}

func (t *ForkTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	forkedTasks, builders, err := t.buildOrPostLoad()
	if err != nil {
		return nil, err
	}

	for _, builder := range builders {
		if _, err := builder.Build(); err != nil {
			log.Error().Err(err).Msg("Error building forked workflow")
			return nil, fmt.Errorf("error building forked workflow: %w", err)
		}
	}

	return t.exec(forkedTasks)
}

func (t *ForkTaskBuilder) PostLoad() error {
	_, builders, err := t.buildOrPostLoad()
	if err != nil {
		return err
	}

	for _, builder := range builders {
		if err := builder.PostLoad(); err != nil {
			log.Error().Err(err).Msg("Error post loading forked workflow")
			return fmt.Errorf("error post loading forked workflow: %w", err)
		}
	}

	return nil
}

func (t *ForkTaskBuilder) Validate() error {
	_, builders, err := t.buildOrPostLoad()
	if err != nil {
		return err
	}

	for _, builder := range builders {
		if err := builder.Validate(); err != nil {
			return fmt.Errorf("error validating forked workflow: %w", err)
		}
	}

	return nil
}

func (t *ForkTaskBuilder) awaitCondition(
	replyErr error, endSeen bool, isCompeting bool, winningCtx workflow.Context, hasReplied []bool,
) func() bool {
	return func() bool {
		// Any short-circuit reason ends the await: a genuine failure
		// from a branch, or a Zigflow end signal from a branch.
		if replyErr != nil || endSeen {
			return true
		}

		predicate := func(v bool) bool { return v }

		if isCompeting {
			return winningCtx != nil
		}

		return utils.SliceEvery(hasReplied, predicate)
	}
}

func (t *ForkTaskBuilder) buildOrPostLoad() ([]*forkedTask, []TaskBuilder, error) {
	forkedTasks := make([]*forkedTask, 0)
	builders := make([]TaskBuilder, 0)

	for _, branch := range *t.task.Fork.Branches {
		childWorkflowName := utils.GenerateChildWorkflowName("fork", t.GetTaskName(), branch.Key)

		forkedTasks = append(forkedTasks, &forkedTask{
			task:              branch,
			childWorkflowName: childWorkflowName,
			taskName:          branch.Key,
		})

		if d := branch.AsDoTask(); d == nil {
			// Single task - register this as a single task workflow
			log.Debug().Str("task", branch.Key).Msg("Registering single task workflow")
			branch = &model.TaskItem{
				Key: childWorkflowName,
				Task: &model.DoTask{
					Do: &model.TaskList{branch},
				},
			}
		}

		builder, err := NewTaskBuilder(childWorkflowName, branch.Task, t.temporalWorker, t.doc, t.eventEmitter, t.taskOpts)
		if err != nil {
			log.Error().Err(err).Msg("Error creating the forked task builder")
			return nil, nil, fmt.Errorf("error creating the forked task builder: %w", err)
		}

		builders = append(builders, builder)
	}

	return forkedTasks, builders, nil
}

func (t *ForkTaskBuilder) exec(forkedTasks []*forkedTask) (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		isCompeting := t.task.Fork.Compete

		logger := workflow.GetLogger(ctx)
		logger.Debug("Forking a task", "isCompeting", isCompeting)

		futures := &utils.CancellableFutures{}

		// Create a new state with no output to pass to the children
		childState := state.Clone().ClearOutput()
		output := map[string]any{}

		// Run the child workflows in parallel
		for _, branch := range forkedTasks {
			opts := workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("%s_fork_%s", workflow.GetInfo(ctx).WorkflowExecution.ID, branch.task.Key),
			}
			if isCompeting {
				// Allow cancellation of children
				opts.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_REQUEST_CANCEL
			}

			childCtx := workflow.WithChildOptions(ctx, opts)
			childCtx, cancelHandler := workflow.WithCancel(childCtx)

			logger.Info("Triggering forked child workflow", "name", branch.childWorkflowName)

			futures.Add(branch.taskName, utils.CancellableFuture{
				Cancel:  cancelHandler,
				Context: childCtx,
				Future:  workflow.ExecuteChildWorkflow(childCtx, branch.childWorkflowName, input, childState),
			})
		}

		// Now they're running, wait for the results.
		fs := &forkState{
			isCompeting: isCompeting,
			output:      output,
			hasReplied:  make([]bool, futures.Length()),
		}

		i := 0
		for taskName, w := range futures.List() {
			// Get the replies in parallel as the "winner" may be last
			workflow.Go(w.Context, func(ctx workflow.Context) {
				var childData map[string]any
				err := w.Future.Get(ctx, &childData)
				if t.handleBranchError(ctx, logger, taskName, err, fs) {
					return
				}
				fs.recordReply(ctx, logger, taskName, childData, i)
				i++
			})
		}

		// Wait for the concurrent tasks to complete
		if err := workflow.Await(ctx, func() bool {
			// Wrap the function so the values are updated each time it's triggered
			return t.awaitCondition(fs.replyErr, fs.endSeen, isCompeting, fs.winningCtx, fs.hasReplied)()
		}); err != nil {
			logger.Error("Error waiting for forked tasks to complete", "error", err)
			return nil, fmt.Errorf("error waiting for forked tasks to complete: %w", err)
		}

		logger.Debug("Forked task has completed")

		if fs.replyErr != nil {
			return nil, fs.replyErr
		}

		// End takes precedence after replyErr because a fork branch that
		// signalled end means "terminate the whole workflow". Surface
		// flow.ErrEnd to the do-task pipeline carrying the branch's
		// effective output so the originating fork task's output and
		// state still update before end keeps propagating upward.
		if fs.endSeen {
			return fs.endOutput, flow.ErrEnd
		}

		if isCompeting {
			logger.Debug("Cancelling other forked workflows")
			futures.CancelOthers(fs.winningCtx)
		}

		return output, nil
	}, nil
}

// forkState bundles the mutable bookkeeping shared by all fork branch
// goroutines so that the per-branch handling code can live outside of
// exec's main closure. Methods on forkState are only ever called from
// inside a workflow.Go goroutine that already holds the workflow
// determinism guarantees, so plain mutation is safe.
type forkState struct {
	isCompeting bool

	hasReplied []bool

	// output is the shared aggregate map across non-competing branches
	// (and the winning branch in competing mode). It is the same map
	// returned by exec, mutated in place.
	output map[string]any

	// winningCtx records the first branch to reply in competing mode.
	winningCtx workflow.Context

	// replyErr captures a genuine branch failure that should fail the
	// whole fork. endSeen + endOutput capture a Zigflow end signal,
	// which terminates the workflow cleanly instead of failing it.
	replyErr  error
	endSeen   bool
	endOutput any
}

// handleBranchError interprets the per-branch future error and updates
// fs accordingly. It returns true if the branch should stop here and
// not contribute to the aggregate output.
func (t *ForkTaskBuilder) handleBranchError(
	ctx workflow.Context,
	logger interface {
		Debug(msg string, keyvals ...interface{})
		Info(msg string, keyvals ...interface{})
		Error(msg string, keyvals ...interface{})
	},
	taskName string,
	err error,
	fs *forkState,
) bool {
	_ = ctx
	if err == nil {
		return false
	}

	if temporal.IsCanceledError(err) {
		logger.Debug("Forked task cancelled", "task", taskName)
		return true
	}

	// A `then: end` directive inside a fork branch crosses the child
	// workflow boundary as a typed Temporal ApplicationError. That is
	// a deliberate workflow termination, not a "fork failure":
	// short-circuit the await without wrapping it as "error forking
	// task" and carry the branch's effective output upward.
	if endPayload, isEnd := flow.DecodeEndApplicationError(err); isEnd {
		logger.Info("Fork branch signalled end; propagating",
			"task", taskName, "carriedOutput", endPayload.Output)
		if !fs.endSeen {
			fs.endSeen = true
			fs.endOutput = endPayload.Output
		}
		return true
	}

	logger.Error("Error forking task", "error", err, "task", taskName)
	fs.replyErr = fmt.Errorf("error forking task: %w", err)
	return false
}

// recordReply records a successful branch reply into the shared output
// state. In competing mode the first reply wins and only its data is
// kept; in non-competing mode every branch contributes under its task
// name.
func (fs *forkState) recordReply(
	ctx workflow.Context,
	logger interface {
		Debug(msg string, keyvals ...interface{})
	},
	taskName string,
	childData map[string]any,
	i int,
) {
	fs.hasReplied[i] = true

	addData := !fs.isCompeting
	if fs.isCompeting && fs.winningCtx == nil {
		logger.Debug(
			"Winner declared",
			"childWorkflowID",
			workflow.GetChildWorkflowOptions(ctx).WorkflowID,
		)
		addData = true
		fs.winningCtx = ctx
	}

	if !addData {
		return
	}

	data := childData
	if !fs.isCompeting {
		data = map[string]any{taskName: childData}
	}
	maps.Copy(fs.output, data)
}

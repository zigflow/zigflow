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
) (*ForkTaskBuilder, error) {
	return &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           task,
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

func (t *ForkTaskBuilder) awaitCondition(
	replyErr error, isCompeting bool, winningCtx workflow.Context, hasReplied []bool,
) func() bool {
	return func() bool {
		if replyErr != nil {
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

		builder, err := NewTaskBuilder(childWorkflowName, branch.Task, t.temporalWorker, t.doc, t.eventEmitter)
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

		// Now they're running, wait for the results
		var replyErr error
		hasReplied := make([]bool, futures.Length())
		var winningCtx workflow.Context

		i := 0
		for taskName, w := range futures.List() {
			// Get the replies in parallel as the "winner" may be last
			workflow.Go(w.Context, func(ctx workflow.Context) {
				var childData map[string]any
				if err := w.Future.Get(ctx, &childData); err != nil {
					if temporal.IsCanceledError(err) {
						logger.Debug("Forked task cancelled", "task", taskName)
						return
					}

					logger.Error("Error forking task", "error", err, "task", taskName)
					replyErr = fmt.Errorf("error forking task: %w", err)
				}

				hasReplied[i] = true

				// Always add non-competing data to the output
				addData := !isCompeting
				if isCompeting && winningCtx == nil {
					logger.Debug(
						"Winner declared",
						"childWorkflowID",
						workflow.GetChildWorkflowOptions(ctx).WorkflowID,
					)

					// We only care about the winning data
					addData = true
					winningCtx = ctx
				}

				if addData {
					data := childData
					if !isCompeting {
						data = map[string]any{
							taskName: childData,
						}
					}
					maps.Copy(output, data)
				}

				i++
			})
		}

		// Wait for the concurrent tasks to complete
		if err := workflow.Await(ctx, func() bool {
			// Wrap the function so the values are updated each time it's triggered
			return t.awaitCondition(replyErr, isCompeting, winningCtx, hasReplied)()
		}); err != nil {
			logger.Error("Error waiting for forked tasks to complete", "error", err)
			return nil, fmt.Errorf("error waiting for forked tasks to complete: %w", err)
		}

		logger.Debug("Forked task has completed")

		if replyErr != nil {
			return nil, replyErr
		}

		if isCompeting {
			logger.Debug("Cancelling other forked workflows")
			futures.CancelOthers(winningCtx)
		}

		return output, nil
	}, nil
}

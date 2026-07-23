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
	sdklog "go.temporal.io/sdk/log"
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

// forkBranch is a single fork branch built once as an inline function.
type forkBranch struct {
	name string
	fn   TemporalWorkflowFunc
}

// forkBranchResult is the self-contained value each branch goroutine sends back
// to the parent workflow goroutine. Branches never write to shared state; the
// parent is the sole owner of ordering, aggregation and error selection.
//
// Only Output is carried back: fork aggregates branch outputs and never
// promotes a branch's Context or Data to the parent (matching the previous
// child-workflow contract, which returned only state.Output).
type forkBranchResult struct {
	index  int
	name   string
	output any
	err    error
}

func (t *ForkTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	branches, err := t.buildBranches()
	if err != nil {
		return nil, err
	}

	return t.exec(branches)
}

func (t *ForkTaskBuilder) PostLoad() error {
	for _, branch := range *t.task.Fork.Branches {
		builder, err := t.branchBuilder(branch)
		if err != nil {
			return err
		}

		if err := builder.PostLoad(); err != nil {
			log.Error().Err(err).Str("branch", branch.Key).Msg("Error post-loading fork branch")
			return fmt.Errorf("error post loading fork branch %q: %w", branch.Key, err)
		}
	}

	return nil
}

func (t *ForkTaskBuilder) Validate() error {
	for _, branch := range *t.task.Fork.Branches {
		builder, err := t.branchBuilder(branch)
		if err != nil {
			return err
		}

		if err := builder.Validate(); err != nil {
			return fmt.Errorf("error validating fork branch %q: %w", branch.Key, err)
		}
	}

	return nil
}

// branchBuilder constructs the inline DoTaskBuilder for one fork branch without
// registering a workflow, threading a unique task path so leaf tasks that reuse
// a name across sibling branches do not collide on a per-task activity alias.
//
//   - A multi-task branch is a do-task scope: the branch key is an intermediate
//     path segment and the body's leaf tasks nest beneath it.
//   - A single-task branch is wrapped so it runs as an inline do-task; the
//     branch key stays the task's own leaf segment, so the parent path is
//     threaded to avoid duplicating it.
func (t *ForkTaskBuilder) branchBuilder(branch *model.TaskItem) (*DoTaskBuilder, error) {
	var (
		taskList *model.TaskList
		taskPath []string
	)

	if d := branch.AsDoTask(); d != nil {
		taskList = d.Do
		taskPath = t.childTaskPath(branch.Key)
	} else {
		taskList = &model.TaskList{branch}
		taskPath = t.taskPath
	}

	builder, err := newInlineDoBuilder(t.temporalWorker, taskList, "", t.doc, t.eventEmitter, t.taskOpts, taskPath)
	if err != nil {
		return nil, fmt.Errorf("error creating fork branch builder %q: %w", branch.Key, err)
	}

	return builder, nil
}

// buildBranches builds every branch body once into an inline TemporalWorkflowFunc.
func (t *ForkTaskBuilder) buildBranches() ([]forkBranch, error) {
	branches := make([]forkBranch, 0, len(*t.task.Fork.Branches))

	for _, branch := range *t.task.Fork.Branches {
		builder, err := t.branchBuilder(branch)
		if err != nil {
			return nil, err
		}

		fn, err := builder.Build()
		if err != nil {
			log.Error().Err(err).Str("branch", branch.Key).Msg("Error building fork branch")
			return nil, fmt.Errorf("error building fork branch %q: %w", branch.Key, err)
		}

		branches = append(branches, forkBranch{name: branch.Key, fn: fn})
	}

	return branches, nil
}

func (t *ForkTaskBuilder) exec(branches []forkBranch) (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		isCompeting := t.task.Fork.Compete

		logger := workflow.GetLogger(ctx)
		logger.Debug("Forking a task", "isCompeting", isCompeting, "branches", len(branches))

		n := len(branches)
		if n == 0 {
			// The schema requires at least one branch; guard defensively so an
			// empty fork is a no-op rather than a workflow that blocks forever
			// waiting on a channel that never receives.
			return map[string]any{}, nil
		}

		// A buffered channel sized to the branch count means a branch send never
		// blocks, so cancelled losers can always complete without deadlocking.
		results := workflow.NewBufferedChannel(ctx, n)
		cancels := make([]workflow.CancelFunc, n)

		for idx, branch := range branches {
			// Each branch runs in its own cancellable context against its own
			// isolated state clone, so no branch can observe or corrupt another
			// branch's (or the parent's) state.
			branchCtx, cancel := workflow.WithCancel(ctx)
			cancels[idx] = cancel

			branchState := state.Clone().ClearOutput()

			index, name, fn := idx, branch.name, branch.fn
			logger.Info("Running fork branch inline", "branch", name)

			workflow.Go(branchCtx, func(gctx workflow.Context) {
				output, err := fn(gctx, input, branchState)
				results.Send(gctx, forkBranchResult{index: index, name: name, output: output, err: err})
			})
		}

		if isCompeting {
			// Competing forks are a race: the first branch to complete wins.
			// This is deliberately completion-order sensitive (it is the
			// meaning of compete) but remains deterministic on replay because
			// Temporal records completion order in history.
			var res forkBranchResult
			results.Receive(ctx, &res)

			// Cancel every other branch so losing in-flight work stops.
			for i, cancel := range cancels {
				if i != res.index {
					cancel()
				}
			}

			return t.resolveWinner(logger, res)
		}

		// Non-competing forks wait for every branch, then select the outcome by
		// declaration index. This is deterministic regardless of completion
		// order (see the compatibility note in the package refactor).
		collected := make([]forkBranchResult, n)
		for range n {
			var res forkBranchResult
			results.Receive(ctx, &res)
			collected[res.index] = res
		}

		return t.aggregate(logger, collected)
	}, nil
}

// resolveWinner interprets the single winning branch of a competing fork.
func (t *ForkTaskBuilder) resolveWinner(logger sdklog.Logger, res forkBranchResult) (any, error) {
	isEnd, endOutput, genuine := classifyForkBranch(res)
	if genuine != nil {
		logger.Error("Error running fork branch tasks", "error", genuine, "branch", res.name)
		return nil, fmt.Errorf("error running fork branch tasks (%s): %w", res.name, genuine)
	}
	if isEnd {
		logger.Info("Fork branch signalled end; propagating", "branch", res.name)
		return endOutput, flow.ErrEnd
	}

	return res.output, nil
}

// aggregate resolves a completed set of non-competing branch results.
//
// Precedence, matching the previous contract (a genuine failure outranks an end
// directive) but made deterministic by declaration index rather than completion
// order:
//  1. lowest-index genuine failure fails the whole fork;
//  2. otherwise the lowest-index end directive terminates the workflow,
//     carrying that branch's effective output;
//  3. otherwise every branch output is aggregated under its branch name.
func (t *ForkTaskBuilder) aggregate(logger sdklog.Logger, collected []forkBranchResult) (any, error) {
	for _, res := range collected {
		if _, _, genuine := classifyForkBranch(res); genuine != nil {
			logger.Error("Error running fork branch tasks", "error", genuine, "branch", res.name)
			return nil, fmt.Errorf("error running fork branch tasks (%s): %w", res.name, genuine)
		}
	}

	for _, res := range collected {
		if isEnd, endOutput, _ := classifyForkBranch(res); isEnd {
			logger.Info("Fork branch signalled end; propagating", "branch", res.name)
			return endOutput, flow.ErrEnd
		}
	}

	output := make(map[string]any, len(collected))
	for _, res := range collected {
		output[res.name] = res.output
	}

	return output, nil
}

// classifyForkBranch interprets a branch result's error. It reports whether the
// branch signalled flow.ErrEnd (with the effective output it carried) or failed
// for a genuine reason.
//
// The encoded end error is decoded first so its carried payload output is not
// lost; a direct inline flow.ErrEnd is the primary path.
func classifyForkBranch(res forkBranchResult) (isEnd bool, endOutput any, genuine error) {
	if res.err == nil {
		return false, nil, nil
	}
	if payload, ok := flow.DecodeEndApplicationError(res.err); ok {
		return true, payload.Output, nil
	}
	if errors.Is(res.err, flow.ErrEnd) {
		return true, res.output, nil
	}

	return false, nil, res.err
}

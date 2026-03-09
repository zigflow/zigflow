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

	"github.com/rs/zerolog/log"
	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func ActivitiesList() []any {
	return activities.Registry
}

type TaskBuilder interface {
	Build() (TemporalWorkflowFunc, error)
	GetTask() model.Task
	GetTaskName() string
	NeverSkipCAN() bool
	ParseMetadata(workflow.Context, *utils.State) error
	PostLoad() error
	ShouldRun(*utils.State) (bool, error)
}

type TemporalWorkflowFunc func(ctx workflow.Context, input any, state *utils.State) (output any, err error)

type builder[T model.Task] struct {
	doc            *model.Workflow
	eventEmitter   *cloudevents.Events
	name           string
	neverSkipCAN   bool
	task           T
	temporalWorker worker.Worker
}

func (d *builder[T]) executeActivity(ctx workflow.Context, activity, input any, state *utils.State) (output any, err error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Calling activity", "name", d.name)

	var res any
	if err := workflow.ExecuteActivity(ctx, activity, d.task, input, state).Get(ctx, &res); err != nil {
		if temporal.IsCanceledError(err) {
			return nil, nil
		}

		logger.Error("Error calling activity", "name", d.name, "error", err)
		return nil, fmt.Errorf("error calling activity: %w", err)
	}

	// Add the result to the state's data
	logger.Debug("Setting data to the state", "key", d.name)
	state.AddData(map[string]any{
		d.name: res,
	})

	return res, nil
}

func (d *builder[T]) GetTask() model.Task {
	return d.task
}

func (d *builder[T]) GetTaskName() string {
	return d.name
}

// Some tasks should never be skipped when doing Continue-As-New
func (d *builder[T]) NeverSkipCAN() bool {
	return d.neverSkipCAN
}

func (d builder[T]) ParseMetadata(ctx workflow.Context, state *utils.State) error {
	logger := workflow.GetLogger(ctx)

	task := d.GetTask().GetBase()

	if len(task.Metadata) == 0 {
		// No metadata set - continue
		return nil
	}

	// Clone the metadata to avoid pollution
	mClone := swUtils.DeepClone(task.Metadata)

	parsed, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(mClone), nil, state)
	if err != nil {
		return fmt.Errorf("error interpolating metadata: %w", err)
	}

	if search, ok := parsed.(map[string]any)[metadata.MetadataSearchAttribute]; ok {
		logger.Debug("Parsing search attributes")
		if err := metadata.ParseSearchAttributes(ctx, search); err != nil {
			logger.Error("Error parsing search attributes", "attributes", search, "error", err)
			return fmt.Errorf("error parsing search attributes: %w", err)
		}
	}

	return nil
}

func (d *builder[T]) PostLoad() error {
	log.Trace().Str("task", d.GetTaskName()).Msg("Task has no post load hook")
	return nil
}

func (d *builder[T]) ShouldRun(state *utils.State) (bool, error) {
	return utils.CheckIfStatement(d.task.GetBase().If, state)
}

// Factory to create a TaskBuilder instance, or die trying
func NewTaskBuilder(
	taskName string,
	task model.Task,
	temporalWorker worker.Worker,
	doc *model.Workflow,
	emitter *cloudevents.Events,
) (TaskBuilder, error) {
	switch t := task.(type) {
	case *model.CallFunction:
		if t.Call == customCallFunctionActivity {
			return NewCallActivityTaskBuilder(temporalWorker, t, taskName, doc, emitter)
		}
		return nil, fmt.Errorf("unsupported call type '%s' for task '%s'", t.Call, taskName)
	case *model.CallGRPC:
		return NewCallGRPCTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.CallHTTP:
		return NewCallHTTPTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.DoTask:
		return NewDoTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.ForTask:
		return NewForTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.ForkTask:
		return NewForkTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.ListenTask:
		return NewListenTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.RaiseTask:
		return NewRaiseTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.RunTask:
		return NewRunTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.SetTask:
		return NewSetTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.SwitchTask:
		return NewSwitchTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.TryTask:
		return NewTryTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	case *model.WaitTask:
		return NewWaitTaskBuilder(temporalWorker, t, taskName, doc, emitter)
	default:
		return nil, fmt.Errorf("unsupported task type '%T' for task '%s'", t, taskName)
	}
}

// Ensure the tasks meets the TaskBuilder type
var (
	_ TaskBuilder = &CallActivityTaskBuilder{}
	_ TaskBuilder = &CallGRPCTaskBuilder{}
	_ TaskBuilder = &CallHTTPTaskBuilder{}
	_ TaskBuilder = &DoTaskBuilder{}
	_ TaskBuilder = &ForTaskBuilder{}
	_ TaskBuilder = &ForkTaskBuilder{}
	_ TaskBuilder = &ListenTaskBuilder{}
	_ TaskBuilder = &RaiseTaskBuilder{}
	_ TaskBuilder = &RunTaskBuilder{}
	_ TaskBuilder = &SetTaskBuilder{}
	_ TaskBuilder = &SwitchTaskBuilder{}
	_ TaskBuilder = &TryTaskBuilder{}
	_ TaskBuilder = &WaitTaskBuilder{}
)

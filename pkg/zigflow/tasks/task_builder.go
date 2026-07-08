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
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
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
	Validate() error
	PostLoad() error
	ShouldRun(*utils.State) (bool, error)
}

// Configure any task opts that come from the CLI
type TaskOpts struct {
	Run *RunTaskOpts
}

type TemporalWorkflowFunc func(ctx workflow.Context, input any, state *utils.State) (output any, err error)

type builder[T model.Task] struct {
	doc            *model.Workflow
	eventEmitter   *cloudevents.Events
	name           string
	neverSkipCAN   bool
	task           T
	taskOpts       *TaskOpts
	temporalWorker worker.Worker
	// Path from the workflow root; disambiguates per-task activity names
	// when sibling scopes reuse a leaf name.
	taskPath []string
}

func (d *builder[T]) perTaskActivityName() string {
	if d.doc == nil || d.doc.Document.Name == "" {
		return d.name
	}
	segments := []string{d.doc.Document.Name}
	if len(d.taskPath) == 0 {
		segments = append(segments, d.name)
	} else {
		segments = append(segments, d.taskPath...)
	}
	return joinActivityNameSegments(segments)
}

// joinActivityNameSegments builds a per-task activity name by escaping each
// segment and joining them with ".".
//
// A task key may legitimately contain dots (e.g. "a.b"). Joining raw
// segments with "." would make such a key indistinguishable from the
// separator, so two different task paths could collide on one activity name
// (["a.b", "c"] and ["a", "b.c"] would both yield "a.b.c"). Escaping makes
// the join unambiguous, so distinct task paths always produce distinct
// activity names and valid task keys are preserved rather than rejected.
func joinActivityNameSegments(segments []string) string {
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = escapeActivityNameSegment(segment)
	}
	return strings.Join(escaped, ".")
}

// escapeActivityNameSegment makes a single segment safe to join with "."
// into a per-task activity name. It escapes the escape character first and
// then the separator, so the encoding is reversible and injective: an
// unescaped "." in the joined name is always a separator, and a "\" always
// introduces a literal "\" or ".". Segments that contain neither character
// (the common case) are returned unchanged, keeping the clean
// "<workflowType>.<task>" form.
func escapeActivityNameSegment(segment string) string {
	segment = strings.ReplaceAll(segment, `\`, `\\`)
	return strings.ReplaceAll(segment, ".", `\.`)
}

func (d *builder[T]) setTaskPath(path []string) {
	d.taskPath = path
}

// slices.Concat allocates a fresh slice so sibling iterations do not
// share the parent's underlying array.
func (d *builder[T]) childTaskPath(name string) []string {
	return slices.Concat(d.taskPath, []string{name})
}

type taskPathSetter interface {
	setTaskPath([]string)
}

func (d *builder[T]) registerActivityForTask(fn any) string {
	name := d.perTaskActivityName()
	registerActivityOnce(d.temporalWorker, fn, name)
	return name
}

// Combines registerActivityForTask with the dispatch closure used by
// CallHTTP/CallGRPC. RunTaskBuilder picks between sub-activities and
// uses registerActivityForTask directly instead.
//
// legacyName is the fixed activity type name recorded by histories created
// before per-task aliases existed; dispatchActivityName decides at runtime
// which name to schedule so open executions keep replaying deterministically.
func (d *builder[T]) buildActivityFunc(fn any, legacyName string) TemporalWorkflowFunc {
	perTaskName := d.registerActivityForTask(fn)
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		return d.executeActivity(ctx, dispatchActivityName(ctx, legacyName, perTaskName), input, state)
	}
}

// dispatchActivityName selects the activity type to schedule, using Temporal
// workflow versioning so renaming activities to per-task aliases does not
// break open histories.
//
// On a new execution GetVersion records a marker and returns the current
// version, so the per-task alias is scheduled. On an execution that started
// before the change there is no marker, GetVersion returns
// workflow.DefaultVersion, and the legacy name the history already recorded
// is scheduled again, keeping the replay deterministic. The worker registers
// both names, so either resolves at runtime.
func dispatchActivityName(ctx workflow.Context, legacyName, perTaskName string) string {
	if workflow.GetVersion(
		ctx, activityNamingVersionChangeID, workflow.DefaultVersion, activityNamingVersion,
	) == workflow.DefaultVersion {
		return legacyName
	}
	return perTaskName
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

func (d *builder[T]) Validate() error {
	log.Trace().Str("task", d.GetTaskName()).Msg("Task has no validate hook")
	return nil
}

func (d *builder[T]) PostLoad() error {
	log.Trace().Str("task", d.GetTaskName()).Msg("Task has no post load hook")
	return nil
}

func (d *builder[T]) ShouldRun(state *utils.State) (bool, error) {
	return utils.CheckIfStatement(d.task.GetBase().If, state)
}

// Factory to create a TaskBuilder instance.
//
//nolint:gocyclo // in a factory a type-switch is a common pattern
func NewTaskBuilder(
	taskName string,
	task model.Task,
	temporalWorker worker.Worker,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
	taskPath []string,
) (TaskBuilder, error) {
	var b TaskBuilder
	var err error

	switch t := task.(type) {
	case *model.CallFunction:
		switch t.Call {
		case customCallFunctionActivity:
			b, err = NewCallActivityTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
		case customCallMCPActivity:
			b, err = NewCallMCPTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
		default:
			return nil, fmt.Errorf("unsupported call type '%s' for task '%s'", t.Call, taskName)
		}
	case *model.CallGRPC:
		b, err = NewCallGRPCTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.CallHTTP:
		b, err = NewCallHTTPTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.DoTask:
		b, err = NewDoTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.ForTask:
		b, err = NewForTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.ForkTask:
		b, err = NewForkTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.ListenTask:
		b, err = NewListenTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.RaiseTask:
		b, err = NewRaiseTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.RunTask:
		b, err = NewRunTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.SetTask:
		b, err = NewSetTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.SwitchTask:
		b, err = NewSwitchTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.TryTask:
		b, err = NewTryTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *model.WaitTask:
		b, err = NewWaitTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	case *models.WaitExtTask:
		b, err = NewWaitExtTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
	default:
		return nil, fmt.Errorf("unsupported task type '%T' for task '%s'", t, taskName)
	}

	if err != nil {
		return nil, err
	}
	if setter, ok := b.(taskPathSetter); ok && taskPath != nil {
		setter.setTaskPath(taskPath)
	}
	return b, nil
}

// Ensure the tasks meets the TaskBuilder type
var (
	_ TaskBuilder = &CallActivityTaskBuilder{}
	_ TaskBuilder = &CallGRPCTaskBuilder{}
	_ TaskBuilder = &CallHTTPTaskBuilder{}
	_ TaskBuilder = &CallMCPTaskBuilder{}
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
	_ TaskBuilder = &WaitExtTaskBuilder{}
)

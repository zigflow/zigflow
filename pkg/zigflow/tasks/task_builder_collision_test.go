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
	"testing"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
)

func TestNestedSameLeafNameRegistersDistinctActivities(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-collision"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-collision.first.step",
		}).
		Once()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-collision.second.step",
		}).
		Once()

	first, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, []string{"first", testConstStep})
	assert.NoError(t, err)
	second, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, []string{"second", testConstStep})
	assert.NoError(t, err)

	_, err = first.Build()
	assert.NoError(t, err)
	_, err = second.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

func TestTryCatchSameLeafNameRegistersDistinctActivities(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-try"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-try.outer.try.step",
		}).
		Once()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-try.outer.catch.step",
		}).
		Once()

	tryPath := []string{"outer", tryBodyPathSegment, testConstStep}
	catchPath := []string{"outer", catchBodyPathSegment, testConstStep}

	tryStep, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, tryPath)
	assert.NoError(t, err)
	catchStep, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, catchPath)
	assert.NoError(t, err)

	_, err = tryStep.Build()
	assert.NoError(t, err)
	_, err = catchStep.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

func TestForkSameLeafNameRegistersDistinctActivities(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fork"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork.dispatch.left.step",
		}).
		Once()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fork.dispatch.right.step",
		}).
		Once()

	leftPath := []string{"dispatch", "left", testConstStep}
	rightPath := []string{"dispatch", "right", testConstStep}

	left, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, leftPath)
	assert.NoError(t, err)
	right, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, rightPath)
	assert.NoError(t, err)

	_, err = left.Build()
	assert.NoError(t, err)
	_, err = right.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

func TestDeeplyNestedPathProducesFullyQualifiedName(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-deep"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-deep.l1.l2.l3.step",
		}).
		Once()

	path := []string{"l1", "l2", "l3", testConstStep}
	b, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, path)
	assert.NoError(t, err)

	_, err = b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

// Regression guard: ForTaskBuilder.Build constructs its inline DoBuilder via
// NewDoTaskBuilder directly, bypassing the factory. Without an explicit
// setTaskPath on that inner builder, the body's tasks register under
// collision-prone names, exactly the bug an e2e run with two sibling For loops
// uncovered.
func TestForBuildPropagatesTaskPathToInnerBody(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-for-build"}}

	w := new(WorkflowRegistryMock)
	w.On("RegisterWorkflowWithOptions", mock.Anything, mock.Anything).Maybe()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-for-build.outer_loop.step",
		}).
		Once()

	b := &ForTaskBuilder{
		builder: builder[*model.ForTask]{
			doc:            doc,
			eventEmitter:   testEvents,
			name:           "outer_loop",
			taskPath:       []string{"outer_loop"},
			temporalWorker: w,
			task: &model.ForTask{
				For: model.ForTaskConfiguration{In: "[]"},
				Do: &model.TaskList{
					&model.TaskItem{Key: testConstStep, Task: newTestHTTPTask()},
				},
			},
		},
	}

	_, err := b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

// Dotted task keys must not collide once segments are escaped. Under naive
// dot-joining []string{"a.b", "c"} and []string{"a", "b.c"} both produce
// "a.b.c"; escaping the dot inside a segment keeps them distinct.
func TestActivityNameSegmentsEscapeDottedKeys(t *testing.T) {
	const wf = "wf"
	first := joinActivityNameSegments([]string{wf, "a.b", "c"})
	second := joinActivityNameSegments([]string{wf, "a", "b.c"})

	assert.Equal(t, `wf.a\.b.c`, first)
	assert.Equal(t, `wf.a.b\.c`, second)
	assert.NotEqual(t, first, second, "dotted task keys must not collide on one activity name")
}

// Segments without dots or backslashes are joined verbatim so the common
// case keeps the readable <workflowType>.<task> form.
func TestActivityNameSegmentsPreserveCleanCommonCase(t *testing.T) {
	const wf = "wf"
	assert.Equal(t, "wf.fetchData", joinActivityNameSegments([]string{wf, "fetchData"}))
	assert.Equal(t, "wf.group.sub.step", joinActivityNameSegments([]string{wf, "group", "sub", "step"}))
}

// End-to-end guard: two task paths that collide under naive dot-joining
// must register two distinct per-task activity names. If escaping were
// dropped both builds would register "wf-dotted.a.b.c"; the second
// registration would dedup against the first and the second expectation
// would go unmet.
func TestDottedTaskPathsRegisterDistinctActivities(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-dotted"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: `wf-dotted.a\.b.c`,
		}).
		Once()
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: `wf-dotted.a.b\.c`,
		}).
		Once()

	first, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, []string{"a.b", "c"})
	assert.NoError(t, err)
	second, err := NewTaskBuilder(testConstStep, newTestHTTPTask(), w, doc, testEvents, nil, []string{"a", "b.c"})
	assert.NoError(t, err)

	_, err = first.Build()
	assert.NoError(t, err)
	_, err = second.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

// Concrete constructors used directly by tests do not set taskPath;
// perTaskActivityName falls back to "<workflowType>.<taskName>".
func TestNilTaskPathFallsBackToBareTaskName(t *testing.T) {
	doc := &model.Workflow{Document: model.Document{Name: "wf-fallback"}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: "wf-fallback.lonely",
		}).
		Once()

	b, err := NewCallHTTPTaskBuilder(w, newTestHTTPTask(), "lonely", doc, testEvents, nil)
	assert.NoError(t, err)

	_, err = b.Build()
	assert.NoError(t, err)

	w.AssertExpectations(t)
}

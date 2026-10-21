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
)

func TestTaskPointerFollowsTheDocument(t *testing.T) {
	target := func() *model.SetTask { return &model.SetTask{} }
	inFork, inTry, inCatch, inFor, inDo, escaped, top := target(), target(), target(), target(), target(), target(), target()
	sameNameElsewhere := target()

	list := func(items ...*model.TaskItem) *model.TaskList {
		l := model.TaskList(items)
		return &l
	}
	item := func(key string, task model.Task) *model.TaskItem { return &model.TaskItem{Key: key, Task: task} }

	doc := &model.Workflow{Do: list(
		item("fan", &model.ForkTask{Fork: model.ForkTaskConfiguration{Branches: list(
			item("left", target()), item("right", inFork),
		)}}),
		item("guarded", &model.TryTask{
			Try:   list(item("step", inTry)),
			Catch: &model.TryTaskCatch{Do: list(item("step", inCatch))},
		}),
		item("loop", &model.ForTask{Do: list(item("each", inFor))}),
		item("group", &model.DoTask{Do: list(item("inner", &model.DoTask{Do: list(item("deep", inDo))}))}),
		item("a/b~c", escaped),
		item("step", top),
		item("other", &model.DoTask{Do: list(item("step", sameNameElsewhere))}),
	)}

	for want, task := range map[string]model.Task{
		"/do/0/fan/fork/branches/1/right":  inFork,
		"/do/1/guarded/try/0/step":         inTry,
		"/do/1/guarded/catch/do/0/step":    inCatch,
		"/do/2/loop/do/0/each":             inFor,
		"/do/3/group/do/0/inner/do/0/deep": inDo,
		"/do/4/a~1b~0c":                    escaped,
		"/do/5/step":                       top,
		"/do/6/other/do/0/step":            sameNameElsewhere,
	} {
		got, ok := taskPointer(doc, task)
		assert.True(t, ok, want)
		assert.Equal(t, want, got)
	}

	_, ok := taskPointer(doc, target())
	assert.False(t, ok, "a task outside the document has no pointer")
}

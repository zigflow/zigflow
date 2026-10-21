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
	"strconv"
	"strings"

	"github.com/open-workflow-specification/sdk-go/v4/model"
)

// pointerTokenEscaper escapes a JSON Pointer reference token (RFC 6901).
var pointerTokenEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// instancePointer is the JSON Pointer to this task in its document, for
// example /do/0/checkout/try/1/charge: the error instance the specification
// asks a runtime to set. A task built outside its document falls back to a
// pointer made of its name.
func (d *builder[T]) instancePointer() string {
	if pointer, ok := taskPointer(d.doc, d.task); ok {
		return pointer
	}
	return "/" + pointerTokenEscaper.Replace(d.name)
}

// taskPointer finds target in doc by identity, so tasks that share a name in
// different scopes get different pointers.
func taskPointer(doc *model.Workflow, target model.Task) (string, bool) {
	if doc == nil {
		return "", false
	}
	return findTaskPointer(doc.Do, "/do", target)
}

func findTaskPointer(list *model.TaskList, prefix string, target model.Task) (string, bool) {
	if list == nil {
		return "", false
	}
	for i, item := range *list {
		pointer := prefix + "/" + strconv.Itoa(i) + "/" + pointerTokenEscaper.Replace(item.Key)
		if item.Task == target {
			return pointer, true
		}
		for _, child := range nestedTaskLists(item.Task) {
			if found, ok := findTaskPointer(child.list, pointer+child.path, target); ok {
				return found, true
			}
		}
	}
	return "", false
}

type nestedTaskList struct {
	path string
	list *model.TaskList
}

// nestedTaskLists returns the task lists a task holds, keyed by their path in
// the task's own definition.
func nestedTaskLists(task model.Task) []nestedTaskList {
	switch t := task.(type) {
	case *model.DoTask:
		return []nestedTaskList{{path: "/do", list: t.Do}}
	case *model.ForTask:
		return []nestedTaskList{{path: "/do", list: t.Do}}
	case *model.ForkTask:
		return []nestedTaskList{{path: "/fork/branches", list: t.Fork.Branches}}
	case *model.TryTask:
		lists := []nestedTaskList{{path: "/try", list: t.Try}}
		if t.Catch != nil {
			lists = append(lists, nestedTaskList{path: "/catch/do", list: t.Catch.Do})
		}
		return lists
	}
	return nil
}

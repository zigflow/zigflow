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

package zigflow

import (
	"fmt"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/zigflow/zigflow/pkg/utils"
)

// WarningCodeNestedDoDefinition marks a `do` task in a task list that also
// contains executable tasks. It has no documentation page, unlike the ERR_ codes.
const WarningCodeNestedDoDefinition = "WARN_NESTED_DO_DEFINITION"

// NestedDoWarnings reports every `do` task in a task list that also contains
// executable tasks, unless a `then:` directive names it. The do builder treats
// such a `do` as a separate workflow definition, and whether it also runs
// inline depends only on its position, which is rarely what the author meant.
// The whole list is considered, so the warning does not depend on ordering. A
// `switch` case target is skipped because redirecting to it starts the
// registered workflow, which is the intended pattern. A task-level `then:`
// target is still reported: that jump looks for a task in the executable list,
// which the `do` is not part of, so the workflow fails at runtime with
// "next target specified but not found". It
// is a warning rather than an error because the workflow it defines can still
// be called with `run: workflow`.
func NestedDoWarnings(doc *model.Workflow) []utils.ValidationErrors {
	if doc == nil {
		return nil
	}

	w := &nestedDoWalker{targets: map[string]bool{}}
	w.walkTaskList(doc.Do, "$.do")

	var warnings []utils.ValidationErrors
	for _, c := range w.candidates {
		if !w.targets[c.Key] {
			warnings = append(warnings, c)
		}
	}

	return warnings
}

type nestedDoWalker struct {
	candidates []utils.ValidationErrors
	targets    map[string]bool
}

func (w *nestedDoWalker) walkTaskList(list *model.TaskList, path string) {
	if list == nil {
		return
	}

	hasExecutable := false
	for _, item := range *list {
		if item.AsDoTask() == nil {
			hasExecutable = true
			break
		}
	}

	for i, item := range *list {
		itemPath := fmt.Sprintf("%s[%d]", path, i)

		if hasExecutable && item.AsDoTask() != nil {
			w.candidates = append(w.candidates, nestedDoWarning(item.Key, itemPath))
		}

		w.walkTaskItem(item, itemPath)
	}
}

// walkTaskItem records the task's `switch` case targets and descends into
// every task list the task runs through the do builder.
func (w *nestedDoWalker) walkTaskItem(item *model.TaskItem, path string) {
	base := fmt.Sprintf("%s.%s", path, item.Key)

	switch {
	case item.AsSwitchTask() != nil:
		for _, cases := range item.AsSwitchTask().Switch {
			for _, c := range cases {
				if c.Then != nil {
					w.targets[c.Then.Value] = true
				}
			}
		}
	case item.AsDoTask() != nil:
		w.walkTaskList(item.AsDoTask().Do, base+".do")
	case item.AsForTask() != nil:
		w.walkTaskList(item.AsForTask().Do, base+".do")
	case item.AsTryTask() != nil:
		try := item.AsTryTask()
		w.walkTaskList(try.Try, base+".try")
		if try.Catch != nil {
			w.walkTaskList(try.Catch.Do, base+".catch.do")
		}
	case item.AsForkTask() != nil:
		// The branches list is run by the fork builder, not the do builder,
		// so only each branch's own task list is checked.
		if branches := item.AsForkTask().Fork.Branches; branches != nil {
			for i, branch := range *branches {
				w.walkTaskItem(branch, fmt.Sprintf("%s.fork.branches[%d]", base, i))
			}
		}
	}
}

func nestedDoWarning(key, path string) utils.ValidationErrors {
	return utils.ValidationErrors{
		Key:  key,
		Code: WarningCodeNestedDoDefinition,
		Path: path,
		Message: fmt.Sprintf(
			"Task %q is a `do` task in a task list that also contains executable tasks. Zigflow treats it as a "+
				"separate workflow definition, not an inline group: it runs inline only if it comes before the "+
				"first executable task, and is skipped if it comes after one. Use normal tasks and flow directives "+
				"to keep the logic in the same workflow execution. If it is called with `run: workflow`, ignore "+
				"this warning.",
			key,
		),
	}
}

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
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
)

const (
	// testConstValue is a generic map key used in test payloads.
	testConstValue = "value"
	// testConstMessage is a map key used in test payloads that carry a message field.
	testConstMessage = "message"
	// testConstRequestID is a map key used for request identifiers in test inputs.
	testConstRequestID = "request_id"
	// testConstData is a map key used in event payloads that carry a data field.
	testConstData = "data"
	// testConstResult is a map key used in test payloads that carry a result field.
	testConstResult = "result"
	// testConstDataFlag is the jq expression used in tests to evaluate a boolean flag from $data.
	testConstDataFlag = "${ $data.flag }"
	// testConstTaskOne is the task name "task-one" used in do-task tests.
	testConstTaskOne = "task-one"
	// testConstTaskTwo is the task name "task-two" used in do-task flow-control tests.
	testConstTaskTwo = "task-two"
	// testConstTaskC is the task name "task-c" used in do-task flow-control tests.
	testConstTaskC = "task-c"
	// testConstTaskA is the task name "task-a" used in task-level then dispatch tests.
	testConstTaskA = "task-a"
	// testConstTaskTarget is the task name "task-target" used in named-redirect tests.
	testConstTaskTarget = "task-target"
	// testConstTaskWithThen is the task name "with-then" used in task-level then dispatch tests.
	testConstTaskWithThen = "with-then"
	// testConstTaskSetChildOutput is the task name "set-child-output" used in end-propagation tests
	// where a child workflow records an output before signalling `then: end`.
	testConstTaskSetChildOutput = "set-child-output"
	// testConstTaskPrior is the task name "prior" used in tests where one task
	// primes state before a later task is examined.
	testConstTaskPrior = "prior"
	// testConstHandledKey is the map key "handled" used in try/catch tests
	// to flag that the catch handler ran.
	testConstHandledKey = "handled"
	// testConstTaskSwitch is the task name "task-switch" used in flow directive tests.
	testConstTaskSwitch = "task-switch"
	// testConstFlowContinue is the literal "continue" flow directive value, repeated in tests.
	testConstFlowContinue = "continue"
	// testConstSwitchMatch is the switch case name "match" used in switch-task tests.
	testConstSwitchMatch = "match"
	// testConstFlag is the data key "flag" used to gate switch and for conditions in tests.
	testConstFlag = "flag"
	// testConstForDataItems is the jq expression used to reference items via .data.items.
	testConstForDataItems = "${ .data.items }"
	// testConstForRefDataItems is the jq expression used to reference items via $data.items.
	testConstForRefDataItems = "${ $data.items }"
	// testConstIdx is the loop index variable name used in for-task tests.
	testConstIdx = "idx"
	// testConstStep is the step task name used in for-task tests.
	testConstStep = "step"
	// testConstItemValue is the sample item string used in for-task iterator tests.
	testConstItemValue = "item-value"
	// testConstChildValue is the child output key used in for-task iterator tests.
	testConstChildValue = "child_value"
	// testConstLast is the key used to store the last-seen item in context propagation tests.
	testConstLast = "last"
	// testConstProcessed is the key used for the processed output in accumulator tests.
	testConstProcessed = "processed"
	// testConstEcho is the shell command used in run-task shell tests.
	testConstEcho = "echo"
	// testConstChild is the map key used in run-task child workflow result tests.
	testConstChild = "child"
	// testConstOK is the string value assigned to an env var in set-task tests.
	testConstOK = "ok"
	// testConstItems is the map key used for the items slice in for-task tests.
	testConstItems = "items"
	// testConstDone is the string value returned by child workflows in run-task tests.
	testConstDone = "done"
	// testConstRequest is the map key used for request payloads in run-task tests.
	testConstRequest = "request"
	// testConstAlpineImage is the container image string reused by run.container tests.
	testConstAlpineImage = "alpine"
	// testConstRunWorkflowVersion is the placeholder workflow version reused by run.workflow tests.
	testConstRunWorkflowVersion = "1.0.0"
	// testConstVal is the value variable name used in for-task accumulator tests.
	testConstVal = "val"
	// testConstCount is the map key used for the iteration count in numeric for-task tests.
	testConstCount = "count"
	// testConstForRefDataCount is the jq expression used to reference a numeric iteration count via $data.count.
	testConstForRefDataCount = "${ $data.count }"
)

var (
	// testWorkflow is a shared workflow instance for testing purposes.
	testWorkflow = &model.Workflow{
		Document: model.Document{
			Namespace: "some-namespace",
			Name:      "some-name",
		},
	}

	// testEvents is a shared events instance for testing purposes.
	testEvents, _ = cloudevents.Load("", nil, testWorkflow)
)

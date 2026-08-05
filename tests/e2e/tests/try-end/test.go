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

package tryend

import (
	_ "embed"

	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// testCase proves `then: end` inside a try body ends the whole workflow. The
// try and catch bodies execute inline in the workflow's own Temporal context,
// so the nested task list cannot rely on that context to tell it apart from
// the true root: without an explicit nested marker the end directive was
// consumed as a clean completion of the try body, the catch handler ran, and
// the task after the try task ran too.
//
// The expected output is the ending task's own output: nothing after it
// contributed, so `catchRan` and `afterTry` must both be absent.
var testCase = utils.TestCase{
	Name:         "try-end",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"result": "ended",
	},
	Test: utils.RunToCompletion[map[string]any],
}

func init() {
	utils.AddTestCase(&testCase)
}

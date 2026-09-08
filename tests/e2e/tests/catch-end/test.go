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

package catchend

import (
	_ "embed"

	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// testCase is the symmetric counterpart to the try-end case: an ordinary
// try-body failure still enters the catch body, and a `then: end` emitted by a
// real task inside that inline catch body ends the whole workflow rather than
// just the catch scope. The task after the try task must not run.
var testCase = utils.TestCase{
	Name:         "catch-end",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"result": map[string]any{
			"handled": true,
		},
	},
	Test: utils.RunToCompletion[map[string]any],
}

func init() {
	utils.AddTestCase(&testCase)
}

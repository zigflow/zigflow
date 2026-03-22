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

package full

import (
	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// expectedResult is the accumulated $context built up by each stage:
//   - init exports { stage: "init", count: 3 }
//   - fetch merges in { postId: "1" }
//   - iterate merges in { looped: true }
//   - route merges in { routed: true }
//
// The final try task wraps the context under the "result" key.
var testCase = utils.TestCase{
	Name:         "full",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"result": map[string]any{
			"stage":  "init",
			"count":  float64(3),
			"postId": "1",
			"looped": true,
			"routed": true,
		},
	},
	Test: utils.RunToCompletion[map[string]any],
}

func init() {
	utils.AddTestCase(testCase)
}

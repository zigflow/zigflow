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

package catchas

import (
	_ "embed"

	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// testCase proves the public catch contract at the workflow YAML level: when
// `catch.as` is configured the caught error is exposed under $data.<as> (here
// `caughtError`) carrying the enriched Temporal fields, and the default
// $data.error key is not populated.
var testCase = utils.TestCase{
	Name:         "catch-as",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"result": map[string]any{
			"caughtErrorPresent":     true,
			"caughtErrorHasType":     true,
			"defaultErrorKeyPresent": false,
		},
	},
	Test: utils.RunToCompletion[map[string]any],
}

func init() {
	utils.AddTestCase(&testCase)
}

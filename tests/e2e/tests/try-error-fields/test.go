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

package tryerrorfields

import (
	_ "embed"

	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// testCase proves a `raise` error caught by a try exposes its own type, status
// and title under $data.error. The try body runs inline, so the catch receives
// the original Zigflow error rather than one flattened by a child workflow
// boundary. status is a JSON number, so it decodes as float64.
var testCase = utils.TestCase{
	Name:         "try-error-fields",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"result": map[string]any{
			"type":   "https://serverlessworkflow.io/spec/1.0.0/errors/validation",
			"status": float64(400),
			"title":  "Boom",
			// A JSON Pointer to the raise task, not the workflow ID.
			"instance": "/do/0/guarded/try/0/failing",
		},
	},
	Test: utils.RunToCompletion[map[string]any],
}

func init() {
	utils.AddTestCase(&testCase)
}

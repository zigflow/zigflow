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

package multifilediffqueues

import (
	"testing"

	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// Both files define a workflow named "workflow" but under different namespaces
// (mf-ns1, mf-ns2), so they land on separate task queues and separate workers.
// This test verifies that a single Zigflow process starts multiple workers and
// each executes correctly.
var testCase = utils.TestCase{
	Name:         "multi-file-diff-queues",
	WorkflowPath: "workflow-ns1.yaml",
	ExtraFiles:   []string{"workflow-ns2.yaml"},
	Test: func(t *testing.T, test *utils.TestCase) {
		utils.RunToCompletionNamed[map[string]any](t,
			"mf-ns1", "workflow", nil,
			map[string]any{"data": map[string]any{"source": "ns1"}},
		)
		utils.RunToCompletionNamed[map[string]any](t,
			"mf-ns2", "workflow", nil,
			map[string]any{"data": map[string]any{"source": "ns2"}},
		)
	},
}

func init() {
	utils.AddTestCase(&testCase)
}

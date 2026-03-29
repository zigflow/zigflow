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

package multifilesamequeue

import (
	"testing"

	"github.com/zigflow/zigflow/tests/e2e/utils"
)

// Both workflow-a and workflow-b share the "mf-same" task queue. This test
// verifies that a single Zigflow process can host both on the same worker
// and execute each independently.
var testCase = utils.TestCase{
	Name:         "multi-file-same-queue",
	WorkflowPath: "workflow-a.yaml",
	ExtraFiles:   []string{"workflow-b.yaml"},
	Test: func(t *testing.T, test *utils.TestCase) {
		utils.RunToCompletionNamed[map[string]any](t,
			"mf-same", "workflow-a", nil,
			map[string]any{"data": map[string]any{"source": "workflow-a"}},
		)
		utils.RunToCompletionNamed[map[string]any](t,
			"mf-same", "workflow-b", nil,
			map[string]any{"data": map[string]any{"source": "workflow-b"}},
		)
	},
}

func init() {
	utils.AddTestCase(&testCase)
}

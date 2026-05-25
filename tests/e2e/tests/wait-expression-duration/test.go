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

// Package waitexpressionduration exercises the expression-aware duration
// form of the wait extension. The seconds field carries a runtime expression
// that resolves to a number drawn from the workflow input. The test verifies
// both the output and that the workflow actually waited for the durable
// timer to fire.
package waitexpressionduration

import (
	"context"
	"testing"
	"time"

	"github.com/mrsimonemms/golang-helpers/temporal"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/tests/e2e/utils"
	"go.temporal.io/sdk/client"
)

const cooldownSeconds = 2

// waitMinimum is the lower bound on observed elapsed time. It is set below
// the configured cooldown to allow for small differences between the test
// process clock and the Temporal server clock.
const waitMinimum = (cooldownSeconds * time.Second) - 500*time.Millisecond

var testCase = utils.TestCase{
	Name:         "wait-expression-duration",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"data": map[string]any{
			"status": "done",
		},
	},
	Test: func(t *testing.T, test *utils.TestCase) {
		c, err := temporal.NewConnectionWithEnvvars(
			temporal.WithZerolog(&zlog.Logger),
		)
		require.NoError(t, err)
		defer c.Close()

		input := map[string]any{"cooldownSeconds": cooldownSeconds}

		startTime := time.Now()
		wCtx := context.Background()

		we, err := c.ExecuteWorkflow(wCtx, client.StartWorkflowOptions{
			TaskQueue: test.Workflow.Document.Namespace,
		}, test.Workflow.Document.Name, input)
		require.NoError(t, err)

		var result map[string]any
		require.NoError(t, we.Get(wCtx, &result))

		assert.Equal(t, test.ExpectedOutput, result)
		assert.GreaterOrEqual(t, time.Since(startTime), waitMinimum,
			"workflow must have waited for the resolved duration")
	},
}

func init() {
	utils.AddTestCase(&testCase)
}

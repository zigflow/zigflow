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

// Package waituntil exercises the wait.until extension with a runtime
// expression that resolves to an RFC 3339 timestamp a couple of seconds in
// the future. The deadline is computed at test runtime and passed via the
// workflow input. The test verifies both the output and that the workflow
// actually waited for the durable timer to fire.
package waituntil

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

// waitDuration is how far in the future the deadline is set. Large enough
// for the workflow's durable timer to be observable in wall-clock time, but
// small enough to keep the e2e suite quick.
const waitDuration = 3 * time.Second

// waitMinimum is the lower bound on observed elapsed time. It is set below
// waitDuration to allow for small clock differences between the test process
// and the Temporal server.
const waitMinimum = waitDuration - 500*time.Millisecond

var testCase = utils.TestCase{
	Name:         "wait-until",
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

		deadline := time.Now().UTC().Add(waitDuration).Format(time.RFC3339Nano)
		input := map[string]any{"deadline": deadline}

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
			"workflow must have waited for the durable timer to fire")
	},
}

func init() {
	utils.AddTestCase(&testCase)
}

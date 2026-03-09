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

package complete

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/mrsimonemms/golang-helpers/temporal"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/tests/e2e/utils"
	"go.temporal.io/sdk/client"
)

var userId = int(3)

type rObj struct {
	State rState `json:"state"`
}

type rState struct {
	Envvar string      `json:"envvar"`
	UUID   string      `json:"uuid"`
	Input  rStateInput `json:"input"`
}

type rStateInput struct {
	UserID int `json:"userId"`
}

var testCase = utils.TestCase{
	Name:         "complete",
	WorkflowPath: "workflow.yaml",
	Input: map[string]any{
		"userId": userId,
	},
	ExpectedOutput: rObj{
		State: rState{
			Envvar: os.Getenv("ZIGGY_EXAMPLE_ENVVAR"),
			Input: rStateInput{
				UserID: userId,
			},
		},
	},
	Test: func(t *testing.T, test utils.TestCase) {
		c, err := temporal.NewConnectionWithEnvvars(
			temporal.WithZerolog(&zlog.Logger),
		)
		assert.NoError(t, err)
		defer c.Close()

		workflowOptions := client.StartWorkflowOptions{
			TaskQueue: test.Workflow.Document.Namespace,
		}

		startTime := time.Now()
		wCtx := context.Background()

		we, err := c.ExecuteWorkflow(wCtx, workflowOptions, test.Workflow.Document.Name, test.Input)
		assert.NoError(t, err)

		// Check the query has returned data correctly
		queryHasRun := false
		go func() {
			for i := range 2 {
				res, err := c.QueryWorkflow(wCtx, we.GetID(), "", "state")
				assert.NoError(t, err)

				type q struct {
					Progress int `json:"progress"`
				}
				var state q

				assert.NoError(t, res.Get(&state))

				if i == 0 {
					assert.Equal(t, q{Progress: 0}, state)
				} else {
					assert.Equal(t, q{Progress: 100}, state)
				}

				time.Sleep(time.Second * 2)
			}

			queryHasRun = true
		}()

		var result rObj
		assert.NoError(t, we.Get(wCtx, &result))

		// Check nothing different, except where we expect difference (mostly random vars)
		diff := cmp.Diff(test.ExpectedOutput, result, cmpopts.IgnoreFields(rObj{}, "State.UUID"))
		assert.Empty(t, diff, "Structs differ:\n%s", diff)

		// Check the random vars meet the standards
		_, err = uuid.Parse(result.State.UUID)
		assert.NoError(t, err, "The generated UUID doesn't match an actual UUID")

		// Check it's taken at least 5 seconds
		assert.GreaterOrEqual(t, time.Since(startTime), 5*time.Second, "Ensure the 5 second wait has happened")

		// Check the async tests have run
		assert.True(t, queryHasRun, "The query tests haven't run")
	},
}

func init() {
	utils.AddTestCase(testCase)
}

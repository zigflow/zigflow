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

package main

import (
	"context"
	"os"
	"time"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func exec() error {
	// The client is a heavyweight object that should be created once per process.
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create client",
		}
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "updates", map[string]any{
		"userId": 3,
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	time.Sleep(time.Second * 2)

	log.Info().Str("event", "event1").Msg("Triggering update")
	updateHandle1, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.GetID(),
		WaitForStage: client.WorkflowUpdateStageCompleted,
		UpdateName:   "temperature",
		Args: []any{
			39,
		},
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error updating",
		}
	}

	var res1 any
	if err := updateHandle1.Get(ctx, &res1); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Update failed",
		}
	}

	log.Info().Interface("response", res1).Msg("First update resolved")

	time.Sleep(time.Second * 2)

	log.Info().Str("event", "event2").Msg("Triggering update")
	updateHandle2, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.GetID(),
		WaitForStage: client.WorkflowUpdateStageCompleted,
		UpdateName:   "bpm",
		Args: []any{
			130,
		},
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error updating",
		}
	}

	var res2 any
	if err := updateHandle2.Get(ctx, &res2); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Update failed",
		}
	}

	log.Info().Interface("response", res2).Msg("Second update resolved")

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

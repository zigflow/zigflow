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

	"github.com/google/uuid"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

type State struct {
	ID       uuid.UUID `json:"id"`
	Progress int       `json:"progressPercentage"`
	Status   string    `json:"status"`
}

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
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "signal")
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	go func() {
		// Change how long we wait before triggering the signal - times out at 10 seconds
		time.Sleep(time.Second * 3)

		// This won't be approved - continue waiting
		log.Info().Msg("Sending signal that it is not approved")
		if err := c.SignalWorkflow(ctx, we.GetID(), "", "approve", false); err != nil {
			// Fatal error in goroutine
			log.Fatal().Err(err).Msg("Error signalling workflow")
		}

		time.Sleep(time.Second * 3)

		// This is approved
		log.Info().Msg("Approve")
		if err := c.SignalWorkflow(ctx, we.GetID(), "", "approve", true); err != nil {
			// Fatal error in goroutine
			log.Fatal().Err(err).Msg("Error signalling workflow")
		}
	}()

	var res any
	if err := we.Get(ctx, &res); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error getting response",
		}
	}

	log.Info().Any("result", res).Msg("Workflow approved in time")

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

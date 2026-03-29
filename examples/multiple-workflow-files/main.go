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
	"sync"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

type trigger struct {
	TaskQueue string
	Workflow  string
}

func call(c client.Client, t *trigger, wg *sync.WaitGroup) error {
	defer wg.Done()

	l := log.With().Str("workflow", t.Workflow).Str("taskQueue", t.TaskQueue).Logger()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: t.TaskQueue,
	}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, t.Workflow)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	l.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	var result any
	if err := we.Get(ctx, &result); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error getting response",
		}
	}

	l.Info().Interface("result", result).Msg("Workflow completed")

	return nil
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

	triggers := []trigger{
		{
			TaskQueue: "zigflow",
			Workflow:  "workflow1",
		},
		{
			TaskQueue: "zigflow",
			Workflow:  "workflow2",
		},
		{
			// Same workflow, different task queue
			TaskQueue: "zigflow1",
			Workflow:  "workflow1",
		},
	}

	errors := make([]error, 0)
	var wg sync.WaitGroup
	for _, t := range triggers {
		wg.Add(1)

		go func() {
			if err := call(c, &t, &wg); err != nil {
				errors = append(errors, err)
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		return gh.FatalError{
			WithParams: func(l *zerolog.Event) *zerolog.Event {
				return l.Any("errors", errors)
			},
			Msg: "Call errored",
		}
	}

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

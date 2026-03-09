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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

type State struct {
	ID       uuid.UUID `json:"id"`
	Approved bool      `json:"approved"`
	Status   string    `json:"status"`
}

func exec(isApproved bool, delay time.Duration) error {
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
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "authoriseChangeRequest", map[string]any{
		// Send the reference to the change - might be a DB record or a PR number
		"changeId": "change-id-ref",
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-quit:
				log.Info().Msg("Workflow completed")
				return
			default:
				res, err := c.QueryWorkflow(ctx, we.GetID(), "", "get_state")
				if err != nil {
					// Keep as fatal as in goroutine
					log.Fatal().Err(err).Msg("Error querying workflow")
				}

				var state State
				if err := res.Get(&state); err != nil {
					// Keep as fatal as in goroutine
					log.Fatal().Err(err).Msg("Error getting query result")
				}
				log.Info().Interface("Query result", state).Msg("Response from query")

				time.Sleep(time.Second * 2)
			}
		}
	}()

	go func() {
		// Change how long we wait before triggering the signal to cater for the timeout
		log.Info().Dur("timeout", delay).Bool("isApproved", isApproved).Msg("Waiting before we trigger review")
		time.Sleep(delay)

		log.Info().Msg("Sending change review response")
		workflowID := fmt.Sprintf("%s_fork_waitForApproval", we.GetID())
		if err := c.SignalWorkflow(ctx, workflowID, "", "review", map[string]any{
			// Any data received here is set to the workflow's state
			"approved": isApproved,
		}); err != nil {
			// Fatal error kept in gorouting
			log.Fatal().Err(err).Msg("Error signalling workflow")
		}
	}()

	log.Info().Msg("Waiting for respose")
	if err := we.Get(ctx, nil); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error getting response",
		}
	}

	quit <- true

	fmt.Println("===")
	fmt.Println("Workflow completed")
	fmt.Println("===")

	return nil
}

func main() {
	logLevel := "info"
	if l, ok := os.LookupEnv("LOG_LEVEL"); ok {
		logLevel = l
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		os.Exit(gh.HandleFatalError(gh.FatalError{
			Cause: err,
			Msg:   "Error parsing log level",
			WithParams: func(l *zerolog.Event) *zerolog.Event {
				return l.Str("level", logLevel)
			},
		}))
	}
	zerolog.SetGlobalLevel(level)

	delay := time.Second * 15
	input := map[string]inputData{
		"approve": {question: "Do you approve of the change? (Y/n)"},
		"delay":   {question: fmt.Sprintf("How long do you want to wait until replying? (%s)", delay)},
	}

	for key, data := range input {
		answer, err := getInput(data.question)
		if err != nil {
			os.Exit(gh.HandleFatalError(err))
		}
		data.answer = answer
		input[key] = data
	}

	approve := input["approve"].answer == "" || strings.EqualFold(input["approve"].answer, "y")

	if d := input["delay"].answer; d != "" {
		var err error
		delay, err = time.ParseDuration(d)
		if err != nil {
			os.Exit(gh.HandleFatalError(err))
		}
	}

	if err := exec(approve, delay); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

func getInput(question string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(question)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.Trim(text, "\n"), nil
}

type inputData struct {
	question string
	answer   string
}

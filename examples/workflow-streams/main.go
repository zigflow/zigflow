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
	"encoding/json"
	"fmt"
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog/log"
	temporal "github.com/zigflow/helpers"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/converter"
	"golang.org/x/sync/errgroup"
)

func exec() error {
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to create client"}
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{TaskQueue: "zigflow"}

	input := map[string]any{}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "workflow-streams", input)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Error executing workflow"}
	}

	g, cctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := subscribeToStreams(cctx, c, we.GetID()); err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Error streaming data",
			}
		}
		return nil
	})

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	var result any
	if err := we.Get(ctx, &result); err != nil {
		return gh.FatalError{Cause: err, Msg: "Error getting response"}
	}

	log.Info().Interface("result", result).Msg("Workflow completed")

	f, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println("===")
	fmt.Println(string(f))
	fmt.Println("===")

	if err := g.Wait(); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error waiting for goroutine",
		}
	}

	return nil
}

type Topic1 struct {
	ID     string  `json:"id"`
	Number float64 `json:"number"`
	Mesage string  `json:"message"`
	Happy  bool    `json:"happy"`
}

func subscribeToStreams(ctx context.Context, c client.Client, workflowID string) error {
	stream := workflowstreams.NewClient(c, workflowID, workflowstreams.Options{})
	dc := converter.GetDefaultDataConverter()

	for item, err := range stream.Subscribe(ctx, workflowstreams.SubscribeOptions{
		Topics: []string{
			"topic1",
			"topic2",
		},
	}) {
		if err != nil {
			return err
		}

		switch item.Topic {
		case "topic1":
			var evt Topic1
			if err := dc.FromPayload(item.Data, &evt); err != nil {
				return err
			}

			fmt.Printf("Topic1: %+v\n", evt)
		case "topic2":
			// Topic2 doesn't receive any data
			var evt any
			if err := dc.FromPayload(item.Data, &evt); err != nil {
				return err
			}

			fmt.Printf("Topic2: %+v\n", evt)
		}
	}

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

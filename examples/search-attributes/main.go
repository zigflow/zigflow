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
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
)

var searchAttributes map[string]enums.IndexedValueType = map[string]enums.IndexedValueType{
	"call":     enums.INDEXED_VALUE_TYPE_TEXT,
	"hello":    enums.INDEXED_VALUE_TYPE_TEXT,
	"waitTime": enums.INDEXED_VALUE_TYPE_INT,
	"userId":   enums.INDEXED_VALUE_TYPE_INT,
}

func upsertSearchAttributes(ctx context.Context, c client.Client, namespace string) error {
	log.Debug().Interface("searchAttributes", searchAttributes).Msg("Upserting search attributes")

	_, err := c.OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
		Namespace:        namespace,
		SearchAttributes: searchAttributes,
	})

	return err
}

func exec() error {
	// The client is a heavyweight object that should be created once per process.
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = client.DefaultNamespace
	}
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

	ctx := context.Background()
	if err := upsertSearchAttributes(ctx, c, namespace); err != nil {
		// If using Temporal Cloud, this will need to be done in the UI
		log.Warn().Err(err).Msg("Error upserting search attributes")
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}

	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "searchAttributes", map[string]any{
		"userId": 3,
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	var result any
	if err := we.Get(ctx, &result); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error getting response",
		}
	}

	log.Info().Interface("result", result).Msg("Workflow completed")

	f, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println("===")
	fmt.Println(string(f))
	fmt.Println("===")

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

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
	"strconv"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"github.com/zigflow/zigflow/pkg/externalstorage"
	"go.temporal.io/sdk/client"
)

func exec() error {
	// The client is a heavyweight object that should be created once per process.
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
		func(o *client.Options) error {
			n, err := strconv.Atoi(os.Getenv("EXTERNAL_STORAGE_PAYLOAD_SIZE_THRESHOLD"))
			if err != nil {
				return err
			}

			e, err := externalstorage.New(context.Background(), externalstorage.Config{
				Type:                 externalstorage.StorageS3,
				PayloadSizeThreshold: n,
				S3: externalstorage.S3Config{
					Bucket:       os.Getenv("EXTERNAL_STORAGE_S3_BUCKET"),
					Region:       os.Getenv("EXTERNAL_STORAGE_S3_REGION"),
					Endpoint:     os.Getenv("EXTERNAL_STORAGE_S3_ENDPOINT"),
					UsePathStyle: os.Getenv("EXTERNAL_STORAGE_S3_USE_PATH_STYLE") == "true",
				},
			})
			if err != nil {
				return err
			}
			o.ExternalStorage = e

			return nil
		},
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
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "hello-world")
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

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
	"crypto/rand"
	"math/big"
	"time"

	"app/cmd"
	basic "app/v1"

	golanghelpers "github.com/mrsimonemms/golang-helpers"
	grpcHelper "github.com/mrsimonemms/golang-helpers/grpc"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	name        = "basic"
	description = "Basic example gRPC service to simulate how the library works"
)

func main() {
	// Create the instance of the gRPC commands - these are defined by the proto files in /v1
	//
	// This receives some config (such as from environment variables) that enables the command
	// to access sensitive resources. In this example, it's receiving a database connection
	// string and the New command creates the connection.
	basicCmd := cmd.New("root:password@localhost:3306")

	// Create the helper
	//
	// The name and description are passed to the Cobra command to create the --help
	// command's name and description.
	// The ServerFactory wires up the gRPC server to the desired gRPC client. Any
	// additional configuration for the server goes in here. By default, it adds
	// the gRPC reflection and health checks.
	g := grpcHelper.New(
		name,
		description,
		[]grpcHelper.ServerFactory{
			func(server *grpc.Server) {
				basic.RegisterBasicServiceServer(server, basicCmd)
			},
		},
		// Add optional customisation to the server
		grpcHelper.Options{
			HealthChecks: map[string]grpcHelper.HealthCheck{
				// gRPC health checks receive a service name property, defaulting to ""
				"": {
					Check: func(s *health.Server) grpc_health_v1.HealthCheckResponse_ServingStatus {
						// Wire into your health check
						//
						// This is a random number generator which returns an error
						// if the generated number is 1
						n, _ := rand.Int(rand.Reader, big.NewInt(9))
						if n.Int64() == 1 {
							return grpc_health_v1.HealthCheckResponse_NOT_SERVING
						}

						return grpc_health_v1.HealthCheckResponse_SERVING
					},
					// Timeout defaults to 10 seconds
					Timeout: golanghelpers.Ptr(time.Second * 5),
				},
			},
		})

	// Define a gRPC command
	//
	// The name should be the same name as the gRPC command, although it doesn't have to be.
	// These commands are only used in "run" mode and ignored in "prod" mode.
	grpcHelper.NewGRPCCommand(g, "command1", grpcHelper.Listener[basic.Command1Response]{
		// Define the Cobra flags
		Flags: func(c *cobra.Command) {
			c.Flags().String("input", "default input", "Some input")
		},
		// Define the run command
		Run: func(c *cobra.Command, s []string) (*basic.Command1Response, error) {
			// Receive the inputs
			input, err := c.Flags().GetString("input")
			cobra.CheckErr(err)

			// Create an input request and invoke the gRPC command.
			return basicCmd.Command1(context.Background(), &basic.Command1Request{Input: input})
		},
	})

	// Add in a streaming command
	grpcHelper.NewGRPCCommand(g, "command2", grpcHelper.Listener[basic.Command2Response]{
		Flags: func(c *cobra.Command) {
			// Add in the flags as before
			c.Flags().String("input1", "default input1", "Some input1")
			c.Flags().String("input2", "default input2", "Some input2")
		},
		Run: func(c *cobra.Command, s []string) (*basic.Command2Response, error) {
			// Get the flags as before
			input1, err := c.Flags().GetString("input1")
			cobra.CheckErr(err)

			input2, err := c.Flags().GetString("input2")
			cobra.CheckErr(err)

			// As this emits multiple messages, the command receives the request
			// and a server. When mocking the command for development purposes,
			// you can use the StreamResponse helper which spoofs the gRPC stream
			// server and sends the output to the logger.
			return nil, basicCmd.Command2(&basic.Command2Request{
				Input1: input1,
				Input2: input2,
			}, &grpcHelper.StreamResponse[basic.Command2Response]{})
		},
	})

	// Let's get cracking
	g.Execute()
}

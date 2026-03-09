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

package cmd

import (
	"fmt"

	basic "app/v1"
)

type client struct {
	db string
}

func (c *client) Run(cmd string) (string, error) {
	return fmt.Sprintf("This has executed %s and the connection is %s", cmd, c.db), nil
}

type Commands struct {
	// Default to the unimplemented version of the server
	basic.UnimplementedBasicServiceServer

	// Spoofed SQL client
	client
}

// Constructor
//
// This receives a connection to demonstrate one way of creating
// private variables that can be used by the gRPC command. This example
// does nothing, but spoofs a database connection.
//
// In reality, you can create your commands however you like. This is just
// how I like to do it.
func New(conn string) *Commands {
	return &Commands{
		client: client{
			db: conn,
		},
	}
}

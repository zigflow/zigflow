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
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	zigflowexamples "github.com/zigflow/zigflow/examples"
	m "github.com/zigflow/zigflow/pkg/mcp"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the Zigflow MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := mcp.NewServer(&mcp.Implementation{
				Name:       "zigflow",
				Version:    Version,
				Title:      "Zigflow",
				WebsiteURL: "https://zigflow.dev",
			}, nil)

			_ = m.New(server, Version, zigflowexamples.EmbeddedFS)

			return server.Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}

	return cmd
}

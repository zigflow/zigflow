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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	m "github.com/zigflow/zigflow/pkg/mcp"
)

func newMCPCmd() *cobra.Command {
	var opts struct {
		Address    string
		Transport  string
		WebsiteURL string
	}

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the Zigflow MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := mcp.NewServer(&mcp.Implementation{
				Name:       "zigflow",
				Version:    Version,
				Title:      "Zigflow",
				WebsiteURL: opts.WebsiteURL,
			}, nil)

			_ = m.New(server, Version)

			switch opts.Transport {
			case "http":
				return m.HTTPHandler(cmd.Context(), server, opts.Address)
			case "stdio":
				return server.Run(cmd.Context(), &mcp.StdioTransport{})
			default:
				return fmt.Errorf("invalid transport type: %q", opts.Transport)
			}
		},
	}

	viper.SetDefault("address", "0.0.0.0:8080")
	cmd.Flags().StringVar(
		&opts.Address, "address",
		viper.GetString("address"), "Address to listen on (HTTP transport only)",
	)

	viper.SetDefault("transport", "stdio")
	cmd.Flags().StringVar(
		&opts.Transport, "transport",
		viper.GetString("transport"), "Transport to use: stdio or http",
	)

	viper.SetDefault("website_url", "https://mcp.zigflow.dev")
	cmd.Flags().StringVar(
		&opts.WebsiteURL, "website-url",
		viper.GetString("website_url"), "WebsiteURL for the server",
	)

	return cmd
}

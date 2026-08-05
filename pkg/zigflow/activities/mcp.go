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

package activities

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

func init() {
	Registry = append(Registry, &CallMCP{})
}

const (
	mcpListTools             = "tools/list"
	mcpCallTool              = "tools/call"
	mcpListPrompts           = "prompts/list"
	mcpGetPrompt             = "prompts/get"
	mcpListResources         = "resources/list"
	mcpReadResource          = "resources/read"
	mcpListResourceTemplates = "resources/templates/list"
)

type CallMCP struct{}

func (c *CallMCP) CallMCPActivity(
	ctx context.Context, task *models.CallMCP, input any, state *utils.State,
) (any, error) {
	logger := activity.GetLogger(ctx)

	stopHeartbeat := metadata.StartActivityHeartbeat(ctx, task.GetBase())
	defer stopHeartbeat()

	impl := &mcp.Implementation{
		Name:    "Zigflow",
		Version: "v1.0.0",
	}

	if cl := task.With.Client; cl != nil {
		impl.Name = cl.Name
		impl.Version = cl.Version
	}

	client := mcp.NewClient(impl, nil)

	var transport mcp.Transport
	if t := task.With.Transport.HTTP; t != nil {
		endpoint := t.Endpoint.String()
		logger.Info("Calling MCP over HTTP", "endpoint", endpoint)
		transport = &mcp.StreamableClientTransport{
			Endpoint: endpoint,
		}
	} else if t := task.With.Transport.STDIO; t != nil {
		logger.Info("Calling MCP over STDIO")
		transport = &mcp.CommandTransport{
			//nolint:gosec // path originates from trusted config, not user input
			Command: exec.CommandContext(ctx, t.Command, t.Arguments...),
		}
	}

	logger.Debug("Connecting to MCP server")
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		logger.Debug("Disconnecting from MCP server")
		if err := session.Close(); err != nil {
			logger.Warn("Error disconnecting from MCP server", "error", err)
		}
	}()

	logger.Debug("Calling MCP method", "method", task.With.Method)
	return c.callMethod(ctx, session, task)
}

func (c *CallMCP) callMethod(
	ctx context.Context,
	session *mcp.ClientSession,
	task *models.CallMCP,
) (any, error) {
	logger := activity.GetLogger(ctx)
	method := task.With.Method

	var result any
	var err error
	switch method {
	case mcpListTools:
		result, err = session.ListTools(ctx, &mcp.ListToolsParams{})
	case mcpCallTool:
		result, err = session.CallTool(ctx, &mcp.CallToolParams{})
	case mcpListPrompts:
		result, err = session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	case mcpGetPrompt:
		result, err = session.GetPrompt(ctx, &mcp.GetPromptParams{})
	case mcpListResources:
		result, err = session.ListResources(ctx, &mcp.ListResourcesParams{})
	case mcpReadResource:
		result, err = session.ReadResource(ctx, &mcp.ReadResourceParams{})
	case mcpListResourceTemplates:
		result, err = session.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{})
	default:
		logger.Error("Invalid MCP method", "method", method)
		return nil, temporal.NewNonRetryableApplicationError(
			"CallMCP given an invalid method",
			"CallMCP non-retryable error",
			fmt.Errorf("invalid mcp method: %s", method),
		)
	}

	if err != nil {
		logger.Error("Error calling MCP method", "method", method, "error", err)
		return nil, fmt.Errorf("error calling mcp method (%s): %w", method, err)
	}

	logger.Debug("Returning response from MCP server", "method", method)
	return result, nil
}

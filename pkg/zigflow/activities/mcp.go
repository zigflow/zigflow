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
	"net/http"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	swUtil "github.com/open-workflow-specification/sdk-go/v4/impl/utils"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/version"
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

	logger.Debug("Setting MCP call timeout", "timeout", task.With.Timeout)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, utils.ToDuration(task.With.Timeout))
	defer cancel()

	impl := &mcp.Implementation{
		Name:    "Zigflow",
		Version: version.Version,
	}

	// Clone and traverse, interpolating the data
	cloneData := swUtil.DeepCloneValue(task.With.Parameters)
	data, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(cloneData), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing http data object: %w", err)
	}
	task.With.Parameters = data

	if cl := task.With.Client; cl != nil {
		if cl.Name != "" {
			impl.Name = cl.Name
		}
		if cl.Version != "" {
			impl.Version = cl.Version
		}
	}

	client := mcp.NewClient(impl, nil)

	transport, err := c.createTransport(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("error creating transport: %w", err)
	}

	protocolVersion := "2025-06-18"
	if pv := task.With.ProtocolVersion; pv != "" {
		protocolVersion = pv
	}

	logger.Debug("Connecting to MCP server")
	session, err := client.Connect(ctx, transport, &mcp.ClientSessionOptions{
		ProtocolVersion: protocolVersion,
	})
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
		result, err = invokeMCP(ctx, &task.With, session.ListTools)
	case mcpCallTool:
		result, err = invokeMCP(ctx, &task.With, session.CallTool)
	case mcpListPrompts:
		result, err = invokeMCP(ctx, &task.With, session.ListPrompts)
	case mcpGetPrompt:
		result, err = invokeMCP(ctx, &task.With, session.GetPrompt)
	case mcpListResources:
		result, err = invokeMCP(ctx, &task.With, session.ListResources)
	case mcpReadResource:
		result, err = invokeMCP(ctx, &task.With, session.ReadResource)
	case mcpListResourceTemplates:
		result, err = invokeMCP(ctx, &task.With, session.ListResourceTemplates)
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

func (c *CallMCP) createTransport(ctx context.Context, task *models.CallMCP) (mcp.Transport, error) {
	logger := activity.GetLogger(ctx)

	var transport mcp.Transport
	if t := task.With.Transport.HTTP; t != nil {
		endpoint := t.Endpoint.String()
		logger.Info("Calling MCP over HTTP", "endpoint", endpoint)
		transport = &mcp.StreamableClientTransport{
			Endpoint: endpoint,
			HTTPClient: &http.Client{
				Transport: &headerRoundTripper{
					headers: t.Headers,
					next:    http.DefaultTransport,
				},
			},
		}
	} else if t := task.With.Transport.STDIO; t != nil {
		logger.Info("Calling MCP over STDIO", "command", t.Command, "arguments", t.Arguments)

		//nolint:gosec // path originates from trusted config, not user input
		cmd := exec.CommandContext(ctx, t.Command, t.Arguments...)
		if len(t.Environment) > 0 {
			cmd.Env = os.Environ() // Need ambient envvars for stdio transport to work
			for key, value := range t.Environment {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
			}
		}

		transport = &mcp.CommandTransport{
			Command: cmd,
		}
	}

	if transport == nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"CallMCP given an invalid transport",
			"CallMCP non-retryable error",
			fmt.Errorf("unknown transport type for MCP call: %v", task.With.Transport),
		)
	}

	return transport, nil
}

// invokeMCP converts the task arguments into the method's parameter type and
// calls the given MCP session method.
func invokeMCP[P, R any](
	ctx context.Context,
	args *models.MCPArguments,
	fn func(context.Context, *P) (R, error),
) (any, error) {
	p, err := args.ToParams[P]()
	if err != nil {
		var zero P
		return nil, fmt.Errorf("error converting mcp arguments to %T: %w", zero, err)
	}
	return fn(ctx, p)
}

type headerRoundTripper struct {
	headers map[string]string
	next    http.RoundTripper
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())

	for key, value := range h.headers {
		clone.Header.Set(key, value)
	}

	return h.next.RoundTrip(clone)
}

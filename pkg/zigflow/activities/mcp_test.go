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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

const (
	testMCPToolName      = "echo"
	testMCPCustomHeader  = "X-Zigflow-Test"
	testMCPAuthHeader    = "Bearer secret-token"
	testMCPPath          = "/mcp"
	testMCPCall          = "mcp"
	testMCPParamName     = "name"
	testMCPAuthorization = "Authorization"
	testMCPKeyMessage    = "message"
	testMCPKeyArguments  = "arguments"
)

type testMCPEchoInput struct {
	Message string `json:"message"`
}

type testMCPEchoOutput struct {
	Message string `json:"message"`
}

type testMCPInitialiseParams struct {
	ProtocolVersion string `json:"protocolVersion"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// testMCPServer is a real MCP server used as a fixture so tests can observe
// what Zigflow sends over the wire. It records request headers and the
// parameters of the initialise request.
type testMCPServer struct {
	*httptest.Server

	mu         sync.Mutex
	headers    []http.Header
	initialise []testMCPInitialiseParams
}

func newTestMCPServer(t *testing.T) *testMCPServer {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: testMCPToolName, Description: "Echo the message"},
		func(_ context.Context, _ *mcp.CallToolRequest, in testMCPEchoInput) (*mcp.CallToolResult, testMCPEchoOutput, error) {
			return nil, testMCPEchoOutput(in), nil
		})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)

	s := &testMCPServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		var msg struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal(body, &msg)

		s.mu.Lock()
		s.headers = append(s.headers, r.Header.Clone())
		if msg.Method == "initialize" { //nolint:misspell // MCP protocol method name
			var p testMCPInitialiseParams
			_ = json.Unmarshal(msg.Params, &p)
			s.initialise = append(s.initialise, p)
		}
		s.mu.Unlock()

		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)

	return s
}

func (s *testMCPServer) recordedHeaders() []http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.headers
}

func (s *testMCPServer) recordedInitialise() []testMCPInitialiseParams {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialise
}

func newTestMCPDuration(d model.DurationInline) *model.Duration {
	return &model.Duration{Value: d}
}

func newTestMCPHTTPTask(endpoint, method string, params any) *models.CallMCP {
	return &models.CallMCP{
		Call: testMCPCall,
		With: models.MCPArguments{
			Method:     method,
			Parameters: params,
			Timeout:    newTestMCPDuration(model.DurationInline{Seconds: 10}),
			Transport: &models.MCPTransport{
				HTTP: &models.MCPTransportHTTP{
					Endpoint: model.NewEndpoint(endpoint),
				},
			},
		},
	}
}

func executeCallMCPActivity(t *testing.T, task *models.CallMCP) (map[string]any, error) {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	c := &CallMCP{}
	env.RegisterActivity(c.CallMCPActivity)

	val, err := env.ExecuteActivity(c.CallMCPActivity, task, nil, utils.NewState())
	if err != nil {
		return nil, err
	}

	var out map[string]any
	require.NoError(t, val.Get(&out))
	return out, nil
}

// withMCPTransport runs createTransport inside an activity context, which it
// requires, and hands the result to fn while that context is still live.
// fn must only use assert (not require) as it may run off the test goroutine.
func withMCPTransport(t *testing.T, task *models.CallMCP, fn func(mcp.Transport, error)) {
	t.Helper()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	c := &CallMCP{}

	const name = "createMCPTransport"
	env.RegisterActivityWithOptions(func(ctx context.Context) error {
		fn(c.createTransport(ctx, task))
		return nil
	}, activity.RegisterOptions{Name: name})

	_, err := env.ExecuteActivity(name)
	require.NoError(t, err)
}

func assertNonRetryableApplicationError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected a Temporal ApplicationError, got %T", err)
	assert.True(t, appErr.NonRetryable(), "expected a non-retryable error, got: %v", err)
	assert.Equal(t, "CallMCP non-retryable error", appErr.Type())
}

func TestCallMCPActivityInvalidTransportIsNonRetryable(t *testing.T) {
	task := &models.CallMCP{
		Call: testMCPCall,
		With: models.MCPArguments{
			Method:    mcpListTools,
			Timeout:   newTestMCPDuration(model.DurationInline{Seconds: 10}),
			Transport: &models.MCPTransport{},
		},
	}

	_, err := executeCallMCPActivity(t, task)

	// errors.As finds the outermost ApplicationError, which is the one
	// Temporal uses to decide whether to retry.
	assertNonRetryableApplicationError(t, err)
	assert.Contains(t, err.Error(), "invalid transport")
}

func TestCallMCPCreateTransportInvalidTransportIsNonRetryable(t *testing.T) {
	task := &models.CallMCP{With: models.MCPArguments{Transport: &models.MCPTransport{}}}

	var gotErr error
	withMCPTransport(t, task, func(tr mcp.Transport, err error) {
		assert.Nil(t, tr)
		gotErr = err
	})

	assertNonRetryableApplicationError(t, gotErr)
}

func TestCallMCPActivityUnsupportedMethodIsNonRetryable(t *testing.T) {
	srv := newTestMCPServer(t)

	_, err := executeCallMCPActivity(t, newTestMCPHTTPTask(srv.URL+testMCPPath, "tools/unknown", nil))

	assertNonRetryableApplicationError(t, err)
	assert.Contains(t, err.Error(), "invalid mcp method: tools/unknown")
}

func TestCallMCPActivityHTTP(t *testing.T) {
	t.Run("lists tools", func(t *testing.T) {
		srv := newTestMCPServer(t)

		out, err := executeCallMCPActivity(t, newTestMCPHTTPTask(srv.URL+testMCPPath, mcpListTools, nil))
		require.NoError(t, err)

		tools, ok := out["tools"].([]any)
		require.True(t, ok, "expected tools in result: %v", out)
		require.Len(t, tools, 1)
		assert.Equal(t, testMCPToolName, tools[0].(map[string]any)[testMCPParamName])
	})

	t.Run("calls a tool with converted parameters", func(t *testing.T) {
		srv := newTestMCPServer(t)

		out, err := executeCallMCPActivity(t, newTestMCPHTTPTask(srv.URL+testMCPPath, mcpCallTool, map[string]any{
			testMCPParamName:    testMCPToolName,
			testMCPKeyArguments: map[string]any{testMCPKeyMessage: testHello},
		}))
		require.NoError(t, err)

		assert.Equal(t, map[string]any{testMCPKeyMessage: testHello}, out["structuredContent"])
	})

	t.Run("evaluates runtime expressions in parameters", func(t *testing.T) {
		srv := newTestMCPServer(t)

		out, err := executeCallMCPActivity(t, newTestMCPHTTPTask(srv.URL+testMCPPath, mcpCallTool, map[string]any{
			testMCPParamName:    testMCPToolName,
			testMCPKeyArguments: map[string]any{testMCPKeyMessage: `${ "hello" + " world" }`},
		}))
		require.NoError(t, err)

		assert.Equal(t, map[string]any{testMCPKeyMessage: "hello world"}, out["structuredContent"])
	})

	t.Run("unknown parameter is rejected", func(t *testing.T) {
		srv := newTestMCPServer(t)

		_, err := executeCallMCPActivity(t, newTestMCPHTTPTask(srv.URL+testMCPPath, mcpCallTool, map[string]any{
			testMCPParamName: testMCPToolName,
			"unknown":        true,
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error converting mcp arguments")
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("sends configured headers to the MCP server", func(t *testing.T) {
		srv := newTestMCPServer(t)
		task := newTestMCPHTTPTask(srv.URL+testMCPPath, mcpListTools, nil)
		task.With.Transport.HTTP.Headers = map[string]string{
			testMCPAuthorization: testMCPAuthHeader,
			testMCPCustomHeader:  "value",
		}

		_, err := executeCallMCPActivity(t, task)
		require.NoError(t, err)

		headers := srv.recordedHeaders()
		require.NotEmpty(t, headers)
		for _, h := range headers {
			assert.Equal(t, testMCPAuthHeader, h.Get(testMCPAuthorization))
			assert.Equal(t, "value", h.Get(testMCPCustomHeader))
		}
	})
}

func TestCallMCPActivityProtocolVersion(t *testing.T) {
	tests := []struct {
		name            string
		protocolVersion string
		want            string
	}{
		{
			name: "defaults when omitted",
			want: "2025-06-18",
		},
		{
			name:            "uses configured version",
			protocolVersion: "2025-03-26",
			want:            "2025-03-26",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestMCPServer(t)
			task := newTestMCPHTTPTask(srv.URL+testMCPPath, mcpListTools, nil)
			task.With.ProtocolVersion = tt.protocolVersion

			_, err := executeCallMCPActivity(t, task)
			require.NoError(t, err)

			init := srv.recordedInitialise()
			require.Len(t, init, 1)
			assert.Equal(t, tt.want, init[0].ProtocolVersion)
		})
	}
}

func TestCallMCPActivityClientInfo(t *testing.T) {
	t.Run("defaults to Zigflow", func(t *testing.T) {
		srv := newTestMCPServer(t)

		_, err := executeCallMCPActivity(t, newTestMCPHTTPTask(srv.URL+testMCPPath, mcpListTools, nil))
		require.NoError(t, err)

		init := srv.recordedInitialise()
		require.Len(t, init, 1)
		assert.Equal(t, "Zigflow", init[0].ClientInfo.Name)
	})

	t.Run("uses configured client", func(t *testing.T) {
		srv := newTestMCPServer(t)
		task := newTestMCPHTTPTask(srv.URL+testMCPPath, mcpListTools, nil)
		task.With.Client = &models.MCPClient{Name: "custom-client", Version: "1.2.3"}

		_, err := executeCallMCPActivity(t, task)
		require.NoError(t, err)

		init := srv.recordedInitialise()
		require.Len(t, init, 1)
		assert.Equal(t, "custom-client", init[0].ClientInfo.Name)
		assert.Equal(t, "1.2.3", init[0].ClientInfo.Version)
	})
}

func TestCallMCPActivityTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	task := newTestMCPHTTPTask(srv.URL+testMCPPath, mcpListTools, nil)
	task.With.Timeout = newTestMCPDuration(model.DurationInline{Milliseconds: 200})

	start := time.Now()
	_, err := executeCallMCPActivity(t, task)

	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "configured timeout must bound the call")
}

func TestCallMCPCreateTransportHTTP(t *testing.T) {
	task := newTestMCPHTTPTask("https://example.com/mcp", mcpListTools, nil)

	withMCPTransport(t, task, func(tr mcp.Transport, err error) {
		assert.NoError(t, err)
		streamable, ok := tr.(*mcp.StreamableClientTransport)
		if assert.True(t, ok, "expected streamable HTTP transport, got %T", tr) {
			assert.Equal(t, "https://example.com/mcp", streamable.Endpoint)
			assert.NotNil(t, streamable.HTTPClient)
		}
	})
}

// headerRecorder is an HTTP handler that records the headers of every request
// it receives, keyed by path.
type headerRecorder struct {
	mu      sync.Mutex
	headers map[string]http.Header
	next    http.HandlerFunc
}

func newHeaderRecorder(t *testing.T, next http.HandlerFunc) (*httptest.Server, *headerRecorder) {
	t.Helper()

	rec := &headerRecorder{headers: map[string]http.Header{}, next: next}
	srv := httptest.NewServer(rec)
	t.Cleanup(srv.Close)
	return srv, rec
}

func (h *headerRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.headers[r.URL.Path] = r.Header.Clone()
	h.mu.Unlock()

	if h.next != nil {
		h.next(w, r)
	}
}

func (h *headerRecorder) get(path string) (http.Header, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	v, ok := h.headers[path]
	return v, ok
}

// postWithMCPHTTPClient builds the HTTP transport for endpoint and sends a
// POST through its client, mirroring how the MCP client talks to the server.
func postWithMCPHTTPClient(t *testing.T, endpoint string, headers map[string]string) {
	t.Helper()

	task := newTestMCPHTTPTask(endpoint, mcpListTools, nil)
	task.With.Transport.HTTP.Headers = headers

	withMCPTransport(t, task, func(tr mcp.Transport, err error) {
		if !assert.NoError(t, err) {
			return
		}
		streamable, ok := tr.(*mcp.StreamableClientTransport)
		if !assert.True(t, ok, "expected streamable HTTP transport, got %T", tr) {
			return
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, strings.NewReader("{}"))
		if !assert.NoError(t, err) {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := streamable.HTTPClient.Do(req)
		if assert.NoError(t, err) {
			assert.NoError(t, resp.Body.Close())
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})
}

func TestCallMCPHTTPTransportHeaders(t *testing.T) {
	headers := map[string]string{
		testMCPAuthorization: testMCPAuthHeader,
		testMCPCustomHeader:  "value",
	}

	t.Run("sent to the endpoint", func(t *testing.T) {
		srv, rec := newHeaderRecorder(t, nil)

		postWithMCPHTTPClient(t, srv.URL+testMCPPath, headers)

		got, ok := rec.get(testMCPPath)
		require.True(t, ok)
		assert.Equal(t, testMCPAuthHeader, got.Get(testMCPAuthorization))
		assert.Equal(t, "value", got.Get(testMCPCustomHeader))
	})

	t.Run("retained on same-origin redirect", func(t *testing.T) {
		srv, rec := newHeaderRecorder(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == testMCPPath {
				http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
			}
		})

		postWithMCPHTTPClient(t, srv.URL+testMCPPath, headers)

		got, ok := rec.get("/redirected")
		require.True(t, ok, "redirect target was not called")
		assert.Equal(t, testMCPAuthHeader, got.Get(testMCPAuthorization))
		assert.Equal(t, "value", got.Get(testMCPCustomHeader))
	})

	t.Run("not forwarded on cross-origin redirect", func(t *testing.T) {
		target, targetRec := newHeaderRecorder(t, nil)
		origin, originRec := newHeaderRecorder(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+"/redirected", http.StatusTemporaryRedirect)
		})

		postWithMCPHTTPClient(t, origin.URL+testMCPPath, headers)

		got, ok := originRec.get(testMCPPath)
		require.True(t, ok)
		assert.Equal(t, testMCPAuthHeader, got.Get(testMCPAuthorization))

		got, ok = targetRec.get("/redirected")
		require.True(t, ok, "redirect target was not called")
		assert.Empty(t, got.Get(testMCPAuthorization), "Authorization must not leak to another origin")
		assert.Empty(t, got.Get(testMCPCustomHeader), "configured headers must not leak to another origin")
	})
}

func requireShell(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(testShellCmd); err != nil {
		t.Skip("sh not available")
	}
}

func newTestMCPSTDIOTask(script string, args []string, env map[string]string) *models.CallMCP {
	return &models.CallMCP{
		Call: testMCPCall,
		With: models.MCPArguments{
			Method:  mcpListTools,
			Timeout: newTestMCPDuration(model.DurationInline{Seconds: 10}),
			Transport: &models.MCPTransport{
				STDIO: &models.MCPTransportSTDIO{
					Command:     testShellCmd,
					Arguments:   append([]string{"-c", script, testShellCmd}, args...),
					Environment: env,
				},
			},
		},
	}
}

// runSTDIOTransportCommand builds the STDIO transport and runs its command
// directly, returning stdout. This exercises the process Zigflow configures
// without involving the MCP protocol.
func runSTDIOTransportCommand(t *testing.T, task *models.CallMCP) string {
	t.Helper()

	var out string
	withMCPTransport(t, task, func(tr mcp.Transport, err error) {
		if !assert.NoError(t, err) {
			return
		}
		cmdTransport, ok := tr.(*mcp.CommandTransport)
		if !assert.True(t, ok, "expected command transport, got %T", tr) {
			return
		}

		b, err := cmdTransport.Command.Output()
		assert.NoError(t, err)
		out = string(b)
	})
	return out
}

func TestCallMCPCreateTransportSTDIO(t *testing.T) {
	requireShell(t)

	t.Run("builds a command transport", func(t *testing.T) {
		task := newTestMCPSTDIOTask("true", []string{"a"}, nil)

		withMCPTransport(t, task, func(tr mcp.Transport, err error) {
			assert.NoError(t, err)
			cmdTransport, ok := tr.(*mcp.CommandTransport)
			if assert.True(t, ok, "expected command transport, got %T", tr) {
				assert.Equal(t, []string{testShellCmd, "-c", "true", testShellCmd, "a"}, cmdTransport.Command.Args)
			}
		})
	})

	t.Run("passes arguments verbatim", func(t *testing.T) {
		task := newTestMCPSTDIOTask(`printf '%s,' "$@"`, []string{"a", "b c", "$HOME"}, nil)

		assert.Equal(t, "a,b c,$HOME,", runSTDIOTransportCommand(t, task))
	})

	t.Run("inherits ambient environment when none configured", func(t *testing.T) {
		t.Setenv("ZIGFLOW_MCP_AMBIENT", "ambient")
		t.Setenv("ZIGFLOW_MCP_OVERRIDE", "original")

		task := newTestMCPSTDIOTask(
			`printf '%s|%s|%s' "$ZIGFLOW_MCP_AMBIENT" "$ZIGFLOW_MCP_OVERRIDE" "$ZIGFLOW_MCP_ADDED"`, nil, nil,
		)

		assert.Equal(t, "ambient|original|", runSTDIOTransportCommand(t, task))
	})

	t.Run("configured environment adds to and overrides ambient environment", func(t *testing.T) {
		t.Setenv("ZIGFLOW_MCP_AMBIENT", "ambient")
		t.Setenv("ZIGFLOW_MCP_OVERRIDE", "original")

		task := newTestMCPSTDIOTask(
			`printf '%s|%s|%s' "$ZIGFLOW_MCP_AMBIENT" "$ZIGFLOW_MCP_OVERRIDE" "$ZIGFLOW_MCP_ADDED"`, nil,
			map[string]string{
				"ZIGFLOW_MCP_OVERRIDE": "configured",
				"ZIGFLOW_MCP_ADDED":    "added",
			},
		)

		assert.Equal(t, "ambient|configured|added", runSTDIOTransportCommand(t, task))
	})
}

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

package mcp

import (
	"context"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

const stageInput = "input"

// maxBytes caps both the request body (via http.MaxBytesHandler) and the
// request headers (via http.Server.MaxHeaderBytes). MCP tool calls for Zigflow
// are small, so 1 MiB is generous while still bounding memory per request.
const maxBytes = 1 << 20 // 1 MiB

// CORS configuration for the public MCP HTTP endpoint. The server is public,
// read-only and stateless, so any origin may call it. Credentials are
// deliberately not enabled. The SDK does not configure CORS itself, so these
// headers are applied by withCORS.
const (
	corsAllowMethods  = "GET, POST, OPTIONS"
	corsAllowHeaders  = "Content-Type, Authorization, MCP-Protocol-Version, MCP-Session-Id, Last-Event-ID"
	corsExposeHeaders = "MCP-Session-Id"
)

type MCP struct {
	version string
}

// newHTTPHandler builds the public-facing MCP HTTP handler. The official
// Streamable HTTP handler is run in stateless mode so clients that do not
// preserve Mcp-Session-Id remain interoperable, and is wrapped with a body
// limit. It is kept separate from HTTPHandler so it can be exercised with
// httptest.
func newHTTPHandler(server *mcp.Server) http.Handler {
	var handler http.Handler = mcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcp.Server {
			return server
		},
		&mcp.StreamableHTTPOptions{
			Stateless: true,
		},
	)

	return withCORS(http.MaxBytesHandler(handler, maxBytes))
}

// withCORS adds permissive CORS headers so browser-based MCP clients can call
// the endpoint, and answers preflight OPTIONS requests with 204 No Content
// without invoking the wrapped MCP handler. The headers are also set on normal
// MCP responses.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", corsAllowMethods)
		h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
		h.Set("Access-Control-Expose-Headers", corsExposeHeaders)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func HTTPHandler(ctx context.Context, server *mcp.Server, address string) error {
	log.Info().Str("address", address).Msg("MCP server listening")

	return listenAndServeGracefully(ctx, &http.Server{
		Addr:              address,
		Handler:           newHTTPHandler(server),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    maxBytes,
	})
}

func listenAndServeGracefully(cctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil

	case <-cctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Info().Msg("MCP server shutting down")

		if err := srv.Shutdown(ctx); err != nil {
			return err
		}

		err := <-errCh
		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	}
}

func New(server *mcp.Server, version string) *MCP {
	m := &MCP{
		version: version,
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:  "get_schema",
		Title: "Get Schema",
		Description: "Returns the Zigflow workflow JSON schema for the current version. Use this to understand valid " +
			"workflow structure before generating or validating YAML.",
	}, m.GetSchema)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_example",
		Title:       "Get Example",
		Description: "Returns a Zigflow example by name, including its YAML content and metadata.",
	}, m.GetExample)

	mcp.AddTool(server, &mcp.Tool{
		Name:  "list_examples",
		Title: "List Examples",
		Description: "Lists the bundled Zigflow workflow examples with short descriptions and tags. " +
			"Use this to discover available examples before calling get_example.",
	}, m.ListExamples)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_workflow",
		Title:       "Validate Workflow",
		Description: "Validates a Zigflow workflow YAML string and returns structured errors by stage.",
	}, m.ValidateWorkflow)

	return m
}

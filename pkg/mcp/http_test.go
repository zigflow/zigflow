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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVersion = "test"

func testServer(t *testing.T) *mcp.Server {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "zigflow", Version: testVersion}, nil)
	New(server, testVersion)

	return server
}

func post(t *testing.T, url, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return res
}

// newHTTPHandler uses the official Streamable HTTP handler, so an initialise
// request must succeed and advertise the tools-only capabilities.
func TestNewHTTPHandler_Initialise(t *testing.T) {
	srv := httptest.NewServer(newHTTPHandler(testServer(t)))
	defer srv.Close()

	//nolint:misspell
	res := post(t, srv.URL, `{"jsonrpc":"2.0","id":1,"method":"initialize",`+
		`"params":{"protocolVersion":"2025-06-18","capabilities":{},`+
		`"clientInfo":{"name":"test","version":"1"}}}`)
	defer func() { _ = res.Body.Close() }()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"protocolVersion":"2025-06-18"`)
	assert.Contains(t, string(body), `"tools"`)
}

// Stateless mode pre-seeds the session state, so a non-initialise method must
// be accepted even without an Mcp-Session-Id header.
func TestNewHTTPHandler_SetLevelWithoutSession(t *testing.T) {
	srv := httptest.NewServer(newHTTPHandler(testServer(t)))
	defer srv.Close()

	res := post(t, srv.URL,
		`{"jsonrpc":"2.0","id":1,"method":"logging/setLevel","params":{"level":"debug"}}`)
	defer func() { _ = res.Body.Close() }()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"result":{}`)
	assert.NotContains(t, string(body), "invalid during session initialization")
}

// Request bodies larger than maxBytes must be rejected rather than read into
// memory unbounded.
func TestNewHTTPHandler_LargeBodyRejected(t *testing.T) {
	srv := httptest.NewServer(newHTTPHandler(testServer(t)))
	defer srv.Close()

	big := strings.Repeat("a", maxBytes+1)
	res := post(t, srv.URL,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"x":"`+big+`"}}`) //nolint:misspell
	defer func() { _ = res.Body.Close() }()

	assert.NotEqual(t, http.StatusOK, res.StatusCode)
}

// Cancelling the context must shut the server down cleanly with no error.
func TestHTTPHandler_GracefulShutdownOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- HTTPHandler(ctx, testServer(t), "127.0.0.1:0")
	}()

	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("HTTPHandler did not return after context cancellation")
	}
}

// A listen failure (invalid address) must be surfaced as an error.
func TestHTTPHandler_ListenError(t *testing.T) {
	err := HTTPHandler(context.Background(), testServer(t), "127.0.0.1:-1")
	assert.Error(t, err)
}

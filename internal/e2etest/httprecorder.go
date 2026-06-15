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

package e2etest

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// RecordedRequest is a single request captured by an HTTPRecorder.
type RecordedRequest struct {
	Method string
	Path   string
	Body   []byte
}

// HTTPRecorder is a local plain-HTTP server that records every request it
// receives and replies 200 with a fixed body. It lets an example reach an HTTP
// backend hermetically while the test asserts which calls were made and what
// they sent (for example that an activity attempt counter is present in a
// request body, not silently null).
type HTTPRecorder struct {
	// URL is the base http://host:port to point a worker's HTTP calls at.
	URL string

	mu       sync.Mutex
	requests []RecordedRequest
}

// Requests returns a copy of the requests recorded so far, in arrival order.
func (r *HTTPRecorder) Requests() []RecordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]RecordedRequest(nil), r.requests...)
}

// RequestsForPath returns the recorded requests whose path equals path.
func (r *HTTPRecorder) RequestsForPath(path string) []RecordedRequest {
	var out []RecordedRequest
	for _, req := range r.Requests() {
		if req.Path == path {
			out = append(out, req)
		}
	}
	return out
}

// StartHTTPRecorder starts the recorder on a loopback port. Every request is
// recorded and answered with 200 and responseBody. The server is shut down when
// the test finishes.
func StartHTTPRecorder(t *testing.T, responseBody string) *HTTPRecorder {
	t.Helper()

	rec := &HTTPRecorder{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		rec.mu.Lock()
		rec.requests = append(rec.requests, RecordedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   body,
		})
		rec.mu.Unlock()

		_, _ = io.WriteString(w, responseBody)
	})

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen for HTTP recorder")

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	})

	rec.URL = "http://" + listener.Addr().String()
	return rec
}

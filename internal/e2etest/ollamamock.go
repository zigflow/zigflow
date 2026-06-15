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
	"encoding/json"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// OllamaMock is a minimal stand-in for the Ollama /api/chat endpoint used by
// examples that drive a local model. It serves the single non-streaming chat
// route those examples call and delegates the assistant content to a
// caller-supplied reply func, so the example owns its (deterministic) model
// behaviour while the transport and call recording live here. It exists so an
// example can run hermetically, with no real Ollama and no model download.
type OllamaMock struct {
	// URL is the base http://host:port to pass to a worker as OLLAMA_HOST.
	URL string

	calls atomic.Int64
}

// Calls returns the number of /api/chat requests served so far. A test can
// assert it is non-zero to prove the model dependency was actually exercised
// rather than skipped by an activity's offline fallback.
func (m *OllamaMock) Calls() int {
	return int(m.calls.Load())
}

// ollamaChatMessage mirrors the subset of Ollama's chat message the examples
// send and receive: a role and its text content.
type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StartOllamaMock starts the mock on a loopback port. For each POST /api/chat
// request, reply is called with the first system prompt and the last user
// prompt from the request messages, and returns the assistant message content
// (which the caller's activity decodes as JSON). The server is shut down when
// the test finishes.
func StartOllamaMock(t *testing.T, reply func(system, user string) string) *OllamaMock {
	t.Helper()

	mock := &OllamaMock{}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []ollamaChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mock.calls.Add(1)

		var system, user string
		for _, msg := range req.Messages {
			switch msg.Role {
			case "system":
				if system == "" {
					system = msg.Content
				}
			case "user":
				user = msg.Content
			}
		}

		resp := struct {
			Message ollamaChatMessage `json:"message"`
		}{
			Message: ollamaChatMessage{Role: "assistant", Content: reply(system, user)},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen for Ollama mock")

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	})

	mock.URL = "http://" + listener.Addr().String()
	return mock
}

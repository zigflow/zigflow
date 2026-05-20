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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Messages []ollamaMessage `json:"messages"`
	Format   string          `json:"format,omitempty"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

func callOllama(ctx context.Context, systemPrompt, userPrompt string, target any) error {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:0.5b"
	}

	body, err := json.Marshal(ollamaRequest{
		Model:  model,
		Stream: false,
		Format: "json",
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling ollama request: %w", err)
	}

	// #nosec G704 -- OLLAMA_HOST is operator-supplied configuration; SSRF is a deployment concern, not a code defect
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(host, "/")+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	// #nosec G704 -- same rationale as above; URL is constructed from operator-supplied OLLAMA_HOST
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling ollama: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("ollama returned status %s", res.Status)
	}

	var decoded ollamaResponse
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return fmt.Errorf("decoding ollama response: %w", err)
	}

	if err := json.Unmarshal([]byte(decoded.Message.Content), target); err != nil {
		return fmt.Errorf("decoding model JSON: %w", err)
	}

	return nil
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func main() {
	const activityTaskQueue = "agent-worker"
	c, err := temporal.NewConnectionWithEnvvars(temporal.WithZerolog(&log.Logger))
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create Temporal client")
	}
	defer c.Close()

	w := worker.New(c, activityTaskQueue, worker.Options{})

	w.RegisterActivityWithOptions(planNextStep, activity.RegisterOptions{Name: "agent.PlanNextStep"})
	w.RegisterActivityWithOptions(lookup, activity.RegisterOptions{Name: "agent.Lookup"})
	w.RegisterActivityWithOptions(summarisePartialResult, activity.RegisterOptions{Name: "agent.SummarisePartialResult"})

	log.Info().Str("taskQueue", activityTaskQueue).Msg("Activity worker started - Waiting for commands")
	if err := w.Run(worker.InterruptCh()); err != nil {
		//nolint:gocritic
		log.Fatal().Err(err).Msg("Worker exited with error")
	}
}

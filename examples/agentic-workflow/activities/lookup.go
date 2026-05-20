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
	"context"
	"os"
	"strings"

	"go.temporal.io/sdk/activity"
)

type LookupRequest struct {
	Query string `json:"query"`
}

type LookupResponse struct {
	Answer string `json:"answer"`
	Source string `json:"source"`
}

const (
	lookupSourceFallback = "fallback:unavailable"
	lookupUnsureAnswer   = "I am unsure."
)

// lookup is a small Ollama-backed factual answer tool. Zigflow passes
// object-shaped arguments, so the request is a struct rather than a bare
// string.
//
// The activity is intentionally simple: it asks the local model for one short
// factual sentence and returns it. There is no retrieval, no knowledge base
// and no chain-of-thought prompting. When the local model is unavailable the
// activity returns an "I am unsure" answer rather than failing, so the agent
// loop can continue to make progress.
func lookup(ctx context.Context, req LookupRequest) (LookupResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running Ollama-backed lookup", "query", req.Query)

	systemPrompt := `You are a tiny factual lookup tool.
Answer the user's question in one short factual sentence.
If you are not sure, say "I am unsure.".
Reply only with JSON in the shape: { "answer": "<one short sentence>" }`

	var raw struct {
		Answer string `json:"answer"`
	}
	if err := callOllama(ctx, systemPrompt, req.Query, &raw); err != nil {
		logger.Info("Ollama unavailable for lookup; returning unsure", "error", err)
		return LookupResponse{
			Answer: lookupUnsureAnswer,
			Source: lookupSourceFallback,
		}, nil
	}

	answer := strings.TrimSpace(raw.Answer)
	if answer == "" {
		answer = lookupUnsureAnswer
	}

	return LookupResponse{
		Answer: answer,
		Source: ollamaSourceLabel(),
	}, nil
}

// ollamaSourceLabel reports the model identifier used for a successful lookup
// so the observation makes it clear where the answer came from.
func ollamaSourceLabel() string {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:0.5b"
	}
	return "ollama:" + model
}

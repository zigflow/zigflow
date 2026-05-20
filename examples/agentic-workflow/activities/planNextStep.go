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
	"fmt"
	"strings"

	"go.temporal.io/sdk/activity"
)

type PlanRequest struct {
	Question     string           `json:"question"`
	Observations []map[string]any `json:"observations"`
	Iteration    int              `json:"iteration"`
}

type PlanResponse struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

const (
	toolLookup      = "lookup"
	toolFinalAnswer = "final_answer"
	argQuery        = "query"
	argAnswer       = "answer"
)

// planNextStep decides the next agent action.
//
// The example is about Zigflow orchestration, not model quality, so the
// activity layers deterministic guardrails around the Ollama planner:
//
//  1. Try Ollama. Accept its response only if it parses into a supported
//     tool with non-empty arguments.
//  2. If Ollama is unavailable or returns nonsense, fall back deterministically:
//     - With no observations, plan a lookup of the original question.
//     - With observations, synthesise a best-effort final answer from the
//     most recent observation so the loop terminates predictably.
//
// The planner does not encode any per-question knowledge. Behaviour with a
// small model will vary, which is expected for the demo.
func planNextStep(ctx context.Context, req PlanRequest) (PlanResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Planning next agent step", "iteration", req.Iteration, "observations", len(req.Observations))

	if plan, ok := tryOllamaPlan(ctx, req); ok {
		logger.Info("Using planner output from Ollama", "tool", plan.Tool)
		return plan, nil
	}

	if len(req.Observations) == 0 {
		logger.Info("No observations yet; falling back to lookup of the original question")
		return PlanResponse{
			Tool: toolLookup,
			Arguments: map[string]any{
				argQuery: req.Question,
			},
		}, nil
	}

	logger.Info("Falling back to a best-effort final answer synthesised from observations")
	return PlanResponse{
		Tool: toolFinalAnswer,
		Arguments: map[string]any{
			argAnswer: synthesiseFromObservations(req),
		},
	}, nil
}

func tryOllamaPlan(ctx context.Context, req PlanRequest) (PlanResponse, bool) {
	systemPrompt := `You are a tiny ReAct planning agent.
You are given a question and a list of past observations from tool calls.
You must decide the next single action.

Tools:
- "lookup":       call a factual lookup tool. Arguments: { "query": "<short factual question>" }
- "final_answer": when the observations let you answer. Arguments: { "answer": "<short answer>" }

Guidance:
- Prefer "lookup" when the observations do not yet contain the fact you need.
- Prefer "final_answer" once one or two observations cover the question.
- If the question has multiple parts, use one lookup per part.

Reply only with JSON in this exact shape, no prose:
{ "tool": "lookup" | "final_answer", "arguments": { ... } }`

	userPrompt := fmt.Sprintf(
		"Question: %s\nObservations: %s\nIteration: %d",
		req.Question,
		mustJSON(req.Observations),
		req.Iteration,
	)

	var raw struct {
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := callOllama(ctx, systemPrompt, userPrompt, &raw); err != nil {
		activity.GetLogger(ctx).Info("Ollama unavailable or returned invalid output; falling back", "error", err)
		return PlanResponse{}, false
	}

	switch raw.Tool {
	case toolLookup:
		query, _ := raw.Arguments[argQuery].(string)
		if strings.TrimSpace(query) == "" {
			return PlanResponse{}, false
		}
		return PlanResponse{
			Tool:      toolLookup,
			Arguments: map[string]any{argQuery: query},
		}, true

	case toolFinalAnswer:
		answer, _ := raw.Arguments[argAnswer].(string)
		if strings.TrimSpace(answer) == "" {
			return PlanResponse{}, false
		}
		return PlanResponse{
			Tool:      toolFinalAnswer,
			Arguments: map[string]any{argAnswer: answer},
		}, true
	}

	return PlanResponse{}, false
}

// synthesiseFromObservations builds a deterministic fallback final answer
// using the most recent observation. It exists so the demo terminates
// predictably even when Ollama returns no plan at all.
func synthesiseFromObservations(req PlanRequest) string {
	if len(req.Observations) == 0 {
		return "I am unsure."
	}

	last := req.Observations[len(req.Observations)-1]
	output, _ := last["output"].(map[string]any)
	if answer, ok := output["answer"].(string); ok && strings.TrimSpace(answer) != "" {
		return fmt.Sprintf("Based on %d lookup(s): %s", len(req.Observations), answer)
	}

	return fmt.Sprintf(
		"Best-effort answer based on %d observation(s) for: %s",
		len(req.Observations),
		req.Question,
	)
}

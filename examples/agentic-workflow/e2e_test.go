//go:build e2e

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
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// These fixtures are the deterministic responses the mock Ollama returns. They
// stand in for a real model so the agent loop runs the same way every time:
// the planner first calls the lookup tool, then answers, and the lookup returns
// a known fact. The strings are distinctive so the assertions can prove the
// model dependency drove the result rather than an activity's offline fallback.
const (
	mockModel = "mock-model"

	lookupQuery         = "capital of France"
	mockLookupAnswer    = "Paris is the capital of France."
	mockFinalAnswer     = "The capital of France is Paris, which has five letters."
	planLookupContent   = `{"tool":"lookup","arguments":{"query":"` + lookupQuery + `"}}`
	planFinalContent    = `{"tool":"final_answer","arguments":{"answer":"` + mockFinalAnswer + `"}}`
	lookupAnswerContent = `{"answer":"` + mockLookupAnswer + `"}`
)

// TestAgenticWorkflowE2E runs the agentic-workflow example end to end. The
// workflow drives a bounded plan/act loop whose planner and lookup activities
// call a local Ollama model. The test owns Temporal, runs the example's own
// activity worker on the agent-worker queue, and points that worker at a mock
// Ollama so the run is hermetic: no real model, no model download, no internet.
//
// The example's workflow.yaml needs no rewriting; the only external wiring is
// the OLLAMA_HOST the activity worker reads, which is set to the mock.
func TestAgenticWorkflowE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	// Mock Ollama. The reply func encodes the deterministic model behaviour the
	// example expects: the lookup tool returns a fact, and the planner looks up
	// first (no observations yet) then answers (once an observation exists),
	// mirroring the activity's own documented logic.
	// Match the planner prompt first: it describes the lookup tool as a "factual
	// lookup tool" too, so only "planning agent" reliably distinguishes the two.
	mock := e2etest.StartOllamaMock(t, func(system, user string) string {
		switch {
		case strings.Contains(system, "planning agent"):
			if strings.Contains(user, "Observations: []") {
				return planLookupContent
			}
			return planFinalContent
		case strings.Contains(system, "factual lookup tool"):
			return lookupAnswerContent
		default:
			return `{}`
		}
	})

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	// The workflow worker serves the zigflow task queue; it does not call Ollama.
	e2etest.StartWorker(ctx, t, temporal.Address, workflowFile)

	// The example's activity worker serves the agent-worker queue and is the
	// only component that calls Ollama, so the mock is wired in here.
	e2etest.StartGoWorker(
		ctx, t, temporal.Address, "./activities",
		"OLLAMA_HOST="+mock.URL,
		"OLLAMA_MODEL="+mockModel,
	)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "agentic-workflow", map[string]any{
		"question":      "What is the capital of France?",
		"maxIterations": 3,
	})
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")

	pretty, _ := json.MarshalIndent(got, "", "  ")
	t.Logf("workflow output:\n%s", pretty)

	assertAgenticResult(t, got, mock)
}

func assertAgenticResult(t *testing.T, got map[string]any, mock *e2etest.OllamaMock) {
	t.Helper()

	// The mock must have been called, proving the agent loop exercised the model
	// dependency rather than an activity's offline fallback.
	assert.GreaterOrEqual(t, mock.Calls(), 2, "planner and lookup should both call Ollama")

	// The final answer is the planner's mock answer; the deterministic fallback
	// would produce a different "Based on ..."/"Best-effort ..." string, so an
	// exact match proves the planner used the model output.
	assert.Equal(t, mockFinalAnswer, got["answer"], "final answer comes from the planner")

	// One lookup happened before the planner answered.
	assert.Equal(t, float64(1), got["iterations"], "one tool call")

	observations, ok := got["observations"].([]any)
	require.True(t, ok, "observations should be an array")
	require.Len(t, observations, 1, "one observation recorded")

	observation, ok := observations[0].(map[string]any)
	require.True(t, ok, "observation should be an object")
	assert.Equal(t, "lookup", observation["tool"], "observation records the lookup tool")

	input, ok := observation["input"].(map[string]any)
	require.True(t, ok, "observation.input should be an object")
	assert.Equal(t, lookupQuery, input["query"], "lookup query echoes the planner argument")

	output, ok := observation["output"].(map[string]any)
	require.True(t, ok, "observation.output should be an object")
	assert.Equal(t, mockLookupAnswer, output["answer"], "lookup answer comes from the model")
	// The source label proves the lookup used the model, not its offline fallback
	// (which would be "fallback:unavailable").
	assert.Equal(t, "ollama:"+mockModel, output["source"], "lookup source records the model")

	// No branch result should carry an unresolved ${ ... } expression string.
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "${", "output must not contain unresolved expressions")
}

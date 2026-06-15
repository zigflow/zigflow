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
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// jsonplaceholderHost is the external host the basic workflow calls. It is
// hardcoded in workflow.yaml, so the test intercepts it locally to stay
// hermetic.
const jsonplaceholderHost = "jsonplaceholder.typicode.com"

// user3JSON and user2JSON mirror the real jsonplaceholder fixtures the example
// expects. The mock serves these so the workflow behaves identically to
// production without touching the internet.
const (
	user3JSON = `{
		"id": 3,
		"name": "Clementine Bauch",
		"username": "Samantha",
		"email": "Nathan@yesenia.net",
		"address": {
			"street": "Douglas Extension",
			"suite": "Suite 847",
			"city": "McKenziehaven",
			"zipcode": "59590-4157",
			"geo": {"lat": "-68.6102", "lng": "-47.0653"}
		},
		"phone": "1-463-123-4447",
		"website": "ramiro.info",
		"company": {
			"name": "Romaguera-Jacobson",
			"catchPhrase": "Face to face bifurcated interface",
			"bs": "e-enable strategic applications"
		}
	}`
	user2JSON = `{
		"id": 2,
		"name": "Ervin Howell",
		"username": "Antonette",
		"email": "Shanna@melissa.tv",
		"address": {
			"street": "Victor Plains",
			"suite": "Suite 879",
			"city": "Wisokyburgh",
			"zipcode": "90566-7771",
			"geo": {"lat": "-43.9509", "lng": "-34.4618"}
		},
		"phone": "010-692-6593 x09125",
		"website": "anastasia.net",
		"company": {
			"name": "Deckow-Crist",
			"catchPhrase": "Proactive didactic contingency",
			"bs": "synergize scalable supply-chains"
		}
	}`
)

// TestBasicE2E runs the basic example end to end against a Temporal dev server
// and a local HTTPS mock, both owned by the test. The workflow mixes
// deterministic fields with run-specific values (UUIDs, clocks), so stable
// fields are asserted exactly and variable fields are checked by shape.
func TestBasicE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	user3 := decodeJSON(t, user3JSON)
	user2 := decodeJSON(t, user2JSON)

	mock := e2etest.StartHTTPSMock(t, []string{jsonplaceholderHost}, map[string]any{
		"/users/3": user3,
		"/users/2": user2,
	})

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	// Keep the worker's Temporal connection direct; only the external HTTPS host
	// is routed through the mock.
	temporalHost, _, err := net.SplitHostPort(temporal.Address)
	require.NoError(t, err)

	workerEnv := append(mock.WorkerEnv(temporalHost), "ZIGGY_EXAMPLE_ENVVAR=some-example-envvar")
	e2etest.StartWorkerWithEnv(ctx, t, workerEnv, temporal.Address, workflowFile)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	// The workflow waits 5s plus a fork with further waits, so allow headroom.
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "basic", map[string]any{"userId": 3})
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")

	pretty, _ := json.MarshalIndent(got, "", "  ")
	t.Logf("workflow output:\n%s", pretty)

	assertBasicResult(t, got, user3, user2)
}

func assertBasicResult(t *testing.T, got, user3, user2 map[string]any) {
	t.Helper()

	// External call results are served from the mock, so assert them exactly.
	assert.Equal(t, user3, got["getUser"], "getUser")
	assert.Equal(t, user3, got["callDoctor"], "callDoctor")
	assert.Equal(t, user2, got["callNurse"], "callNurse")

	baseData, ok := got["baseData"].(map[string]any)
	require.True(t, ok, "baseData should be an object")

	// Variable fields: shape only.
	e2etest.AssertValidUUID(t, baseData["uuid"])
	e2etest.AssertNumeric(t, baseData["now"])
	e2etest.AssertIntegerLike(t, baseData["timestamp"])
	e2etest.AssertDatetimeString(t, baseData["now_formatted"])
	e2etest.AssertDatetimeString(t, baseData["timestamp_formatted"])
	e2etest.AssertISO8601UTC(t, baseData["now_iso8601"])

	object, ok := baseData["object"].(map[string]any)
	require.True(t, ok, "baseData.object should be an object")
	e2etest.AssertValidUUID(t, object["uuid"])

	array, ok := baseData["array"].([]any)
	require.True(t, ok, "baseData.array should be an array")
	require.Len(t, array, 2, "baseData.array length")
	e2etest.AssertValidUUID(t, array[0])

	// Deterministic fields: assert directly without mutating the result.
	assert.Equal(t, "some-example-envvar", baseData["envvar"], "baseData.envvar")
	assert.Equal(t, float64(3), baseData["inputUserId"], "baseData.inputUserId")
	assert.Equal(t, "world", object["hello"], "baseData.object.hello")
	assert.Equal(t, map[string]any{"hello": "world"}, array[1], "baseData.array[1]")
}

// decodeJSON parses a JSON object literal into a map. Decoding the fixtures the
// same way the workflow result is decoded keeps their types aligned (every JSON
// number becomes float64), so require.Equal compares cleanly.
func decodeJSON(t *testing.T, s string) map[string]any {
	t.Helper()

	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(s), &out))
	return out
}

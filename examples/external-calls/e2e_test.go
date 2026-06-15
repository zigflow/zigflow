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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// jsonplaceholderHost is the external host the HTTP branch calls. It is
// hardcoded in workflow.yaml, so the test intercepts it locally to stay
// hermetic.
const jsonplaceholderHost = "jsonplaceholder.typicode.com"

// grpcInput is the value supplied to the gRPC Command1 method via
// $env.GRPC_INPUT (ZIGGY_GRPC_INPUT below). The basic gRPC service echoes it
// back inside a fixed sentence, so the expected output is fully deterministic.
const grpcInput = "some-input"

// expectedGRPCOutput is what the basic BasicService.Command1 returns for
// grpcInput: the cmd.client.Run sentence in examples/external-calls/grpc/basic.
const expectedGRPCOutput = "This has executed some-input and the connection is root:password@localhost:3306"

// user3JSON mirrors the real jsonplaceholder fixture for user 3, matching the
// data used by examples/basic. The mock serves it so the HTTP branch behaves
// identically to production without touching the internet.
const user3JSON = `{
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

// TestExternalCallsE2E runs the external-calls example end to end. The workflow
// is a non-competing fork with a gRPC branch and an HTTP branch.
//
// The gRPC backend runs as the basic BasicService in a Testcontainers
// container, which stands in for the compose "grpc" service; the HTTP endpoint
// is served by a local HTTPS mock, and Temporal is owned by the test. Docker
// Compose is not involved: the test manages every dependency itself, and the
// workflow is started directly through the Temporal Go client rather than via
// the compose "starter" service.
func TestExternalCallsE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	// Start the gRPC backend. The basic service listens on :3000 inside the
	// container, the same port the compose "grpc" service exposes.
	grpcDir, err := filepath.Abs(filepath.Join("grpc", "basic"))
	require.NoError(t, err)
	grpc := e2etest.StartGoServiceContainer(ctx, t, grpcDir, "3000/tcp")

	// Reach the container through a loopback forwarder. Zigflow validates
	// service.host as an RFC 1123 hostname and rejects bare IPs, but under
	// Docker-in-Docker the container is only reachable by its gateway IP. The
	// forwarder bridges "localhost" to that address so the workflow can use a
	// valid hostname.
	forwarder := e2etest.StartLoopbackForwarder(ctx, t, grpc.Address)

	// Serve the HTTP branch's external host from a local HTTPS mock.
	user3 := decodeJSON(t, user3JSON)
	mock := e2etest.StartHTTPSMock(t, []string{jsonplaceholderHost}, map[string]any{
		"/users/3": user3,
	})

	// The committed workflow.yaml encodes the Docker Compose topology: it reaches
	// the gRPC backend at "grpc:3000" and the proto file by its in-container path.
	// This test runs the worker on the host against Testcontainers-managed
	// dependencies, so writeHostWorkflow rewrites only those runtime wiring fields
	// (gRPC host, gRPC port, proto endpoint path), leaving the workflow's
	// semantics untouched. service.host and service.port are typed, validated
	// fields (host is a hostname, port an int), so unlike arguments.input they
	// cannot be $env expressions; a test-specific copy is the cleanest option.
	workflowFile := writeHostWorkflow(t, forwarder.Host, forwarder.Port)

	// Keep the worker's Temporal connection direct; only the external HTTPS host
	// is routed through the mock. The gRPC call targets localhost (the
	// forwarder), which is already in the mock's default NO_PROXY, so it bypasses
	// the proxy too.
	temporalHost, _, err := net.SplitHostPort(temporal.Address)
	require.NoError(t, err)

	workerEnv := append(mock.WorkerEnv(temporalHost), "ZIGGY_GRPC_INPUT="+grpcInput)
	e2etest.StartWorkerWithEnv(ctx, t, workerEnv, temporal.Address, workflowFile)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "external-calls")
	require.NoError(t, err, "execute workflow")

	var got map[string]any
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")

	pretty, _ := json.MarshalIndent(got, "", "  ")
	t.Logf("workflow output:\n%s", pretty)

	assertExternalCallsResult(t, got, user3)
}

func assertExternalCallsResult(t *testing.T, got, user3 map[string]any) {
	t.Helper()

	// The fork keys each branch's result under the branch name.
	assert.Equal(t, user3, got["http"], "http branch returns the mocked user 3")

	grpcBranch, ok := got["grpc"].(map[string]any)
	require.True(t, ok, "grpc branch should be an object")
	assert.Equal(t, expectedGRPCOutput, grpcBranch["output"], "grpc Command1 output")

	// No branch result should carry an unresolved ${ ... } expression string.
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "${", "output must not contain unresolved expressions")
}

// writeHostWorkflow adapts the canonical workflow.yaml for a host-side worker
// running against Testcontainers-managed dependencies. The committed workflow
// targets the Docker Compose topology, so this rewrites only the runtime wiring
// fields that the host-side, container-per-dependency setup changes:
//
//   - gRPC host    ("grpc" -> the loopback forwarder host)
//   - gRPC port    (3000   -> the forwarder's mapped port)
//   - proto endpoint path (in-container path -> the host path)
//
// Everything else, including the HTTP branch and the ${ $env.GRPC_INPUT }
// argument, is preserved verbatim: this adapts the container/compose wiring
// assumptions without changing the workflow's semantics. The patched copy is
// written to a temp file and its path returned.
//
// Each replacement is guarded so the test fails loudly if workflow.yaml drifts
// from the substrings this rewrite depends on, rather than silently testing the
// unpatched (container-only) values.
func writeHostWorkflow(t *testing.T, host, port string) string {
	t.Helper()

	source, err := os.ReadFile("workflow.yaml")
	require.NoError(t, err, "read workflow.yaml")

	protoPath, err := filepath.Abs(filepath.Join("grpc", "basic", "proto", "basic", "v1", "basic.proto"))
	require.NoError(t, err)

	replacements := []struct{ old, new string }{
		{
			old: "file:///go/app/examples/external-calls/grpc/basic/proto/basic/v1/basic.proto",
			new: "file://" + protoPath,
		},
		{old: "host: grpc", new: "host: " + host},
		{old: "port: 3000", new: "port: " + port},
	}

	patched := string(source)
	for _, r := range replacements {
		require.Equalf(t, 1, strings.Count(patched, r.old),
			"expected exactly one occurrence of %q in workflow.yaml; the example may have drifted", r.old)
		patched = strings.Replace(patched, r.old, r.new, 1)
	}

	// dest is rooted at the test's own temp dir with a fixed name, so the path
	// is fully test-controlled despite gosec's taint analysis flagging it.
	dest := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(dest, []byte(patched), 0o600), "write patched workflow") //nolint:gosec // test-controlled path

	return dest
}

// decodeJSON parses a JSON object literal into a map. Decoding the fixture the
// same way the workflow result is decoded keeps their types aligned (every JSON
// number becomes float64), so assert.Equal compares cleanly.
func decodeJSON(t *testing.T, s string) map[string]any {
	t.Helper()

	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(s), &out))
	return out
}

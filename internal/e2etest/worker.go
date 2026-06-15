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
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// zigflowModule is the module path of the Zigflow binary. Running it with
// "go run" resolves within the current module regardless of the test's working
// directory, so an example can start a real worker without importing internal
// command wiring.
const zigflowModule = "github.com/zigflow/zigflow"

// runSubcommand is the Zigflow subcommand that starts workers.
const runSubcommand = "run"

// BinaryEnvVar names the environment variable that, when set, points at a
// prebuilt Zigflow binary. "task e2e" builds the binary once and exports this
// so example tests skip the per-test "go run" compile step.
const BinaryEnvVar = "ZIGFLOW_E2E_BINARY"

// bindInUseMarker is the substring a worker logs when it fails to bind a listen
// address. The metrics port binds 127.0.0.1:0 and so never collides, leaving
// the pre-allocated health port as the only address that can produce this, so
// seeing it means a health-port collision the caller can retry past.
const bindInUseMarker = "address already in use"

// workerStartAttempts bounds how many times StartWorkerWithEnv retries a worker
// that died from a health-port collision before giving up.
const workerStartAttempts = 3

// workerCommand returns the program and leading arguments used to start the
// Zigflow worker. It prefers a prebuilt binary named by ZIGFLOW_E2E_BINARY and
// falls back to "go run" for direct local invocations such as
// "go test -tags=e2e ./examples/...".
func workerCommand(t *testing.T) (program string, args []string) {
	t.Helper()

	if binary := os.Getenv(BinaryEnvVar); binary != "" {
		// The path is operator-supplied via the environment for this test
		// helper, so the traversal is intentional.
		info, err := os.Stat(binary) //nolint:gosec // test-controlled path
		require.NoErrorf(t, err, "%s=%q is set but not usable", BinaryEnvVar, binary)
		require.Falsef(t, info.IsDir(), "%s=%q is a directory, not a binary", BinaryEnvVar, binary)
		return binary, []string{runSubcommand}
	}

	t.Logf("%s not set; falling back to \"go run\" (slower)", BinaryEnvVar)
	return "go", []string{"run", zigflowModule, runSubcommand}
}

// StartWorker runs the Zigflow worker as a subprocess against the given
// Temporal address with the supplied workflow files registered. It blocks until
// the worker reports ready, then returns. The worker is killed automatically
// when the test finishes.
//
// Worker stdout and stderr are streamed to the test log, so a failure shows the
// worker output inline rather than requiring a separate container inspection.
func StartWorker(ctx context.Context, t *testing.T, temporalAddress string, workflowFiles ...string) {
	t.Helper()
	StartWorkerWithEnv(ctx, t, nil, temporalAddress, workflowFiles...)
}

// StartWorkerWithEnv behaves like StartWorker but appends extraEnv (each entry
// "KEY=value") to the worker process environment. This lets an example inject
// configuration the worker reads at runtime, for example $env workflow values
// (ZIGGY_*) or proxy settings that point HTTP calls at a local mock, without
// leaking those variables into the test process itself.
func StartWorkerWithEnv(ctx context.Context, t *testing.T, extraEnv []string, temporalAddress string, workflowFiles ...string) {
	t.Helper()

	// The health port is pre-allocated so the readiness probe knows where to
	// poll. Unlike the metrics port (which binds 127.0.0.1:0 and never clashes),
	// that pre-allocation leaves a window in which a parallel worker can grab the
	// same port, after which this worker exits with "address already in use". A
	// collision is purely transient, so retry with a fresh port a few times
	// before failing. Non-collision startup failures still fail immediately.
	for attempt := 1; attempt <= workerStartAttempts; attempt++ {
		if startWorkerAttempt(ctx, t, attempt, extraEnv, temporalAddress, workflowFiles...) {
			return
		}
	}

	t.Fatalf("worker never became ready: health port was already in use on all %d attempts", workerStartAttempts)
}

// startWorkerAttempt starts the worker once on a fresh health port and waits for
// it to report ready. It returns true once ready. It returns false only when the
// worker exited because its health port was already in use, signalling the
// caller to retry; any other startup failure fails the test via t.Fatalf so real
// problems are not retried or hidden.
func startWorkerAttempt(
	ctx context.Context, t *testing.T, attempt int, extraEnv []string, temporalAddress string, workflowFiles ...string,
) bool {
	t.Helper()

	healthPort := freePort(t)

	program, args := workerCommand(t)
	for _, f := range workflowFiles {
		args = append(args, "--file", f)
	}
	args = append(
		args,
		"--temporal-address", temporalAddress,
		"--health-listen-address", fmt.Sprintf("127.0.0.1:%d", healthPort),
		// Bind metrics to port 0 so the kernel assigns a free port at bind time.
		// The test never scrapes metrics, so the chosen port need not be known;
		// letting the OS pick avoids both the default-9090 clash between parallel
		// workers and the race inherent in pre-allocating a numbered port.
		"--metrics-listen-address", "127.0.0.1:0",
	)

	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	// Run in its own process group so the whole tree can be signalled on
	// cleanup, including the compiled worker that "go run" execs.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "worker stdout pipe")
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err, "worker stderr pipe")

	require.NoError(t, cmd.Start(), "start worker")

	// Scan the worker output (streaming it to the test log) and watch for the
	// bind-collision marker.
	var sawBindCollision atomic.Bool
	var streams sync.WaitGroup
	streams.Add(2)
	go scanWorkerOutput(t, stdout, &sawBindCollision, &streams)
	go scanWorkerOutput(t, stderr, &sawBindCollision, &streams)

	// Drain the pipes to EOF before reaping: cmd.Wait closes them, so reaping
	// first could drop the worker's final lines (including the bind error).
	// exited closes only once the output is fully scanned and the process
	// reaped, so sawBindCollision is settled by the time it is observed.
	exited := make(chan struct{})
	go func() {
		streams.Wait()
		_ = cmd.Wait()
		close(exited)
	}()

	if workerReady(ctx, healthPort, exited) {
		t.Cleanup(func() {
			// Kill the whole process group; ignore errors as the process may
			// have already exited. Then wait for it to be reaped.
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			<-exited
		})
		return true
	}

	// Not ready. Stop the process so its pipes close and the reaper completes,
	// which also settles sawBindCollision, then classify the failure.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	<-exited

	if sawBindCollision.Load() {
		t.Logf("worker start attempt %d/%d: health port %d already in use; retrying with a fresh port",
			attempt, workerStartAttempts, healthPort)
		return false
	}

	t.Fatalf("worker did not become ready (attempt %d/%d, health port %d); see worker output above",
		attempt, workerStartAttempts, healthPort)
	return false
}

// StartGoWorker runs an in-repo Go worker package (for example an example's
// own activity worker) as a subprocess via "go run", connecting to Temporal
// through the standard TEMPORAL_ADDRESS environment variable. It is used by
// examples whose activities run on a separate task queue served by their own
// worker. The process is killed when the test finishes.
//
// Unlike StartWorker there is no readiness probe: such workers expose no health
// endpoint, and Temporal holds the task on the queue until the worker polls, so
// the workflow simply waits for it.
func StartGoWorker(ctx context.Context, t *testing.T, temporalAddress, pkg string, extraEnv ...string) {
	t.Helper()

	cmd := exec.CommandContext(ctx, "go", "run", pkg)
	cmd.Env = append(os.Environ(), "TEMPORAL_ADDRESS="+temporalAddress)
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "go worker stdout pipe")
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err, "go worker stderr pipe")

	require.NoError(t, cmd.Start(), "start go worker")

	go streamToTestLog(t, "go-worker", stdout)
	go streamToTestLog(t, "go-worker", stderr)

	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})
}

// workerReady polls the worker readiness endpoint until it returns 200 OK,
// returning true once ready. It returns false if the worker process exits first
// (exited is closed) or the timeout elapses, leaving the caller to classify the
// failure. It never fails the test itself.
func workerReady(ctx context.Context, healthPort int, exited <-chan struct{}) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/readyz", healthPort)
	deadline := time.Now().Add(2 * time.Minute)

	for time.Now().Before(deadline) {
		select {
		case <-exited:
			return false // worker exited before becoming ready
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err == nil {
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return true
				}
			}
		}

		// Wait before the next poll, but wake immediately if the worker exits.
		select {
		case <-exited:
			return false
		case <-time.After(500 * time.Millisecond):
		}
	}

	return false
}

// scanWorkerOutput copies worker output line by line into the test log and flags
// when a line reports a listen-address bind collision. It signals done when the
// reader reaches EOF.
func scanWorkerOutput(t *testing.T, r io.Reader, sawBindCollision *atomic.Bool, done *sync.WaitGroup) {
	t.Helper()
	defer done.Done()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, bindInUseMarker) {
			sawBindCollision.Store(true)
		}
		t.Logf("[worker] %s", line)
	}

	if err := scanner.Err(); err != nil {
		t.Logf("[worker] error reading worker output: %v", err)
	}
}

// streamToTestLog copies worker output line by line into the test log.
func streamToTestLog(t *testing.T, prefix string, r io.Reader) {
	t.Helper()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		t.Logf("[%s] %s", prefix, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.Logf("[%s] error reading worker output: %v", prefix, err)
	}
}

// freePort asks the kernel for an available TCP port.
func freePort(t *testing.T) int {
	t.Helper()

	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "allocate free port")
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port
}

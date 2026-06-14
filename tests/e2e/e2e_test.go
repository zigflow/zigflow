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

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"github.com/zigflow/zigflow/tests/e2e/utils"

	_ "github.com/zigflow/zigflow/tests/e2e/tests"
)

type harness struct {
	Cases []utils.TestCase
	// ExampleFiles are the workflow.yaml paths of every discovered example. They
	// are loaded into one shared worker rather than a worker per case.
	ExampleFiles []string
}

var h *harness

// zigflowBin is the path to the zigflow binary built once in TestMain. Tests
// exec this prebuilt binary instead of "go run ." so the module is compiled a
// single time, rather than once per parallel test case where the invocations
// would otherwise serialise behind the Go build lock.
var zigflowBin string

// buildZigflow compiles the zigflow binary once into a temporary directory and
// returns its path together with a cleanup function to remove it.
func buildZigflow() (bin string, cleanup func(), err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}

	dir, err := os.MkdirTemp("", "zigflow-e2e-")
	if err != nil {
		return "", nil, err
	}

	bin = path.Join(dir, "zigflow")

	cmd := exec.Command("go", "build", "-o", bin, ".") //nolint
	cmd.Dir = path.Join(cwd, "..", "..")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, fmt.Errorf("building zigflow binary: %w", err)
	}

	return bin, func() { _ = os.RemoveAll(dir) }, nil
}

// startWorker launches a zigflow worker for the given workflow files and returns
// a function that kills it (and its process group). The worker polls in the
// background; callers do not wait for it to become ready because Temporal queues
// the workflow task until a poller picks it up.
func startWorker(ctx context.Context, workDir string, files ...string) (stop func(), err error) {
	healthPort, err := getFreePort()
	if err != nil {
		return nil, err
	}
	metricsPort, err := getFreePort()
	if err != nil {
		return nil, err
	}

	args := []string{"run"}
	for _, f := range files {
		args = append(args, "--file", f)
	}
	args = append(
		args,
		"--health-listen-address", fmt.Sprintf("localhost:%d", healthPort),
		"--metrics-listen-address", fmt.Sprintf("localhost:%d", metricsPort),
	)

	cmd := exec.CommandContext(ctx, zigflowBin, args...)
	cmd.Env = os.Environ()
	cmd.Dir = workDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }, nil
}

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer func() {
				err = l.Close()
			}()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return port, err
}

func setup() (*harness, error) {
	cases := make([]utils.TestCase, 0)

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for _, c := range utils.GetTestCases() {
		c.WorkflowPath = path.Join(cwd, "tests", c.Name, c.WorkflowPath)

		for i, f := range c.ExtraFiles {
			c.ExtraFiles[i] = path.Join(cwd, "tests", c.Name, f)
		}

		workflowDefinition, err := zigflow.LoadFromFile(c.WorkflowPath)
		if err != nil {
			return nil, err
		}
		c.Workflow = workflowDefinition
		cases = append(cases, c)
	}

	// Auto-discover example-based tests. These already carry an absolute
	// WorkflowPath, so they are loaded directly rather than joined against the
	// tests/ directory like the registry cases above.
	exampleCases, err := utils.DiscoverExamples(path.Join(cwd, "..", "..", "examples"))
	if err != nil {
		return nil, err
	}

	exampleFiles := make([]string, 0, len(exampleCases))
	for _, c := range exampleCases {
		workflowDefinition, err := zigflow.LoadFromFile(c.WorkflowPath)
		if err != nil {
			return nil, err
		}
		c.Workflow = workflowDefinition
		cases = append(cases, c)
		exampleFiles = append(exampleFiles, c.WorkflowPath)
	}

	return &harness{
		Cases:        cases,
		ExampleFiles: exampleFiles,
	}, nil
}

func TestMain(m *testing.M) {
	logLevel := "info"
	if l, ok := os.LookupEnv("LOG_LEVEL"); ok {
		logLevel = l
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Printf("logger setup failed: %v", err)
		// Non-zero exit so the test run fails clearly.
		os.Exit(1)
	}

	zerolog.SetGlobalLevel(level)

	// Reuse a binary built by the caller (task e2e builds it while the Docker
	// stack starts, so the two overlap). Fall back to building one here so a
	// direct `go test -tags=e2e` still works on its own.
	cleanup := func() {}
	if bin := os.Getenv("ZIGFLOW_E2E_BINARY"); bin != "" {
		zigflowBin = bin
	} else {
		bin, c, err := buildZigflow()
		if err != nil {
			log.Printf("e2e binary build failed: %v", err)
			// Non-zero exit so the test run fails clearly.
			os.Exit(1)
		}
		zigflowBin = bin
		cleanup = c
	}

	testHarness, err := setup()
	if err != nil {
		log.Printf("e2e setup failed: %v", err)
		cleanup()
		// Non-zero exit so the test run fails clearly.
		os.Exit(1)
	}
	h = testHarness

	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("e2e getwd failed: %v", err)
		cleanup()
		os.Exit(1)
	}

	// One shared worker serves every example. They run on the same task queue,
	// so a worker per example would route workflow tasks between each other and
	// stall. The repo root is the working directory so file:// references in
	// example workflows resolve.
	stopExamples := func() {}
	if len(h.ExampleFiles) > 0 {
		stop, err := startWorker(context.Background(), path.Join(cwd, "..", ".."), h.ExampleFiles...)
		if err != nil {
			log.Printf("e2e example worker failed to start: %v", err)
			cleanup()
			os.Exit(1)
		}
		stopExamples = stop
	}

	code := m.Run()
	stopExamples()
	cleanup()
	os.Exit(code)
}

func TestE2E(t *testing.T) {
	if h == nil {
		t.Fatal("harness is nil - setup not run")
	}

	cwd, err := os.Getwd()
	assert.NoError(t, err, "working directory")

	repoRoot := path.Join(cwd, "..", "..")

	for _, test := range h.Cases {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			// Example cases are served by the shared worker started in TestMain.
			// Bespoke cases each get their own worker on their own task queue.
			if !test.Example {
				files := append([]string{test.WorkflowPath}, test.ExtraFiles...)
				stop, err := startWorker(t.Context(), repoRoot, files...)
				assert.NoError(t, err, "start worker")
				t.Cleanup(stop)
			}

			test.Test(t, &test)
		})
	}
}

// TestE2EMultiFileDuplicateRejected asserts that Zigflow exits with a non-zero
// status and emits a clear error when two workflow files define the same
// workflow name on the same task queue. This must be caught at startup, before
// any worker is started.
func TestE2EMultiFileDuplicateRejected(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	fixtureDir := path.Join(cwd, "tests", "multi-file-duplicate")

	healthPort, err := getFreePort()
	require.NoError(t, err)

	metricsPort, err := getFreePort()
	require.NoError(t, err)

	var stderr bytes.Buffer
	cmd := exec.Command( //nolint
		zigflowBin,
		"run",
		"--file", path.Join(fixtureDir, "workflow-a.yaml"),
		"--file", path.Join(fixtureDir, "workflow-b.yaml"),
		"--health-listen-address", fmt.Sprintf("localhost:%d", healthPort),
		"--metrics-listen-address", fmt.Sprintf("localhost:%d", metricsPort),
	)
	cmd.Env = os.Environ()
	cmd.Dir = path.Join(cwd, "..", "..")
	cmd.Stderr = &stderr

	// The process must exit on its own - give it enough time to compile and
	// run startup validation, but not so long that a hanging process blocks CI.
	done := make(chan error, 1)
	require.NoError(t, cmd.Start())
	go func() { done <- cmd.Wait() }()

	select {
	case runErr := <-done:
		assert.Error(t, runErr, "expected Zigflow to exit with a non-zero status")
		output := stderr.String()
		assert.True(
			t,
			strings.Contains(output, "Duplicate workflow name") ||
				strings.Contains(output, "duplicate"),
			"expected duplicate-name error in stderr, got: %s", output,
		)
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		t.Fatal("Zigflow did not exit within the timeout - duplicate detection may not be working")
	}
}

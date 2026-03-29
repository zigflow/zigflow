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
}

var h *harness

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

	return &harness{
		Cases: cases,
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

	testHarness, err := setup()
	if err != nil {
		log.Printf("e2e setup failed: %v", err)
		// Non-zero exit so the test run fails clearly.
		os.Exit(1)
	}
	h = testHarness

	code := m.Run()
	os.Exit(code)
}

func TestE2E(t *testing.T) {
	if h == nil {
		t.Fatal("harness is nil - setup not run")
	}

	cwd, err := os.Getwd()
	assert.NoError(t, err, "working directory")

	for _, test := range h.Cases {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			healthPort, err := getFreePort()
			assert.NoError(t, err, "health port")

			metricsPort, err := getFreePort()
			assert.NoError(t, err, "metrics port")

			args := []string{
				"run",
				".",
				"run",
				"--file", test.WorkflowPath,
			}
			for _, f := range test.ExtraFiles {
				args = append(args, "--file", f)
			}
			args = append(args,
				"--health-listen-address", fmt.Sprintf("localhost:%d", healthPort),
				"--metrics-listen-address", fmt.Sprintf("localhost:%d", metricsPort),
			)

			// Start the Zigflow binary with the loaded workflow
			go (func() {
				//nolint
				cmd := exec.CommandContext(ctx, "go", args...)
				cmd.Env = os.Environ()
				cmd.Dir = path.Join(cwd, "..", "..")
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Setpgid: true,
				}
				assert.NoError(t, cmd.Start())

				t.Cleanup(func() {
					syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				})
			})()

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
	cmd := exec.Command("go", //nolint
		"run", ".",
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
		assert.True(t,
			strings.Contains(output, "Duplicate workflow name") ||
				strings.Contains(output, "duplicate"),
			"expected duplicate-name error in stderr, got: %s", output,
		)
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		t.Fatal("Zigflow did not exit within the timeout - duplicate detection may not be working")
	}
}

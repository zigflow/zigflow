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
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"syscall"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
				"--health-listen-address", fmt.Sprintf("localhost:%d", healthPort),
				"--metrics-listen-address", fmt.Sprintf("localhost:%d", metricsPort),
			}

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

			test.Test(t, test)
		})
	}
}

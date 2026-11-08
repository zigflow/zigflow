//go:build !windows

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

package cmd

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTestSignalContextSecondInterrupt(t *testing.T) {
	if os.Getenv("ZIGFLOW_TEST_SIGNAL_HELPER") == "1" {
		ctx, stop := testSignalContext(context.Background())
		defer stop()
		_, _ = os.Stdout.WriteString("ready:" + strconv.Itoa(os.Getpid()) + "\n")
		<-ctx.Done()
		_, _ = os.Stdout.WriteString("cancelled\n")
		time.Sleep(10 * time.Second) // Simulate a slow Temporal cleanup.
		return
	}

	commandCtx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(commandCtx, "go", "test", "-v", "-run", "^TestTestSignalContextSecondInterrupt$", ".")
	command.Env = append(os.Environ(), "ZIGFLOW_TEST_SIGNAL_HELPER=1")
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, command.Start())
	defer func() { _ = command.Process.Kill() }()

	lines := make(chan string, 2)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	expectLine := func(want string) string {
		t.Helper()
		deadline := time.After(30 * time.Second)
		for {
			select {
			case line, ok := <-lines:
				require.True(t, ok, "helper exited before %q", want)
				if strings.Contains(line, want) {
					return strings.TrimPrefix(line, "    ")
				}
			case <-deadline:
				t.Fatalf("helper did not report %q", want)
			}
		}
	}

	ready := expectLine("ready:")
	pid, err := strconv.Atoi(strings.TrimPrefix(ready, "ready:"))
	require.NoError(t, err)
	require.NoError(t, syscall.Kill(pid, syscall.SIGINT))
	expectLine("cancelled")
	time.Sleep(100 * time.Millisecond) // Allow the cancellation callback to run.
	require.NoError(t, syscall.Kill(pid, syscall.SIGINT))

	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case waitErr := <-done:
		require.Error(t, waitErr)
	case <-time.After(2 * time.Second):
		_ = command.Process.Kill()
		waitErr := <-done
		t.Fatalf("second interrupt was swallowed during cleanup: %v", waitErr)
	}
}

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
	"fmt"
	"os"
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
	stdout, outputWriter, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = stdout.Close() }()
	binary, err := os.Executable()
	require.NoError(t, err)
	process, err := os.StartProcess(binary, []string{binary, "-test.v", "-test.run=^TestTestSignalContextSecondInterrupt$"},
		&os.ProcAttr{
			Env:   append(os.Environ(), "ZIGFLOW_TEST_SIGNAL_HELPER=1"),
			Files: []*os.File{os.Stdin, outputWriter, os.Stderr},
		})
	require.NoError(t, outputWriter.Close())
	require.NoError(t, err)
	stopOnDeadline := context.AfterFunc(commandCtx, func() { _ = process.Kill() })
	defer stopOnDeadline()
	defer func() { _ = process.Kill() }()

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
	go func() {
		state, waitErr := process.Wait()
		if waitErr == nil && state.Success() {
			waitErr = fmt.Errorf("helper exited successfully after second interrupt")
		}
		done <- waitErr
	}()
	select {
	case waitErr := <-done:
		require.NoError(t, waitErr)
	case <-time.After(2 * time.Second):
		_ = process.Kill()
		waitErr := <-done
		t.Fatalf("second interrupt was swallowed during cleanup: %v", waitErr)
	}
}

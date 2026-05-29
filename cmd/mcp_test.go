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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const transportFlag = "--transport"

// An unknown transport must fail before any server is started.
func TestMCPCmd_InvalidTransport(t *testing.T) {
	cmd := newMCPCmd()
	cmd.SetArgs([]string{transportFlag, "bogus"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transport type")
}

// The stdio branch must be selected for --transport stdio. A pre-cancelled
// context lets server.Run return immediately, so the test needs no interactive
// stdio session and stays deterministic.
func TestMCPCmd_StdioSelected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := newMCPCmd()
	cmd.SetArgs([]string{transportFlag, "stdio"})

	err := cmd.ExecuteContext(ctx)
	// stdio was selected: we get the context error, never the invalid-transport
	// error.
	if err != nil {
		assert.NotContains(t, err.Error(), "invalid transport type")
	}
}

// The http branch must be selected for --transport http and shut down cleanly
// when the context is cancelled. Port 0 picks an ephemeral port to avoid
// fixed-port flakiness.
func TestMCPCmd_HTTPSelected(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := newMCPCmd()
	cmd.SetArgs([]string{transportFlag, "http", "--address", "127.0.0.1:0"})

	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}

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

package models

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMCPParamName = "name"

func TestMCPArgumentsToParams(t *testing.T) {
	t.Run("nil parameters produce zero-value params", func(t *testing.T) {
		args := &MCPArguments{}

		got, err := args.ToParams[mcp.ListToolsParams]()
		require.NoError(t, err)
		assert.Equal(t, &mcp.ListToolsParams{}, got)
	})

	t.Run("parameters are decoded using json field names", func(t *testing.T) {
		args := &MCPArguments{Parameters: map[string]any{
			testMCPParamName: "echo",
			"arguments":      map[string]any{"message": "hello"},
		}}

		got, err := args.ToParams[mcp.CallToolParams]()
		require.NoError(t, err)
		assert.Equal(t, "echo", got.Name)
		assert.Equal(t, map[string]any{"message": "hello"}, got.Arguments)
	})

	t.Run("prompt arguments are decoded", func(t *testing.T) {
		args := &MCPArguments{Parameters: map[string]any{
			testMCPParamName: "greeting",
			"arguments":      map[string]any{"who": "world"},
		}}

		got, err := args.ToParams[mcp.GetPromptParams]()
		require.NoError(t, err)
		assert.Equal(t, "greeting", got.Name)
		assert.Equal(t, map[string]string{"who": "world"}, got.Arguments)
	})

	t.Run("unknown parameters are rejected", func(t *testing.T) {
		args := &MCPArguments{Parameters: map[string]any{
			testMCPParamName: "echo",
			"unknown":        true,
		}}

		_, err := args.ToParams[mcp.CallToolParams]()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("wrongly typed parameters are rejected", func(t *testing.T) {
		args := &MCPArguments{Parameters: map[string]any{"uri": 123}}

		_, err := args.ToParams[mcp.ReadResourceParams]()
		require.Error(t, err)
	})

	t.Run("non-object parameters are rejected", func(t *testing.T) {
		args := &MCPArguments{Parameters: "not-an-object"}

		_, err := args.ToParams[mcp.CallToolParams]()
		require.Error(t, err)
	})
}

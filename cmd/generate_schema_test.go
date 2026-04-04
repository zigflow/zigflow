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
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateSchemaCmd(t *testing.T) {
	t.Run("prints JSON schema to stdout", func(t *testing.T) {
		var buf bytes.Buffer

		cmd := newGenerateSchemaCmd()
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--format", "json"})

		err := cmd.Execute()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(buf.Bytes(), &result)
		assert.NoError(t, err, "output should be valid JSON")
		assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", result["$schema"])
	})

	t.Run("prints YAML schema to stdout", func(t *testing.T) {
		var buf bytes.Buffer

		cmd := newGenerateSchemaCmd()
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--format", "yaml"})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.NotEmpty(t, buf.Bytes())
		assert.Contains(t, buf.String(), "$schema:")
	})

	t.Run("defaults to JSON format", func(t *testing.T) {
		var buf bytes.Buffer

		cmd := newGenerateSchemaCmd()
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(buf.Bytes(), &result)
		assert.NoError(t, err, "default output should be valid JSON")
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		cmd := newGenerateSchemaCmd()
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"--format", "toml"})

		err := cmd.Execute()
		assert.Error(t, err)
	})
}

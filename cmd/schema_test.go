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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchemaCmd(t *testing.T) {
	tests := []struct {
		Name        string
		Args        []string
		ExpectError bool
	}{
		{
			Name: "JSON output",
			Args: []string{"--output", "json"},
		},
		{
			Name: "YAML output",
			Args: []string{"--output", "yaml"},
		},
		{
			Name:        "unsupported output format",
			Args:        []string{"--output", "toml"},
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cmd := newSchemaCmd()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(test.Args)

			err := cmd.Execute()

			if test.ExpectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

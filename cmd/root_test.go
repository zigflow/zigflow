/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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

func TestNewRootCmd_Subcommands(t *testing.T) {
	cmd := newRootCmd()

	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	assert.True(t, names["graph"])
	assert.True(t, names["run"])
	assert.True(t, names["version"])
	assert.True(t, names["validate"])
	assert.True(t, names["schema"])
	assert.True(t, names["generate-docs"])
}

func TestNewRootCmd_Flags(t *testing.T) {
	cmd := newRootCmd()

	assert.NotNil(t, cmd.PersistentFlags().Lookup("disable-telemetry"))
	assert.NotNil(t, cmd.PersistentFlags().Lookup("log-level"))
}

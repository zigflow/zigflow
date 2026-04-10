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

package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/telemetry"
)

func TestNewRunCmd_Flags(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	assert.NotNil(t, cmd.Flags().Lookup("file"))
	assert.NotNil(t, cmd.Flags().Lookup("validate"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-address"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-namespace"))
	assert.NotNil(t, cmd.Flags().Lookup("codec-endpoint"))
	assert.NotNil(t, cmd.Flags().Lookup("codec-headers"))
	assert.NotNil(t, cmd.Flags().Lookup("convert-data"))
	assert.NotNil(t, cmd.Flags().Lookup("converter-key-path"))
	assert.NotNil(t, cmd.Flags().Lookup("cloudevents-config"))
	assert.NotNil(t, cmd.Flags().Lookup("env-prefix"))
	assert.NotNil(t, cmd.Flags().Lookup("health-listen-address"))
	assert.NotNil(t, cmd.Flags().Lookup("metrics-listen-address"))
	assert.NotNil(t, cmd.Flags().Lookup("dir"))
	assert.NotNil(t, cmd.Flags().Lookup("glob"))
}

// ---- --watch flags ----

func TestNewRunCmd_WatchFlags(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	watchFlag := cmd.Flags().Lookup("watch")
	require.NotNil(t, watchFlag)
	assert.Equal(t, "false", watchFlag.DefValue)

	debounceFlag := cmd.Flags().Lookup("watch-debounce")
	require.NotNil(t, debounceFlag)
	assert.Equal(t, "300ms", debounceFlag.DefValue)
}

func TestNewRunCmd_WatchFlagsBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("watch", "true"))
	require.NoError(t, cmd.Flags().Set("watch-debounce", "500ms"))

	watchFlag := cmd.Flags().Lookup("watch")
	assert.Equal(t, "true", watchFlag.Value.String())

	debounceFlag := cmd.Flags().Lookup("watch-debounce")
	assert.Equal(t, "500ms", debounceFlag.Value.String())
}

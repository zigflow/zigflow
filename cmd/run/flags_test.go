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
	assert.NotNil(t, cmd.Flags().Lookup("temporal-server-name"))
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
	assert.NotNil(t, cmd.Flags().Lookup("max-concurrent-activity-execution-size"))
	assert.NotNil(t, cmd.Flags().Lookup("max-concurrent-workflow-task-execution-size"))
	assert.NotNil(t, cmd.Flags().Lookup("task-queue-activities-per-second"))
}

// ---- --temporal-server-name flag ----

func TestNewRunCmd_TemporalServerNameFlag(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("temporal-server-name")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.DefValue)
}

func TestNewRunCmd_TemporalServerNameFlagBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("temporal-server-name", "my-namespace.tmprl.cloud"))

	flag := cmd.Flags().Lookup("temporal-server-name")
	assert.Equal(t, "my-namespace.tmprl.cloud", flag.Value.String())
}

func TestNewRunCmd_TemporalServerNameFlagDefaultIsEmpty(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("temporal-server-name")
	require.NotNil(t, flag)
	// When omitted the flag is empty so existing connection behaviour is unchanged.
	assert.Equal(t, "", flag.Value.String())
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

// ---- worker tuning flags ----

func TestNewRunCmd_WorkerTuningFlags(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	actFlag := cmd.Flags().Lookup("max-concurrent-activity-execution-size")
	require.NotNil(t, actFlag)
	assert.Equal(t, "0", actFlag.DefValue)

	wfFlag := cmd.Flags().Lookup("max-concurrent-workflow-task-execution-size")
	require.NotNil(t, wfFlag)
	assert.Equal(t, "0", wfFlag.DefValue)

	tqFlag := cmd.Flags().Lookup("task-queue-activities-per-second")
	require.NotNil(t, tqFlag)
	assert.Equal(t, "0", tqFlag.DefValue)
}

func TestNewRunCmd_WorkerTuningFlagsBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("max-concurrent-activity-execution-size", "10"))
	require.NoError(t, cmd.Flags().Set("max-concurrent-workflow-task-execution-size", "5"))
	require.NoError(t, cmd.Flags().Set("task-queue-activities-per-second", "2.5"))

	assert.Equal(t, "10", cmd.Flags().Lookup("max-concurrent-activity-execution-size").Value.String())
	assert.Equal(t, "5", cmd.Flags().Lookup("max-concurrent-workflow-task-execution-size").Value.String())
	assert.Equal(t, "2.5", cmd.Flags().Lookup("task-queue-activities-per-second").Value.String())
}

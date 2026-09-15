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

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// runCmd returns the run subcommand of a freshly built root command. The root
// command is what wires viper to the environment, so flag defaults resolved
// from envvars are only observable through it.
func runCmd(t *testing.T) *cobra.Command {
	t.Helper()

	for _, sub := range newRootCmd().Commands() {
		if sub.Name() == "run" {
			return sub
		}
	}

	t.Fatal("run subcommand not found")
	return nil
}

// The Redis external storage settings are configured by envvar in the example
// compose stack and in container deployments generally, so the flags must pick
// their defaults up from the environment.
func TestNewRootCmd_RunRedisExternalStorageEnvvars(t *testing.T) {
	envvars := map[string]string{
		"EXTERNAL_STORAGE_REDIS_DRIVER_NAME":              "my-redis-driver",
		"EXTERNAL_STORAGE_REDIS_KEY_PREFIX":               "tenant-a:payloads",
		"EXTERNAL_STORAGE_REDIS_ADDRESS":                  "redis:6379",
		"EXTERNAL_STORAGE_REDIS_USERNAME":                 testRedisUsername,
		"EXTERNAL_STORAGE_REDIS_DATABASE":                 "3",
		"EXTERNAL_STORAGE_REDIS_TTL":                      "1h",
		"EXTERNAL_STORAGE_REDIS_TLS_ENABLED":              testTrue,
		"EXTERNAL_STORAGE_REDIS_TLS_CA":                   "/tls/ca.pem",
		"EXTERNAL_STORAGE_REDIS_TLS_CERT":                 "/tls/cert.pem",
		"EXTERNAL_STORAGE_REDIS_TLS_KEY":                  "/tls/key.pem",
		"EXTERNAL_STORAGE_REDIS_TLS_SERVER_NAME":          "redis.example.com",
		"EXTERNAL_STORAGE_REDIS_TLS_INSECURE_SKIP_VERIFY": testTrue,
	}
	for name, value := range envvars {
		t.Setenv(name, value)
	}

	want := map[string]string{
		"external-storage-redis-driver-name":              "my-redis-driver",
		"external-storage-redis-key-prefix":               "tenant-a:payloads",
		"external-storage-redis-address":                  "redis:6379",
		"external-storage-redis-username":                 testRedisUsername,
		"external-storage-redis-database":                 "3",
		"external-storage-redis-ttl":                      "1h0m0s",
		"external-storage-redis-tls-enabled":              testTrue,
		"external-storage-redis-tls-ca":                   "/tls/ca.pem",
		"external-storage-redis-tls-cert":                 "/tls/cert.pem",
		"external-storage-redis-tls-key":                  "/tls/key.pem",
		"external-storage-redis-tls-server-name":          "redis.example.com",
		"external-storage-redis-tls-insecure-skip-verify": testTrue,
	}

	cmd := runCmd(t)
	for name, value := range want {
		t.Run(name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(name)
			require.NotNil(t, flag)
			assert.Equal(t, value, flag.Value.String())
		})
	}
}

// The Redis password is read from the environment like the other settings, but
// it must not be echoed back in the command's help output.
func TestNewRootCmd_RunRedisPasswordEnvvarIsMasked(t *testing.T) {
	t.Setenv("EXTERNAL_STORAGE_REDIS_PASSWORD", "sup3rs3cret")

	cmd := runCmd(t)

	flag := cmd.Flags().Lookup("external-storage-redis-password")
	require.NotNil(t, flag)
	assert.Equal(t, "sup3rs3cret", flag.Value.String(), "the password must still reach the Redis client")
	assert.Equal(t, "***", flag.DefValue)
	assert.NotContains(t, cmd.UsageString(), "sup3rs3cret")
}

func TestNewRootCmd_RunRedisPasswordIsNotMaskedWhenUnset(t *testing.T) {
	t.Setenv("EXTERNAL_STORAGE_REDIS_PASSWORD", "")

	flag := runCmd(t).Flags().Lookup("external-storage-redis-password")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.DefValue)
}

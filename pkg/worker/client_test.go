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

package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	"go.temporal.io/sdk/client"
)

func TestClientOptionsRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  ClientConfig
		message string
	}{
		{name: "codec", config: ClientConfig{ConvertData: "invalid"}, message: "invalid codec type"},
		{name: "storage", config: ClientConfig{ExternalStorage: "invalid"}, message: "invalid external storage type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options, err := ClientOptions(t.Context(), nil, &tc.config)
			require.ErrorContains(t, err, tc.message)
			assert.Nil(t, options)
		})
	}
}

func TestClientOptionsDoesNotMutateConfiguration(t *testing.T) {
	connection := &temporal.TemporalOpts{
		Address: "localhost:7233", Namespace: "sdk-test",
		HealthListenAddress:  "invalid-listen-address",
		MetricsListenAddress: "invalid-listen-address",
	}
	original := *connection
	cfg := &ClientConfig{}
	// Building and applying options repeatedly must not register listeners or
	// process-global metrics, or alter a configuration reused by SDK callers.
	for range 2 {
		options, err := ClientOptions(t.Context(), connection, cfg)
		require.NoError(t, err)
		var opts client.Options
		for _, option := range options {
			require.NoError(t, option(&opts))
		}
		assert.Equal(t, connection.Address, opts.HostPort)
		assert.Equal(t, connection.Namespace, opts.Namespace)
		assert.Nil(t, opts.MetricsHandler)
		assert.Len(t, opts.ContextPropagators, 1)
	}
	assert.Equal(t, original, *connection)
	assert.Equal(t, &ClientConfig{}, cfg)
}

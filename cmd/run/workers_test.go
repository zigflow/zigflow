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

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/codec"
	"go.temporal.io/sdk/client"
)

// applyOptions applies a slice of temporal.Options to a fresh client.Options
// and returns the result. Each option is applied inside a deferred recover so
// that options which register global HTTP handlers (e.g. Prometheus) do not
// panic when called more than once across tests. TLS options are pure
// in-memory and always succeed.
func applyOptions(options []temporal.Options) client.Options {
	opts := &client.Options{}
	for _, o := range options {
		func() {
			defer func() { _ = recover() }()
			_ = o(opts)
		}()
	}
	return *opts
}

// stubTemporalConnection replaces newTemporalConnection with a test double that
// captures the options it receives. It returns a restore function that must be
// deferred by the caller.
func stubTemporalConnection(captured *client.Options, called *bool) func() {
	original := newTemporalConnection
	newTemporalConnection = func(options ...temporal.Options) (client.Client, error) {
		*called = true
		*captured = applyOptions(options)
		return nil, nil
	}
	return func() { newTemporalConnection = original }
}

// ---- initTemporalClient: TLS server name propagation ----

func TestInitTemporalClient_ServerNamePropagation(t *testing.T) {
	tests := []struct {
		name           string
		tlsEnabled     bool
		serverName     string
		expectTLSBlock bool
		expectSNI      string
	}{
		{
			name:           "server name is propagated when TLS is enabled",
			tlsEnabled:     true,
			serverName:     "your-namespace.tmprl.cloud",
			expectTLSBlock: true,
			expectSNI:      "your-namespace.tmprl.cloud",
		},
		{
			name:           "empty server name leaves SNI unset when TLS is enabled",
			tlsEnabled:     true,
			serverName:     "",
			expectTLSBlock: true,
			expectSNI:      "",
		},
		{
			name:           "no TLS block when TLS is disabled, even with server name set",
			tlsEnabled:     false,
			serverName:     "your-namespace.tmprl.cloud",
			expectTLSBlock: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var captured client.Options
			called := false
			defer stubTemporalConnection(&captured, &called)()

			opts := &runOptions{
				TemporalTLSEnabled: test.tlsEnabled,
				TemporalServerName: test.serverName,
			}

			_, err := initTemporalClient(opts)
			require.NoError(t, err)
			require.True(t, called, "newTemporalConnection was not invoked")

			if test.expectTLSBlock {
				require.NotNil(t, captured.ConnectionOptions.TLS, "expected TLS config to be set")
				assert.Equal(t, test.expectSNI, captured.ConnectionOptions.TLS.ServerName)
			} else {
				assert.Nil(t, captured.ConnectionOptions.TLS, "expected no TLS config")
			}
		})
	}
}

func TestBuildWorkersByTaskQueue(t *testing.T) {
	tests := []struct {
		name          string
		maxWorkflowTx int
		wantErr       bool
		errContains   string
	}{
		{
			name:          "max concurrent workflow task size of 1 is rejected",
			maxWorkflowTx: 1,
			wantErr:       true,
			errContains:   "cannot be set to 1",
		},
		{
			name:          "zero (disabled) is accepted",
			maxWorkflowTx: 0,
		},
		{
			name:          "value greater than 1 is accepted",
			maxWorkflowTx: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := &runOptions{
				MaxConcurrentWorkflowTaskExecutionSize: tc.maxWorkflowTx,
			}
			// nil client and empty registrations: validation fires before the
			// client is used, and the registration loop is skipped when empty.
			workers, err := buildWorkersByTaskQueue(nil, []*workflowRegistration{}, nil, opts)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workers)
				if tc.errContains != "" {
					assert.ErrorContains(t, err, tc.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workers)
			}
		})
	}
}

func TestBuildDataConverter(t *testing.T) {
	tests := []struct {
		Name         string
		ConvertData  string
		Endpoint     string
		KeyPath      string
		CodecHeaders map[string]string
		ExpectNil    bool
		ExpectError  bool
	}{
		{
			Name:        "disabled returns nil without reading key file",
			ConvertData: "",
			KeyPath:     "",
			ExpectNil:   true,
		},
		{
			Name:        "aes with missing key file returns error",
			ConvertData: "aes",
			KeyPath:     "/nonexistent/path/keys.yaml",
			ExpectNil:   true,
			ExpectError: true,
		},
		{
			Name:        "remote returns converter without error",
			ConvertData: "remote",
			Endpoint:    "http://localhost:8080",
			ExpectNil:   false,
		},
		{
			Name:         "remote with headers returns converter without error",
			ConvertData:  "remote",
			Endpoint:     "http://localhost:8080",
			CodecHeaders: map[string]string{"Authorization": "Bearer token"},
			ExpectNil:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			codecType, _ := codec.ParseCodecType(test.ConvertData)
			dc, err := codec.NewDataConverter(codecType, test.Endpoint, test.KeyPath, test.CodecHeaders)

			if test.ExpectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.ExpectNil {
				assert.Nil(t, dc)
			} else {
				assert.NotNil(t, dc)
			}
		})
	}
}

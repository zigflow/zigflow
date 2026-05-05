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
	sdkworker "go.temporal.io/sdk/worker"
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
			serverName:     testTemporalServerName,
			expectTLSBlock: true,
			expectSNI:      testTemporalServerName,
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
			serverName:     testTemporalServerName,
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

// capturedWorker records the arguments passed to newWorker by a single call.
type capturedWorker struct {
	taskQueue string
	opts      sdkworker.Options
}

// stubNewWorker replaces newWorker with a test double that records every call
// into captures. It creates the real worker using a lazy client (which satisfies
// the SDK's non-nil client check without opening a real connection) so that
// subsequent RegisterActivity and RegisterWorkflow calls succeed. It returns a
// restore function that must be deferred by the caller.
func stubNewWorker(captures *[]capturedWorker) func() {
	orig := newWorker
	newWorker = func(_ client.Client, taskQueue string, opts sdkworker.Options) sdkworker.Worker {
		*captures = append(*captures, capturedWorker{taskQueue: taskQueue, opts: opts})
		lazyClient, _ := client.NewLazyClient(client.Options{})
		return orig(lazyClient, taskQueue, opts)
	}
	return func() { newWorker = orig }
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

// ---- versioning identity errors ----

func TestBuildWorkersByTaskQueue_VersioningMissingIdentity(t *testing.T) {
	// A single minimal registration is enough to enter the per-worker creation
	// path; the error is returned before worker.New or zigflow.NewWorkflow are
	// called, so the definition content does not matter here.
	reg := &workflowRegistration{
		TaskQueue:    "test-queue",
		WorkflowType: "test-workflow",
	}

	tests := []struct {
		name        string
		opts        *runOptions
		errContains string
	}{
		{
			name: "missing build ID returns error",
			opts: &runOptions{
				EnableVersioning: true,
				DeploymentName:   "my-deploy",
			},
			errContains: "temporal-worker-build-id required",
		},
		{
			name: "missing deployment name returns error",
			opts: &runOptions{
				EnableVersioning:  true,
				DeploymentBuildID: "my-build-id",
			},
			errContains: "temporal-deployment-name required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workers, err := buildWorkersByTaskQueue(nil, []*workflowRegistration{reg}, nil, tc.opts)
			assert.Error(t, err)
			assert.Nil(t, workers)
			assert.ErrorContains(t, err, tc.errContains)
		})
	}
}

// ---- DeploymentOptions population ----

func TestBuildWorkersByTaskQueue_VersioningDeploymentOptions(t *testing.T) {
	dir := t.TempDir()
	file := writeTempWorkflow(t, dir, "test-queue", "test-workflow")
	regs, err := loadWorkflows([]string{file}, "", newTestValidator(t), false)
	require.NoError(t, err)

	var captured []capturedWorker
	defer stubNewWorker(&captured)()

	// defaultVersioningBehaviour mirrors what PreRunE sets: use the package
	// map so the test does not import go.temporal.io/sdk/workflow directly.
	opts := &runOptions{
		EnableVersioning:           true,
		DeploymentBuildID:          "test-build-id",
		DeploymentName:             "test-deploy",
		defaultVersioningBehaviour: versioningBehaviours[versioningBehaviourAutoUpgrade],
	}
	ws, err := buildWorkersByTaskQueue(nil, regs, nil, opts)
	require.NoError(t, err)
	require.NotNil(t, ws)
	require.Len(t, captured, 1, "expected exactly one worker to be created")

	c := captured[0]
	assert.Equal(t, "test-queue", c.taskQueue)
	assert.True(t, c.opts.DeploymentOptions.UseVersioning)
	assert.Equal(t, "test-build-id", c.opts.DeploymentOptions.Version.BuildID)
	assert.Equal(t, "test-deploy", c.opts.DeploymentOptions.Version.DeploymentName)
	assert.Equal(t, versioningBehaviours[versioningBehaviourAutoUpgrade], c.opts.DeploymentOptions.DefaultVersioningBehavior)
}

func TestBuildWorkersByTaskQueue_VersioningDisabledNoDeploymentOptions(t *testing.T) {
	var captured []capturedWorker
	defer stubNewWorker(&captured)()

	dir := t.TempDir()
	file := writeTempWorkflow(t, dir, "test-queue", "test-workflow")
	regs, err := loadWorkflows([]string{file}, "", newTestValidator(t), false)
	require.NoError(t, err)

	opts := &runOptions{EnableVersioning: false}
	ws, err := buildWorkersByTaskQueue(nil, regs, nil, opts)
	require.NoError(t, err)
	require.NotNil(t, ws)
	require.Len(t, captured, 1)

	assert.False(t, captured[0].opts.DeploymentOptions.UseVersioning)
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

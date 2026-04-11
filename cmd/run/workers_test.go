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
	"github.com/zigflow/zigflow/pkg/codec"
)

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

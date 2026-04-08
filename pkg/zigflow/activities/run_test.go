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

package activities

import (
	"path/filepath"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
)

func TestCallShellActivityInterpolatesEnvironmentExpressions(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	testEnv := s.NewTestActivityEnvironment()

	run := &Run{}
	testEnv.RegisterActivity(run.CallShellActivity)

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Shell: &model.Shell{
				Command: "sh",
				Arguments: &model.RunArguments{
					Value: []string{"-c", `printf "%s" "$OPENAI_API_KEY"`},
				},
				Environment: map[string]string{
					"OPENAI_API_KEY": "${ $env.OPENAI_API_KEY }",
				},
			},
		},
	}

	state := utils.NewState()
	state.Env["OPENAI_API_KEY"] = "secret-key"

	val, err := testEnv.ExecuteActivity(run.CallShellActivity, task, nil, state)
	require.NoError(t, err)
	require.True(t, val.HasValue())

	var output string
	require.NoError(t, val.Get(&output))
	assert.Equal(t, "secret-key", output)
}

func TestInterpolateContainerConfigInterpolatesEnvironmentAndVolumes(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image:   "${ $env.IMAGE }",
				Command: "",
				Environment: map[string]string{
					"SERVICE_VALUE": "${ $env.SERVICE_VALUE }",
				},
				Volumes: map[string]interface{}{
					"./tmp": "${ $env.CONTAINER_MOUNT }",
				},
				Arguments: []string{"${ $env.ARG_1 }"},
			},
		},
	}

	state := utils.NewState()
	state.Env["IMAGE"] = "alpine:3.20"
	state.Env["SERVICE_VALUE"] = "test-value"
	state.Env["CONTAINER_MOUNT"] = "/workspace"
	state.Env["ARG_1"] = "echo"

	containerData, err := interpolateContainerConfig(task, state)
	require.NoError(t, err)
	require.NotNil(t, containerData)

	envCmd := appendDockerEnvFlags([]string{"docker", "run"}, containerData["environment"])
	assert.Contains(t, envCmd, "--env=SERVICE_VALUE=test-value")

	volCmd, err := appendDockerVolumeFlags([]string{"docker", "run"}, containerData["volumes"])
	require.NoError(t, err)
	expectedLocal, err := filepath.Abs("./tmp")
	require.NoError(t, err)
	assert.Contains(t, volCmd, "--volume="+expectedLocal+":/workspace")
}

func TestDockerEntrypointFromInterpolated(t *testing.T) {
	tests := []struct {
		name          string
		value         any
		expected      string
		hasEntrypoint bool
		wantErr       bool
	}{
		{
			name:          "nil command is omitted",
			value:         nil,
			expected:      "",
			hasEntrypoint: false,
			wantErr:       false,
		},
		{
			name:          "empty command is omitted",
			value:         "",
			expected:      "",
			hasEntrypoint: false,
			wantErr:       false,
		},
		{
			name:          "non-empty command is used",
			value:         "/bin/sh",
			expected:      "/bin/sh",
			hasEntrypoint: true,
			wantErr:       false,
		},
		{
			name:          "non-string command fails",
			value:         123,
			expected:      "",
			hasEntrypoint: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entrypoint, hasEntrypoint, err := dockerEntrypointFromInterpolated(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, entrypoint)
			assert.Equal(t, tt.hasEntrypoint, hasEntrypoint)
		})
	}
}

func TestDockerImageFromInterpolated(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
		wantErr  bool
	}{
		{
			name:    "nil image fails",
			value:   nil,
			wantErr: true,
		},
		{
			name:    "empty image fails",
			value:   "",
			wantErr: true,
		},
		{
			name:    "whitespace image fails",
			value:   "   ",
			wantErr: true,
		},
		{
			name:     "valid image passes",
			value:    "alpine:3.20",
			expected: "alpine:3.20",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := dockerImageFromInterpolated(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, image)
		})
	}
}

func TestDockerArgsFromInterpolated(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected []string
		wantErr  bool
	}{
		{
			name:     "nil arguments returns empty",
			value:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "string slice passes through",
			value:    []string{"echo", "hello"},
			expected: []string{"echo", "hello"},
			wantErr:  false,
		},
		{
			name:     "any slice is stringified",
			value:    []any{"echo", 7, true},
			expected: []string{"echo", "7", "true"},
			wantErr:  false,
		},
		{
			name:    "nil argument fails fast",
			value:   []any{"echo", nil},
			wantErr: true,
		},
		{
			name:    "non-array arguments fail",
			value:   "echo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := dockerArgsFromInterpolated(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, args)
		})
	}
}

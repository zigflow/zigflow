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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/telemetry"
)

// ---- PreRunE: versioning validation ----

func TestPreRunE_VersioningValidation(t *testing.T) {
	tests := []struct {
		name             string
		enableVersioning string
		versioningType   string
		wantErr          bool
		errContains      string
	}{
		{
			name:             "invalid versioning type returns error when versioning enabled",
			enableVersioning: testEnableVersioningTrue,
			versioningType:   "not-a-valid-type",
			wantErr:          true,
			errContains:      "invalid default versioning behaviour type",
		},
		{
			name:             "valid pinned type succeeds",
			enableVersioning: testEnableVersioningTrue,
			versioningType:   "pinned",
		},
		{
			name:             "valid autoupgrade type succeeds",
			enableVersioning: testEnableVersioningTrue,
			versioningType:   "autoupgrade",
		},
		{
			name:             "invalid type is ignored when versioning is disabled",
			enableVersioning: "false",
			versioningType:   "not-a-valid-type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := New(func() *telemetry.Telemetry { return nil })
			require.NoError(t, cmd.Flags().Set("enable-versioning", tc.enableVersioning))
			require.NoError(t, cmd.Flags().Set("default-versioning-type", tc.versioningType))

			err := cmd.PreRunE(cmd, []string{})
			if tc.wantErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPanicMessage(t *testing.T) {
	tests := []struct {
		Name     string
		Input    any
		Expected string
	}{
		{
			Name:     "error value",
			Input:    errors.New("something went wrong"),
			Expected: "something went wrong",
		},
		{
			Name:     "string value",
			Input:    "a plain string",
			Expected: "a plain string",
		},
		{
			Name:     "other value",
			Input:    42,
			Expected: fmt.Sprintf("%+v", 42),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, panicMessage(test.Input))
		})
	}
}

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

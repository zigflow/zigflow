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

package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCodecType(t *testing.T) {
	tests := []struct {
		input   string
		want    CodecType
		wantErr bool
	}{
		{"", CodecNone, false},
		{"aes", CodecAES, false},
		{"remote", CodecRemote, false},
		{"invalid", "", true},
		{"AES", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCodecType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewDataConverter(t *testing.T) {
	t.Run("CodecNone returns nil", func(t *testing.T) {
		dc, err := NewDataConverter(CodecNone, "", "", nil)
		require.NoError(t, err)
		assert.Nil(t, dc)
	})

	t.Run("CodecAES returns error for missing key file", func(t *testing.T) {
		_, err := NewDataConverter(CodecAES, "", "nonexistent-keys.yaml", nil)
		assert.Error(t, err)
	})

	t.Run("CodecRemote returns a DataConverter", func(t *testing.T) {
		dc, err := NewDataConverter(CodecRemote, "http://localhost:8081", "", nil)
		require.NoError(t, err)
		assert.NotNil(t, dc)
	})

	t.Run("unknown codec type returns error", func(t *testing.T) {
		_, err := NewDataConverter(CodecType("bogus"), "", "", nil)
		assert.Error(t, err)
	})
}

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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAESConverter(t *testing.T) {
	t.Run("missing key file returns error", func(t *testing.T) {
		_, err := NewAESConverter("nonexistent-keys.yaml")
		assert.Error(t, err)
	})

	t.Run("valid key file returns a DataConverter", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "keys-*.yaml")
		require.NoError(t, err)
		_, err = f.WriteString("- id: key0\n  key: passphrasewhichneedstobe32bytes!\n")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		dc, err := NewAESConverter(f.Name())
		require.NoError(t, err)
		assert.NotNil(t, dc)
	})

	t.Run("empty key file returns error", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "keys-*.yaml")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		_, err = NewAESConverter(f.Name())
		assert.Error(t, err)
	})
}

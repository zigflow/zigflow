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
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewGenerateDocsCmd(t *testing.T) {
	t.Run("generates markdown files in explicit output directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "generatedocs_test")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.RemoveAll(tmpDir))
		}()

		root := &cobra.Command{Use: "test", Short: "Test root command"}
		cmd := newGenerateDocsCmd(root)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{tmpDir})

		err = cmd.Execute()
		assert.NoError(t, err)

		entries, err := os.ReadDir(tmpDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, entries)
		for _, e := range entries {
			assert.True(t, strings.HasSuffix(e.Name(), ".md"),
				"expected only .md files, got: %s", e.Name())
		}
	})

	t.Run("uses current working directory when no output dir is given", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "generatedocs_wd_test")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.RemoveAll(tmpDir))
		}()

		t.Chdir(tmpDir)

		root := &cobra.Command{Use: "test", Short: "Test root command"}
		cmd := newGenerateDocsCmd(root)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{})

		err = cmd.Execute()
		assert.NoError(t, err)

		expectedDir := tmpDir + "/docs/docs/cli"
		entries, err := os.ReadDir(expectedDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, entries)
	})
}

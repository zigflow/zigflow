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
	"github.com/spf13/cobra"
)

// newGenerateCmd returns a hidden parent command that groups maintainer
// code-generation utilities under `zigflow generate`. It is hidden from end-user
// help output because its subcommands are intended for contributors and CI only.
func newGenerateCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Run maintainer code generators",
		Long: `Run code generators used during development and CI.

These commands are intended for Zigflow maintainers. They regenerate
artefacts that are checked in to the repository, such as CLI documentation
and the JSON schema file.`,
		Hidden: true,
	}

	cmd.AddCommand(
		newGenerateDocsCmd(root),
		newGenerateSchemaCmd(),
	)

	return cmd
}

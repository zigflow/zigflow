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
	"errors"
	"fmt"
	"os"
	"path"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/spf13/cobra"
	"github.com/zigflow/zigflow/pkg/llmsdoc"
)

const (
	llmsTemplate = "llms.txt.in"
	llmsOutput   = "static/llms.txt"
)

func newGenerateLLMSCmd() *cobra.Command {
	var check bool

	cmd := &cobra.Command{
		Use:   "generate-llms",
		Short: "Generate docs/static/llms.txt from authoritative sources",
		Long: `Renders the generated reference regions of docs/static/llms.txt from the
template at docs/llms.txt.in, using the schema, bundled examples,
validation error registry and MCP tool registration as the source of truth.

Run with --check to verify the committed file is up to date without writing it.`,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, err := os.Getwd()
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error getting current working directory",
				}
			}

			if len(args) > 0 {
				pwd = path.Join(pwd, args[0])
			}

			llmsTemplatePath := path.Join(pwd, llmsTemplate)
			llmsOutputPath := path.Join(pwd, llmsOutput)

			template, err := os.ReadFile(llmsTemplatePath)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   fmt.Sprintf("Error reading template %s", llmsTemplatePath),
				}
			}

			rendered, err := llmsdoc.Render(string(template))
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error rendering llms.txt",
				}
			}

			if check {
				current, err := os.ReadFile(llmsOutputPath)
				if err != nil {
					return gh.FatalError{
						Cause: err,
						Msg:   fmt.Sprintf("Error reading %s", llmsOutputPath),
					}
				}

				if string(current) != rendered {
					return gh.FatalError{
						Cause: errors.New("generated content differs from committed file"),
						Msg:   fmt.Sprintf("%s is out of date; run `zigflow generate-llms`", llmsOutputPath),
					}
				}

				return nil
			}

			if err := os.WriteFile(llmsOutputPath, []byte(rendered), 0o644); err != nil { //nolint:gosec // world-readable doc artefact
				return gh.FatalError{
					Cause: err,
					Msg:   fmt.Sprintf("Error writing %s", llmsOutputPath),
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Verify the committed llms.txt is up to date and exit non-zero if not")

	return cmd
}

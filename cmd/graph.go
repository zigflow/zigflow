/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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
	"fmt"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/zigflow/pkg/graph"
	"github.com/mrsimonemms/zigflow/pkg/zigflow"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	var opts struct {
		Output string
	}

	cmd := &cobra.Command{
		Use:   "graph <workflow-file>",
		Short: "Generate a visual graph of a Zigflow workflow",
		Long: `Generate a visual graph of a Zigflow workflow definition.

This command loads the provided workflow file and renders a structural diagram
that shows tasks, control flow, branching, loops, error handling, and — for
files that define multiple Temporal workflows — each workflow as a separate
labelled section.

The diagram reflects the same workflow topology that the Zigflow runtime uses:
tasks are shown in execution order, switch branches indicate conditional routing,
fork tasks show parallel branches, for tasks show loop bodies, and try/catch
tasks show the error-handling path.

Use the --output flag to select the renderer. Currently supported:

  mermaid   Mermaid flowchart (https://mermaid.ai)

The output is written to stdout and can be piped directly into tools or saved
to a file for use in documentation, pull-request descriptions, or any Mermaid-
compatible renderer.

Arguments:
  workflow-file   Path to the Zigflow workflow file to graph`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			workflowDefinition, err := zigflow.LoadFromFile(filePath)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Unable to load workflow file",
				}
			}

			gen, err := graph.New(graph.Format(opts.Output))
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Unsupported output format",
				}
			}

			output, err := gen.Generate(workflowDefinition)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error generating graph",
				}
			}

			fmt.Print(output)
			return nil
		},
	}

	cmd.Flags().StringVarP(
		&opts.Output, "output", "o",
		string(graph.FormatMermaid), "Output format (mermaid)",
	)

	return cmd
}

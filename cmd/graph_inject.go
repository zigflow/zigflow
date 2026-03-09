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
	"fmt"
	"os"
	"path/filepath"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/spf13/cobra"
	"github.com/zigflow/zigflow/pkg/graph"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

func newGraphInjectCmd() *cobra.Command {
	var opts struct {
		Output       string
		WorkflowFile string
		StartMarker  string
		EndMarker    string
	}

	cmd := &cobra.Command{
		Use:   "inject [flags] <target-file> [target-file...]",
		Short: "Inject a workflow graph into a file between marker comments",
		Long: `Inject a rendered workflow graph into one or more target files.

For each target file the command finds a pair of marker comments, replaces
everything between them with a freshly generated graph, then writes the result
back. Running inject is idempotent: if the graph is already current the file
content does not change.

Auto-detect form (no --workflow flag):

    zigflow graph inject <target-file> [target-file...]

The workflow file path is read from the start marker embedded in each target
file. Embed a path like this:

    <!-- ZIGFLOW_GRAPH_START ./workflow.yaml -->
    <!-- ZIGFLOW_GRAPH_END -->

When no path is embedded the default "` + graph.DefaultWorkflowFile + `" is used, resolved
relative to the target file's directory. If a target file contains no
ZIGFLOW_GRAPH_START marker it is skipped silently, so the command is safe
to use with pass_filenames: true in pre-commit hooks.

Explicit form (with --workflow flag):

    zigflow graph inject --workflow <workflow-file> <target-file> [target-file...]

All targets use the supplied workflow file and the --start-marker value.

Default markers:

    ` + graph.DefaultStartMarkerPrefix + ` -->
    ` + graph.DefaultEndMarker,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, targetFile := range args {
				var err error
				if opts.WorkflowFile != "" {
					err = execGraphInject(opts.WorkflowFile, targetFile, opts.Output, opts.StartMarker, opts.EndMarker)
				} else {
					err = execGraphInjectAuto(targetFile, opts.Output, opts.EndMarker)
				}
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(
		&opts.Output, "output", "o",
		string(graph.FormatMermaid), "Output format (mermaid)",
	)
	cmd.Flags().StringVarP(
		&opts.WorkflowFile, "workflow", "w",
		"", "Explicit workflow file (auto-detected from start marker if not set)",
	)
	cmd.Flags().StringVarP(
		&opts.StartMarker, "start-marker", "s",
		graph.DefaultStartMarkerPrefix+" -->", "Start marker comment (requires --workflow)",
	)
	cmd.Flags().StringVarP(
		&opts.EndMarker, "end-marker", "e",
		graph.DefaultEndMarker, "End marker comment",
	)

	return cmd
}

// execGraphInjectAuto handles the one-argument form: the workflow path is read
// from the embedded start marker inside targetFile.
func execGraphInjectAuto(targetFile, outputFormat, endMarker string) error {
	data, err := os.ReadFile(targetFile)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to read target file"}
	}

	workflowPath, startMarker, found := graph.ParseEmbeddedPath(string(data), graph.DefaultWorkflowFile)
	if !found {
		// No ZIGFLOW_GRAPH_START in this file — skip silently.
		return nil
	}

	// Resolve the workflow path relative to the target file's directory.
	workflowPath = filepath.Join(filepath.Dir(targetFile), workflowPath)

	return execGraphInject(workflowPath, targetFile, outputFormat, startMarker, endMarker)
}

func execGraphInject(workflowFile, targetFile, outputFormat, startMarker, endMarker string) error {
	// Resolve to an absolute path to prevent path traversal.
	absTarget, pathErr := filepath.Abs(targetFile)
	if pathErr != nil {
		return gh.FatalError{Cause: pathErr, Msg: "Unable to resolve target file path"}
	}

	wf, err := zigflow.LoadFromFile(workflowFile)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to load workflow file"}
	}

	gen, err := graph.New(graph.Format(outputFormat))
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unsupported output format"}
	}

	graphOutput, err := gen.Generate(wf)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Error generating graph"}
	}

	// Wrap in a fenced code block so it renders in Markdown.
	codeBlock := fmt.Sprintf("```%s\n%s```", outputFormat, graphOutput)

	info, err := os.Stat(absTarget)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to read target file"}
	}

	fileData, err := os.ReadFile(absTarget)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to read target file"}
	}

	result, err := graph.InjectGraph(string(fileData), startMarker, endMarker, codeBlock)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to inject graph"}
	}

	// absTarget is resolved via filepath.Abs from a user-supplied CLI argument; path traversal is
	// intentional and expected for a CLI tool. gosec G703 is a false positive here.
	if err := os.WriteFile(absTarget, []byte(result), info.Mode()); err != nil { //nolint:gosec
		return gh.FatalError{Cause: err, Msg: "Unable to write target file"}
	}

	return nil
}

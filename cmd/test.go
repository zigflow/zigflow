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
	"time"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	temporal "github.com/zigflow/helpers"
	runcmd "github.com/zigflow/zigflow/cmd/run"
	"github.com/zigflow/zigflow/pkg/testrunner"
)

func newTestCmd() *cobra.Command {
	var opts struct {
		Input      string
		InputJSON  string
		Timeout    time.Duration
		TemporalUI string
		temporal   *temporal.TemporalOpts
	}

	opts.temporal = &temporal.TemporalOpts{}

	cmd := &cobra.Command{
		Use:   "test <workflow-file>",
		Short: "Run a workflow once against Temporal and print the result",
		Long: `Validate a workflow file, start a short-lived in-process worker, execute the
workflow with JSON input, wait for completion, and print the result.

This command is intended for local development and CI scripts. It connects to
Temporal using the same flags as zigflow run. A dev Temporal server is usually
started with "temporal server start-dev".

All test runs use the fixed task queue "` + runcmd.TestTaskQueue + `" so they do not
compete with long-lived zigflow run workers on document.taskQueue from the file.

Only one zigflow test process at a time may poll that queue for a given Temporal
address and namespace; other invocations wait on a file lock until the run
finishes.

Arguments:
  workflow-file   Path to the Zigflow workflow file to test`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestCmd(cmd, args[0], opts.Input, opts.InputJSON, opts.Timeout, opts.TemporalUI, opts.temporal)
		},
	}

	viper.SetDefault("test_timeout", 10*time.Minute)
	cmd.Flags().StringVar(
		&opts.Input, "input",
		viper.GetString("test_input"), "Path to a JSON file used as workflow input",
	)
	cmd.Flags().StringVar(
		&opts.InputJSON, "input-json",
		viper.GetString("test_input_json"), "Inline JSON workflow input",
	)
	cmd.Flags().DurationVar(
		&opts.Timeout, "timeout",
		viper.GetDuration("test_timeout"), "Maximum time to wait for workflow completion",
	)
	viper.SetDefault("temporal_ui_address", "http://localhost:8233")
	cmd.Flags().StringVar(
		&opts.TemporalUI, "temporal-ui-address",
		viper.GetString("temporal_ui_address"), "Temporal Web UI base URL for inspect links on failure",
	)

	temporal.NewCobraOpts(cmd, opts.temporal)

	return cmd
}

func runTestCmd(
	cmd *cobra.Command,
	workflowFile, inputPath, inputJSON string,
	timeout time.Duration,
	temporalUI string,
	temporalOpts *temporal.TemporalOpts,
) error {
	if inputPath != "" && inputJSON != "" {
		return fmt.Errorf("use only one of --input or --input-json")
	}

	var (
		input any
		err   error
	)
	switch {
	case inputPath != "":
		input, err = testrunner.LoadInputFile(inputPath)
	case inputJSON != "":
		input, err = testrunner.ParseInputJSON(inputJSON)
	default:
		input = map[string]any{}
	}
	if err != nil {
		return err
	}

	result, err := testrunner.Run(cmd.Context(), testrunner.Config{
		WorkflowFile: workflowFile,
		Input:        input,
		Timeout:      timeout,
		Temporal:     temporalOpts,
		TemporalUI:   temporalUI,
		Telemetry:    app.Telemetry,
	})
	if err != nil {
		return err
	}

	if err := testrunner.FormatHuman(cmd.OutOrStdout(), result); err != nil {
		return err
	}

	if result.Status != testrunner.StatusCompleted {
		msg := fmt.Sprintf("workflow test failed: %s", result.Status)
		if result.Err != nil {
			msg = fmt.Sprintf("%s: %v", msg, result.Err)
		}
		return gh.FatalError{Msg: msg}
	}

	return nil
}

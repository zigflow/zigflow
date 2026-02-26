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
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCmd() *cobra.Command {
	viper.AutomaticEnv()

	var opts struct {
		LogLevel string
	}

	rootCmd := &cobra.Command{
		Use:     "zigflow",
		Version: Version,
		Short:   "A Temporal DSL for turning declarative YAML into production-ready workflows",
		Long: `Zigflow is a command-line tool for building and running Temporal workflows
defined in declarative YAML. It uses the CNCF Serverless Workflow specification
to let you describe durable business processes in a structured, human-readable
format, giving you the reliability and fault tolerance of Temporal without
writing boilerplate worker code.

With Zigflow, you can:
- Define workflow logic using simple YAML DSL that maps directly to Temporal
  concepts like activities, signals, queries and retries.

- Run workflows locally or in production, with workers and task queues
  automatically defined from your workflow files.

- Reuse workflow components and enforce consistent patterns across your Temporal
  estate, making it easier to share and maintain workflow logic across teams and
  projects.

The CLI includes commands for validating, executing, and generating helpers
for your workflows, making it an intuitive interface for both developers and
operators. Zigflow aims to reduce the cognitive load of writing boilerplate
Temporal code while preserving the full power and extensibility of the Temporal
platform.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level, err := zerolog.ParseLevel(opts.LogLevel)
			if err != nil {
				return err
			}
			zerolog.SetGlobalLevel(level)

			return nil
		},
	}

	viper.SetDefault("log_level", zerolog.InfoLevel.String())
	rootCmd.PersistentFlags().StringVarP(
		&opts.LogLevel, "log-level", "l",
		viper.GetString("log_level"), "Set log level",
	)

	rootCmd.AddCommand(
		newRunCmd(),
		newVersionCmd(),
		newValidateCmd(),
		newSchemaCmd(),
		newGraphCmd(),
		newGenerateDocsCmd(rootCmd),
	)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

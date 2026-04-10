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
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/zigflow/zigflow/pkg/codec"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/utils"
)

type runOptions struct {
	CloudEventsConfig                      string
	CodecEndpoint                          string
	CodecHeaders                           map[string]string
	ConvertData                            string
	ConvertKeyPath                         string
	EnvPrefix                              string
	DirectoryGlob                          string
	DirectoryPath                          string
	Files                                  []string
	GracefulShutdownTimeout                time.Duration
	HealthListenAddress                    string
	MaxConcurrentActivityExecutionSize     int
	MaxConcurrentWorkflowTaskExecutionSize int
	MetricsListenAddress                   string
	MetricsPrefix                          string
	TaskQueueActivitiesPerSecond           float64
	TemporalAddress                        string
	TemporalAPIKey                         string
	TemporalMTLSCertPath                   string
	TemporalMTLSKeyPath                    string
	TemporalNamespace                      string
	TemporalServerName                     string
	TemporalTLSEnabled                     bool
	Validate                               bool
	Watch                                  bool
	WatchDebounce                          time.Duration

	Telemetry *telemetry.Telemetry
}

func panicMessage(r any) string {
	switch v := r.(type) {
	case error:
		return v.Error()
	case string:
		return v
	default:
		return fmt.Sprintf("%+v", v)
	}
}

func runRunCmd(ctx context.Context, opts *runOptions) error {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal().
				Str("type", fmt.Sprintf("%T", r)).
				Str("panicMsg", panicMessage(r)).
				Msg("Recovered from panic")
		}
	}()

	registrations, err := prepareRegistrations(opts)
	if err != nil {
		return err
	}

	temporalClient, err := initTemporalClient(opts)
	if err != nil {
		return err
	}
	// Registered first, runs last (LIFO). The client is closed only after all
	// workers have been stopped by the defer below.
	defer func() {
		log.Trace().Msg("Closing Temporal connection")
		temporalClient.Close()
		log.Trace().Msg("Temporal connection closed")
	}()

	prefix := opts.EnvPrefix + "_"
	log.Debug().Str("prefix", prefix).Msg("Loading envvars to state")
	envvars := utils.LoadEnvvars(prefix)

	if err := runScheduleUpdates(ctx, temporalClient, registrations, envvars); err != nil {
		return err
	}

	startedWorkers, err := startInitialWorkers(ctx, temporalClient, registrations, envvars, opts)
	if err != nil {
		return err
	}

	if opts.Watch {
		watchFiles, err := discoverWorkflowFiles(opts)
		if err != nil {
			for _, w := range startedWorkers {
				w.Stop()
			}
			return err
		}

		log.Info().Strs("files", watchFiles).Msg("Watch mode enabled. Not recommended for production use.")

		// runWatchMode owns worker lifecycle; only telemetry needs cleanup here.
		defer func() {
			if opts.Telemetry != nil {
				opts.Telemetry.Shutdown()
			}
		}()

		return runWatchMode(ctx, watchFiles, temporalClient, opts, envvars, startedWorkers)
	}

	// Registered second, runs first (LIFO). Workers are stopped before the
	// Temporal client is closed. Telemetry.Shutdown is nil-safe.
	defer func() {
		log.Info().Int("count", len(startedWorkers)).Msg("Stopping workers")
		for _, w := range startedWorkers {
			w.Stop()
		}
		if opts.Telemetry != nil {
			opts.Telemetry.Shutdown()
		}
	}()

	// Block until SIGINT/SIGTERM or context cancellation.
	waitForShutdown(ctx)
	return nil
}

// New constructs the run cobra.Command. telemetryFn is called when the command
// executes; by that point PersistentPreRunE will have populated the telemetry
// instance, so it is safe to dereference.
func New(telemetryFn func() *telemetry.Telemetry) *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start Zigflow workflow workers",
		Long: `Start one or more Zigflow workflow workers that connect to Temporal and process
workflow executions defined in the provided workflow definitions.

Workflow definitions are loaded from the files specified with --file, from a
directory matched with --dir and --glob, or from both sources combined. All
definitions are validated before any worker is started.

Workflows that share a task queue (defined by document.taskQueue) are
registered on a single shared worker. Each distinct task queue gets its own
worker. Multiple definitions targeting the same task queue and sharing a
workflowType are rejected before startup.

The process blocks until interrupted and then stops all workers cleanly before
closing the Temporal connection.

Use this command to deploy and run your Zigflow workflows in any environment,
from local development to production.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := codec.ParseCodecType(opts.ConvertData); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Telemetry = telemetryFn()
			return runRunCmd(cmd.Context(), &opts)
		},
	}

	registerRunFlags(cmd, &opts)

	return cmd
}

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
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/matthewmueller/glob"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/codec"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"github.com/zigflow/zigflow/pkg/zigflow/tasks"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type runOptions struct {
	CloudEventsConfig       string
	CodecEndpoint           string
	CodecHeaders            map[string]string
	ConvertData             string
	ConvertKeyPath          string
	EnvPrefix               string
	DirectoryGlob           string
	DirectoryPath           string
	Files                   []string
	GracefulShutdownTimeout time.Duration
	HealthListenAddress     string
	MetricsListenAddress    string
	MetricsPrefix           string
	TemporalAddress         string
	TemporalAPIKey          string
	TemporalMTLSCertPath    string
	TemporalMTLSKeyPath     string
	TemporalTLSEnabled      bool
	TemporalNamespace       string
	Validate                bool

	Telemetry *telemetry.Telemetry
}

// workflowRegistration holds a loaded and validated workflow definition ready
// to be registered on a Temporal worker. TaskQueue is derived from
// document.taskQueue. WorkflowType is derived from document.workflowType,
// which is the Temporal type identifier used during worker registration.
type workflowRegistration struct {
	SourceFile   string
	Definition   *model.Workflow
	Events       *cloudevents.Events
	TaskQueue    string
	WorkflowType string
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

func runValidation(validator *utils.Validator, workflowDefinition any) error {
	log.Debug().Msg("Running validation")
	res, err := validator.ValidateStruct(workflowDefinition)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error creating validation stack",
		}
	}
	if res != nil {
		return gh.FatalError{
			Msg: "Validation failed",
			WithParams: func(l *zerolog.Event) *zerolog.Event {
				f := []struct {
					Key     string
					Message string
				}{}
				for _, r := range res {
					f = append(f, struct {
						Key     string
						Message string
					}{
						Key:     r.Key,
						Message: r.Message,
					})
				}
				return l.Interface("validationErrors", f)
			},
		}
	}
	log.Debug().Msg("Validation passed")
	return nil
}

// discoverWorkflowFiles collects workflow file paths from --file flags and from
// --dir/--glob directory scanning. Both sources may be used together. Each path
// is normalised to an absolute path before deduplication, so that relative and
// absolute references to the same file are treated as one. Returns an error if
// no files are found.
func discoverWorkflowFiles(opts *runOptions) ([]string, error) {
	seen := make(map[string]struct{})
	var files []string

	addFile := func(f string) error {
		abs, err := filepath.Abs(f)
		if err != nil {
			return gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", f)
				},
				Msg: "Error resolving workflow file path",
			}
		}
		if _, ok := seen[abs]; !ok {
			seen[abs] = struct{}{}
			files = append(files, abs)
		}
		return nil
	}

	for _, f := range opts.Files {
		if err := addFile(f); err != nil {
			return nil, err
		}
	}

	if opts.DirectoryPath != "" {
		globbed, err := glob.Glob(opts.DirectoryPath, opts.DirectoryGlob)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				Msg:   "Error compiling glob",
			}
		}
		for _, f := range globbed {
			if err := addFile(f); err != nil {
				return nil, err
			}
		}
	}

	if len(files) == 0 {
		return nil, gh.FatalError{
			Msg: "No workflow files found",
		}
	}

	return files, nil
}

// loadWorkflows parses, optionally validates, and loads the CloudEvents handler
// for each file. Returns one workflowRegistration per file.
func loadWorkflows(
	files []string,
	cloudEventsConfig string,
	validator *utils.Validator,
	validate bool,
) ([]*workflowRegistration, error) {
	registrations := make([]*workflowRegistration, 0, len(files))

	for _, file := range files {
		if validate {
			if err := zigflow.ValidateFile(file); err != nil {
				return nil, gh.FatalError{
					Cause: err,
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Str("file", file)
					},
					Msg: "Schema validation failed",
				}
			}
		}

		def, err := zigflow.LoadFromFile(file)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Unable to load workflow file",
			}
		}

		// Defensive check: workflowType and taskQueue are used as Temporal
		// registration keys and worker-grouping keys respectively. An empty
		// value would silently produce a broken worker or a duplicate-key
		// collision, so reject such definitions here regardless of schema
		// validation.
		if def.Document.Name == "" {
			return nil, gh.FatalError{
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Workflow document.workflowType must not be empty",
			}
		}
		if def.Document.Namespace == "" {
			return nil, gh.FatalError{
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Workflow document.taskQueue must not be empty",
			}
		}

		if validate {
			if err := runValidation(validator, def); err != nil {
				return nil, err
			}
		}

		log.Debug().
			Str("file", file).
			Str("cloudEventsConfig", cloudEventsConfig).
			Msg("Registering CloudEvents handler")

		events, err := cloudevents.Load(cloudEventsConfig, validator, def)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Error creating CloudEvents handler",
			}
		}

		registrations = append(registrations, &workflowRegistration{
			SourceFile:   file,
			Definition:   def,
			Events:       events,
			TaskQueue:    def.Document.Namespace,
			WorkflowType: def.Document.Name,
		})
	}

	return registrations, nil
}

// validateWorkflowConflicts detects registrations that would conflict on the
// same Temporal worker. Temporal uses document.workflowType as the workflow
// type identifier (via RegisterWorkflowWithOptions), so two workflows with the
// same workflowType on the same taskQueue cannot coexist on a single worker.
func validateWorkflowConflicts(registrations []*workflowRegistration) error {
	// seen maps task queue -> workflow name -> source file
	seen := make(map[string]map[string]string)

	for _, reg := range registrations {
		if _, ok := seen[reg.TaskQueue]; !ok {
			seen[reg.TaskQueue] = make(map[string]string)
		}
		if existing, ok := seen[reg.TaskQueue][reg.WorkflowType]; ok {
			return gh.FatalError{
				Msg: "Duplicate workflow name on the same task queue",
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.
						Str("workflowType", reg.WorkflowType).
						Str("taskQueue", reg.TaskQueue).
						Str("file", reg.SourceFile).
						Str("conflictsWith", existing)
				},
			}
		}
		seen[reg.TaskQueue][reg.WorkflowType] = reg.SourceFile
	}

	return nil
}

// runScheduleUpdates updates Temporal schedules for all workflow registrations
// before any worker is started.
func runScheduleUpdates(
	ctx context.Context,
	temporalClient client.Client,
	registrations []*workflowRegistration,
	envvars map[string]any,
) error {
	for _, reg := range registrations {
		log.Info().Str("workflow", reg.WorkflowType).Msg("Updating schedules")
		if err := zigflow.UpdateSchedules(ctx, temporalClient, reg.Definition, envvars); err != nil {
			return gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.
						Str("workflow", reg.WorkflowType).
						Str("taskQueue", reg.TaskQueue).
						Str("file", reg.SourceFile)
				},
				Msg: "Error updating Temporal schedules",
			}
		}
	}
	return nil
}

// startAllWorkers starts each worker in the map and returns the list of
// successfully started workers. Workers are started in sorted task-queue order
// so that startup sequence is deterministic regardless of map iteration order.
// On failure it stops any workers that were already started before returning
// the error, so the caller does not need to track partial state.
func startAllWorkers(workers map[string]worker.Worker) ([]worker.Worker, error) {
	taskQueues := make([]string, 0, len(workers))
	for tq := range workers {
		taskQueues = append(taskQueues, tq)
	}
	sort.Strings(taskQueues)

	started := make([]worker.Worker, 0, len(workers))
	for _, taskQueue := range taskQueues {
		w := workers[taskQueue]
		log.Info().Str("task-queue", taskQueue).Msg("Starting worker")
		if err := w.Start(); err != nil {
			for _, sw := range started {
				sw.Stop()
			}
			return nil, gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("task-queue", taskQueue)
				},
				Msg: "Unable to start worker",
			}
		}
		started = append(started, w)
	}
	return started, nil
}

// buildWorkersByTaskQueue creates one Temporal worker per distinct task queue
// and registers all workflow definitions onto the appropriate worker. Workflows
// that share a task queue are registered on the same worker.
func buildWorkersByTaskQueue(
	temporalClient client.Client,
	registrations []*workflowRegistration,
	envvars map[string]any,
	opts *runOptions,
) (map[string]worker.Worker, error) {
	workers := make(map[string]worker.Worker)

	for _, reg := range registrations {
		w, ok := workers[reg.TaskQueue]
		if !ok {
			pollerAutoscaler := worker.NewPollerBehaviorAutoscaling(worker.PollerBehaviorAutoscalingOptions{})
			w = worker.New(temporalClient, reg.TaskQueue, worker.Options{
				WorkflowTaskPollerBehavior: pollerAutoscaler,
				ActivityTaskPollerBehavior: pollerAutoscaler,
				NexusTaskPollerBehavior:    pollerAutoscaler,
				WorkerStopTimeout:          opts.GracefulShutdownTimeout,
			})
			workers[reg.TaskQueue] = w
			log.Debug().Str("task-queue", reg.TaskQueue).Msg("Created worker for task queue")
			activities := tasks.ActivitiesList()
			log.Debug().
				Str("task-queue", reg.TaskQueue).
				Int("count", len(activities)).
				Msg("Registering shared activities on worker")
			for _, a := range activities {
				w.RegisterActivity(a)
			}
		}

		log.Info().
			Str("task-queue", reg.TaskQueue).
			Str("workflow", reg.WorkflowType).
			Str("file", reg.SourceFile).
			Msg("Registering workflow")

		if err := zigflow.NewWorkflow(w, reg.Definition, envvars, reg.Events, opts.Telemetry); err != nil {
			return nil, gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.
						Str("workflow", reg.WorkflowType).
						Str("file", reg.SourceFile)
				},
				Msg: "Unable to build workflow from DSL",
			}
		}
	}

	return workers, nil
}

// prepareRegistrations discovers, loads, and validates all workflow files.
// It encapsulates the pipeline from path resolution through conflict detection
// so that runRunCmd stays within a manageable cyclomatic complexity budget.
func prepareRegistrations(opts *runOptions) ([]*workflowRegistration, error) {
	files, err := discoverWorkflowFiles(opts)
	if err != nil {
		return nil, err
	}

	log.Debug().Int("count", len(files)).Msg("Discovered workflow files")

	validator, err := utils.NewValidator()
	if err != nil {
		return nil, gh.FatalError{Cause: err, Msg: "Error creating validator"}
	}

	registrations, err := loadWorkflows(files, opts.CloudEventsConfig, validator, opts.Validate)
	if err != nil {
		return nil, err
	}

	if err := validateWorkflowConflicts(registrations); err != nil {
		return nil, err
	}

	return registrations, nil
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

	codecType, _ := codec.ParseCodecType(opts.ConvertData)
	dataConverter, err := codec.NewDataConverter(codecType, opts.CodecEndpoint, opts.ConvertKeyPath, opts.CodecHeaders)
	if err != nil {
		return err
	}

	// The Temporal client is a heavyweight object created once per process.
	log.Trace().Msg("Connecting to Temporal")
	temporalClient, err := temporal.NewConnection(
		temporal.WithHostPort(opts.TemporalAddress),
		temporal.WithNamespace(opts.TemporalNamespace),
		temporal.WithTLS(opts.TemporalTLSEnabled),
		temporal.WithAuthDetection(
			opts.TemporalAPIKey,
			opts.TemporalMTLSCertPath,
			opts.TemporalMTLSKeyPath,
		),
		temporal.WithDataConverter(dataConverter),
		temporal.WithZerolog(&log.Logger),
		temporal.WithPrometheusMetrics(opts.MetricsListenAddress, opts.MetricsPrefix, nil),
	)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create client",
		}
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

	workers, err := buildWorkersByTaskQueue(temporalClient, registrations, envvars, opts)
	if err != nil {
		return err
	}

	taskQueues := make([]string, 0)
	for taskQueue := range workers {
		taskQueues = append(taskQueues, taskQueue)
	}

	temporal.NewHealthCheck(ctx, taskQueues, opts.HealthListenAddress, temporalClient)
	if opts.Telemetry != nil {
		opts.Telemetry.StartWorker()
	}

	// startAllWorkers handles its own partial cleanup on failure, so no
	// started workers leak if one fails to start.
	startedWorkers, err := startAllWorkers(workers)
	if err != nil {
		return err
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
	select {
	case <-worker.InterruptCh():
		log.Info().Msg("Received interrupt signal")
	case <-ctx.Done():
		log.Info().Msg("Context cancelled")
	}

	return nil
}

// registerWorkflowSourceFlags registers the flags that control where workflow
// definitions are loaded from: explicit file paths, directory scanning, and
// the CloudEvents config that accompanies them.
func registerWorkflowSourceFlags(cmd *cobra.Command, opts *runOptions) {
	cmd.Flags().StringVar(
		&opts.CloudEventsConfig, "cloudevents-config",
		viper.GetString("cloudevents_config"), "Path to CloudEvents config file",
	)

	cmd.Flags().StringVarP(
		&opts.DirectoryPath, "dir", "d",
		viper.GetString("workflow_directory"), "Directory containing workflow files",
	)

	// Envvars are delimited by ", "
	cmd.Flags().StringSliceVarP(
		&opts.Files, "file", "f",
		viper.GetStringSlice("workflow_file"), "Path to workflow file (may be specified multiple times)",
	)

	viper.SetDefault("workflow_directory_glob", "*.{yaml,yml,json}")
	cmd.Flags().StringVar(
		&opts.DirectoryGlob, "glob",
		viper.GetString("workflow_directory_glob"), "Glob pattern when using --dir",
	)
}

// registerTemporalConnectionFlags registers the flags that govern how the
// process connects to Temporal: address, namespace, authentication, and TLS.
func registerTemporalConnectionFlags(cmd *cobra.Command, opts *runOptions) {
	viper.SetDefault("temporal_address", client.DefaultHostPort)
	cmd.Flags().StringVarP(
		&opts.TemporalAddress, "temporal-address", "H",
		viper.GetString("temporal_address"), "Address of the Temporal server",
	)

	cmd.Flags().StringVar(
		&opts.TemporalAPIKey, "temporal-api-key",
		viper.GetString("temporal_api_key"), "API key for Temporal authentication",
	)
	// Hide the default value to avoid spaffing the API to command line
	gh.HideCommandOutput(cmd, "temporal-api-key")

	cmd.Flags().StringVar(
		&opts.TemporalMTLSCertPath, "tls-client-cert-path",
		viper.GetString("temporal_tls_client_cert_path"), "Path to mTLS client cert, usually ending in .pem",
	)

	cmd.Flags().StringVar(
		&opts.TemporalMTLSKeyPath, "tls-client-key-path",
		viper.GetString("temporal_tls_client_key_path"), "Path to mTLS client key, usually ending in .key",
	)

	viper.SetDefault("temporal_namespace", client.DefaultNamespace)
	cmd.Flags().StringVarP(
		&opts.TemporalNamespace, "temporal-namespace", "n",
		viper.GetString("temporal_namespace"), "Temporal namespace to use",
	)

	cmd.Flags().BoolVar(
		&opts.TemporalTLSEnabled, "temporal-tls",
		viper.GetBool("temporal_tls"), "Enable TLS Temporal connection",
	)
}

func registerRunFlags(cmd *cobra.Command, opts *runOptions) {
	registerWorkflowSourceFlags(cmd, opts)
	registerTemporalConnectionFlags(cmd, opts)

	cmd.Flags().StringVar(
		&opts.CodecEndpoint, "codec-endpoint",
		viper.GetString("codec_endpoint"), "Remote codec server endpoint",
	)

	cmd.Flags().StringToStringVar(
		&opts.CodecHeaders, "codec-headers",
		viper.GetStringMapString("codec_headers"), "Remote codec server headers",
	)
	gh.HideCommandOutput(cmd, "codec-headers")

	cmd.Flags().StringVar(
		&opts.ConvertData, "convert-data",
		viper.GetString("convert_data"), fmt.Sprintf("Data conversion mode: %q, %q, or %q", codec.CodecNone, codec.CodecAES, codec.CodecRemote),
	)

	viper.SetDefault("converter_key_path", "keys.yaml")
	cmd.Flags().StringVar(
		&opts.ConvertKeyPath, "converter-key-path",
		viper.GetString("converter_key_path"), "Path to conversion keys to encrypt Temporal data with AES",
	)

	viper.SetDefault("env_prefix", "ZIGGY")
	cmd.Flags().StringVar(
		&opts.EnvPrefix, "env-prefix",
		viper.GetString("env_prefix"), "Load envvars with this prefix to the workflow",
	)

	viper.SetDefault("graceful_shutdown_timeout", time.Second*10)
	cmd.Flags().DurationVar(
		&opts.GracefulShutdownTimeout, "graceful-shutdown-timeout",
		viper.GetDuration("graceful_shutdown_timeout"), "Maximum time to wait for in-flight work to complete on shutdown. Set to 0 to disable",
	)

	viper.SetDefault("health_listen_address", "0.0.0.0:3000")
	cmd.Flags().StringVar(
		&opts.HealthListenAddress, "health-listen-address",
		viper.GetString("health_listen_address"), "Address of health server",
	)

	viper.SetDefault("metrics_listen_address", "0.0.0.0:9090")
	cmd.Flags().StringVar(
		&opts.MetricsListenAddress, "metrics-listen-address",
		viper.GetString("metrics_listen_address"), "Address of Prometheus metrics server",
	)

	cmd.Flags().StringVar(
		&opts.MetricsPrefix, "metrics-prefix",
		viper.GetString("metrics_prefix"), "Prefix for metrics",
	)

	viper.SetDefault("validate", true)
	cmd.Flags().BoolVar(
		&opts.Validate, "validate",
		viper.GetBool("validate"), "Run workflow validation",
	)
}

func newRunCmd() *cobra.Command {
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
			opts.Telemetry = app.Telemetry
			return runRunCmd(cmd.Context(), &opts)
		},
	}

	registerRunFlags(cmd, &opts)

	return cmd
}

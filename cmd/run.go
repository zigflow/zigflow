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
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
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
	Watch                   bool
	WatchDebounce           time.Duration

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

// launchWorkers prepares registrations, builds workers, and starts them.
// On any error it stops any workers that were partially started before returning.
func launchWorkers(
	temporalClient client.Client,
	opts *runOptions,
	envvars map[string]any,
) ([]worker.Worker, error) {
	registrations, err := prepareRegistrations(opts)
	if err != nil {
		return nil, err
	}

	workers, err := buildWorkersByTaskQueue(temporalClient, registrations, envvars, opts)
	if err != nil {
		return nil, err
	}

	return startAllWorkers(workers)
}

// stopWorkerList stops each worker in the slice.
func stopWorkerList(workers []worker.Worker) {
	log.Info().Int("count", len(workers)).Msg("Watch: stopping workers")
	for _, w := range workers {
		w.Stop()
	}
}

// newStoppedTimer creates a timer and immediately stops it, draining any
// pending tick. Use timer.Reset(d) to arm it for the first time.
func newStoppedTimer(d time.Duration) *time.Timer {
	t := time.NewTimer(d)
	if !t.Stop() {
		<-t.C
	}
	return t
}

// resetDebounce safely stops t and resets it to d, draining any pending tick
// so the timer fires exactly once after d has elapsed.
func resetDebounce(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// isWatchableEvent reports whether the fsnotify event should trigger a reload.
// Write covers in-place saves; Create and Rename cover atomic editor patterns
// such as vim's rename-over-original approach.
func isWatchableEvent(e fsnotify.Event) bool {
	return e.Has(fsnotify.Write) || e.Has(fsnotify.Create) || e.Has(fsnotify.Rename)
}

// handleDebounce executes a reload cycle after the debounce timer fires.
// It logs the changed files, attempts to rebuild workers, swaps them on success,
// and always refreshes the watcher to recover inodes lost by rename-style saves.
// It returns the updated current workers and a fresh (empty) changedFiles map.
func handleDebounce(
	watcher *fsnotify.Watcher,
	temporalClient client.Client,
	opts *runOptions,
	envvars map[string]any,
	changedFiles map[string]struct{},
	current []worker.Worker,
) (nextWorkers []worker.Worker, remainingChanges map[string]struct{}) {
	if len(changedFiles) == 0 {
		return current, changedFiles
	}

	names := make([]string, 0, len(changedFiles))
	for f := range changedFiles {
		names = append(names, filepath.Base(f))
	}
	sort.Strings(names)
	log.Warn().Str("files", strings.Join(names, ", ")).Msg("Watch: reloading workers")

	next, loadErr := launchWorkers(temporalClient, opts, envvars)
	if loadErr != nil {
		log.Error().Err(loadErr).Msg("Watch: reload failed, keeping existing workers")
	} else {
		stopWorkerList(current)
		current = next
		log.Info().Int("count", len(current)).Msg("Watch: workers reloaded successfully")
	}
	if err := refreshWatcher(watcher, opts); err != nil {
		log.Error().Err(err).Msg("Watch: failed to refresh file watches")
	}
	return current, make(map[string]struct{})
}

// refreshWatcher removes all currently watched paths and re-adds the resolved
// workflow files. This is called after every debounce-triggered reload (whether
// it succeeded or failed) to recover watches that were lost because an editor
// replaced a file via a temp-file rename, which causes fsnotify to silently
// drop the watch on the original inode.
//
// Refresh is two-phase: all target files are added first (fsnotify.Add is
// idempotent for already-watched paths), and stale paths are only removed after
// every add succeeds. This means a failed add leaves the previous watch set
// intact rather than leaving watch mode partially disabled.
func refreshWatcher(w *fsnotify.Watcher, opts *runOptions) error {
	// Discover files first. If discovery fails, leave the watcher unchanged so
	// subsequent events can still be received.
	files, err := discoverWorkflowFiles(opts)
	if err != nil {
		return err
	}

	// Phase 1: add all target files. fsnotify.Add is idempotent for paths that
	// are already watched, so this also refreshes inodes lost by rename-style
	// saves. On any failure, return before touching the existing watch list.
	target := make(map[string]struct{}, len(files))
	for _, f := range files {
		target[f] = struct{}{}
		if err := w.Add(f); err != nil {
			return fmt.Errorf("watch: re-add %s: %w", f, err)
		}
	}

	// Phase 2: remove paths that are no longer in the target set.
	for _, p := range w.WatchList() {
		if _, ok := target[p]; !ok {
			_ = w.Remove(p)
		}
	}
	return nil
}

// runWatchMode watches files for changes and reloads workers on each change.
// It blocks until ctx is cancelled or an interrupt signal is received.
// On reload failure it logs the error and keeps the existing workers running
// so the system is never left with zero workers.
func runWatchMode(
	ctx context.Context,
	files []string,
	temporalClient client.Client,
	opts *runOptions,
	envvars map[string]any,
	current []worker.Worker,
) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watch: create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	for _, f := range files {
		if err := watcher.Add(f); err != nil {
			return fmt.Errorf("watch: add %s: %w", f, err)
		}
	}
	log.Info().
		Int("count", len(files)).
		Dur("debounce", opts.WatchDebounce).
		Msg("Watch: watching workflow files for changes")

	defer func() { stopWorkerList(current) }()

	debounce := newStoppedTimer(opts.WatchDebounce)

	// List of changed files - use map to autodedupe
	changedFiles := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-worker.InterruptCh():
			log.Info().Msg("Watch: received interrupt signal")
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if isWatchableEvent(event) {
				changedFiles[event.Name] = struct{}{}

				log.Debug().
					Str("file", event.Name).
					Str("op", event.Op.String()).
					Msg("Watch: file change detected, debouncing")
				resetDebounce(debounce, opts.WatchDebounce)
			}
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Error().Err(watchErr).Msg("Watch: watcher error")
		case <-debounce.C:
			current, changedFiles = handleDebounce(watcher, temporalClient, opts, envvars, changedFiles, current)
		}
	}
}

// initTemporalClient creates the codec data converter and the Temporal client.
// The caller is responsible for closing the returned client.
func initTemporalClient(opts *runOptions) (client.Client, error) {
	codecType, _ := codec.ParseCodecType(opts.ConvertData)
	dataConverter, err := codec.NewDataConverter(codecType, opts.CodecEndpoint, opts.ConvertKeyPath, opts.CodecHeaders)
	if err != nil {
		return nil, err
	}

	log.Trace().Msg("Connecting to Temporal")
	tc, err := temporal.NewConnection(
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
		return nil, gh.FatalError{Cause: err, Msg: "Unable to create client"}
	}
	return tc, nil
}

// startInitialWorkers builds workers from registrations, registers the health
// check, starts telemetry, and starts all workers. It is the single call that
// takes the process from validated registrations to running workers.
func startInitialWorkers(
	ctx context.Context,
	tc client.Client,
	registrations []*workflowRegistration,
	envvars map[string]any,
	opts *runOptions,
) ([]worker.Worker, error) {
	workers, err := buildWorkersByTaskQueue(tc, registrations, envvars, opts)
	if err != nil {
		return nil, err
	}

	taskQueues := make([]string, 0, len(workers))
	for tq := range workers {
		taskQueues = append(taskQueues, tq)
	}
	temporal.NewHealthCheck(ctx, taskQueues, opts.HealthListenAddress, tc)

	if opts.Telemetry != nil {
		opts.Telemetry.StartWorker()
	}

	return startAllWorkers(workers)
}

// waitForShutdown blocks until the process receives an interrupt signal or the
// context is cancelled.
func waitForShutdown(ctx context.Context) {
	select {
	case <-worker.InterruptCh():
		log.Info().Msg("Received interrupt signal")
	case <-ctx.Done():
		log.Info().Msg("Context cancelled")
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

	cmd.Flags().BoolVar(
		&opts.Watch, "watch",
		viper.GetBool("watch"), "Reload workers automatically when workflow files change (for development use)",
	)

	viper.SetDefault("watch_debounce", 300*time.Millisecond)
	cmd.Flags().DurationVar(
		&opts.WatchDebounce, "watch-debounce",
		viper.GetDuration("watch_debounce"), "Debounce duration for file change events when using --watch",
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

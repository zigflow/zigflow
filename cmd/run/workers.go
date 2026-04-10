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
	"sort"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zigflow/zigflow/pkg/codec"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"github.com/zigflow/zigflow/pkg/zigflow/tasks"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// newTemporalConnection is the function used to establish a Temporal client. It
// is a package-level variable so tests can substitute a test double without
// spinning up a real Temporal server.
var newTemporalConnection = temporal.NewConnection

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
	if opts.MaxConcurrentWorkflowTaskExecutionSize == 1 {
		return nil, gh.FatalError{
			Msg: "Max concurrent workflow task execution size cannot be set to 1",
		}
	}

	workers := make(map[string]worker.Worker)

	for _, reg := range registrations {
		w, ok := workers[reg.TaskQueue]
		if !ok {
			pollerAutoscaler := worker.NewPollerBehaviorAutoscaling(worker.PollerBehaviorAutoscalingOptions{})
			w = worker.New(temporalClient, reg.TaskQueue, worker.Options{
				WorkflowTaskPollerBehavior:             pollerAutoscaler,
				ActivityTaskPollerBehavior:             pollerAutoscaler,
				NexusTaskPollerBehavior:                pollerAutoscaler,
				WorkerStopTimeout:                      opts.GracefulShutdownTimeout,
				MaxConcurrentActivityExecutionSize:     opts.MaxConcurrentActivityExecutionSize,
				MaxConcurrentWorkflowTaskExecutionSize: opts.MaxConcurrentWorkflowTaskExecutionSize,
				TaskQueueActivitiesPerSecond:           opts.TaskQueueActivitiesPerSecond,
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

// initTemporalClient creates the codec data converter and the Temporal client.
// The caller is responsible for closing the returned client.
func initTemporalClient(opts *runOptions) (client.Client, error) {
	codecType, _ := codec.ParseCodecType(opts.ConvertData)
	dataConverter, err := codec.NewDataConverter(codecType, opts.CodecEndpoint, opts.ConvertKeyPath, opts.CodecHeaders)
	if err != nil {
		return nil, err
	}

	log.Trace().Msg("Connecting to Temporal")
	tc, err := newTemporalConnection(
		temporal.WithHostPort(opts.TemporalAddress),
		temporal.WithNamespace(opts.TemporalNamespace),
		temporal.WithTLS(opts.TemporalTLSEnabled, temporal.WithTLSServerName(opts.TemporalServerName)),
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

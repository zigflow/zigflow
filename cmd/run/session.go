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

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog/log"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/codec"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// TestTaskQueue is the fixed Temporal task queue used by zigflow test. It is set
// once on session runOptions so worker registration and workflow starts stay aligned.
// document.taskQueue in workflow YAML is not used for test runs in v1.
const TestTaskQueue = "zigflow-test"

// SessionConfig configures a short-lived in-process worker used by zigflow test.
type SessionConfig struct {
	Workflow  *model.Workflow
	Source    string
	Temporal  *temporal.TemporalOpts
	Telemetry *telemetry.Telemetry
}

// Session runs Zigflow workers and exposes the Temporal client used to register them.
type Session struct {
	client    client.Client
	workers   []worker.Worker
	taskQueue string
}

// Client returns the Temporal client for this session. The client is closed when
// the session stops.
func (s *Session) Client() client.Client {
	return s.client
}

// TaskQueue returns the Temporal task queue this session registered workers on and
// that callers should use for ExecuteWorkflow.
func (s *Session) TaskQueue() string {
	return s.taskQueue
}

// Stop stops workers and closes the Temporal client.
func (s *Session) Stop() {
	for _, w := range s.workers {
		w.Stop()
	}
	if s.client != nil {
		log.Trace().Msg("Closing Temporal connection")
		s.client.Close()
	}
}

// StartSession prepares an already loaded workflow, starts workers, and returns
// a session that must be stopped by the caller. It does not enable watch mode
// or block on signals.
func StartSession(ctx context.Context, cfg SessionConfig) (*Session, error) {
	if cfg.Workflow == nil {
		return nil, fmt.Errorf("workflow is required")
	}
	if cfg.Source == "" {
		cfg.Source = "<memory>"
	}

	opts, err := runOptionsForSession(cfg)
	if err != nil {
		return nil, err
	}

	validator, err := utils.NewValidator()
	if err != nil {
		return nil, gh.FatalError{Cause: err, Msg: "Error creating validator"}
	}

	registration, err := buildWorkflowRegistration(
		cfg.Source,
		cfg.Workflow,
		"",
		validator,
		true,
		opts.registrationTaskQueue,
	)
	if err != nil {
		return nil, err
	}
	registrations := []*workflowRegistration{registration}

	temporalClient, err := initTemporalClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	prefix := opts.EnvPrefix + "_"
	envvars := utils.LoadEnvvars(prefix)

	if !opts.skipScheduleUpdates {
		if err := runScheduleUpdates(ctx, temporalClient, registrations, envvars); err != nil {
			temporalClient.Close()
			return nil, err
		}
	}

	startedWorkers, err := startInitialWorkers(ctx, temporalClient, registrations, envvars, opts)
	if err != nil {
		temporalClient.Close()
		return nil, err
	}

	return &Session{
		client:    temporalClient,
		workers:   startedWorkers,
		taskQueue: opts.registrationTaskQueue,
	}, nil
}

func runOptionsForSession(cfg SessionConfig) (*runOptions, error) {
	if cfg.Temporal == nil {
		cfg.Temporal = &temporal.TemporalOpts{}
	}
	if cfg.Temporal.HealthListenAddress == "" {
		cfg.Temporal.HealthListenAddress = "127.0.0.1:0"
	}
	if _, ok := activities.ValidContainerRuntimes[activities.ContainerRuntime("docker")]; !ok {
		return nil, fmt.Errorf("internal error: docker container runtime not registered")
	}

	opts := &runOptions{
		EnvPrefix:             "ZIGGY",
		ContainerRuntime:      string(activities.ContainerRuntimeDocker),
		ConvertData:           string(codec.CodecNone),
		ConvertFailureData:    true,
		ConvertKeyPath:        "keys.yaml",
		Telemetry:             cfg.Telemetry,
		temporal:              cfg.Temporal,
		registrationTaskQueue: TestTaskQueue,
		skipScheduleUpdates:   true,
		skipClientMetrics:     true,
	}

	if _, err := codec.ParseCodecType(opts.ConvertData); err != nil {
		return nil, err
	}

	return opts, nil
}

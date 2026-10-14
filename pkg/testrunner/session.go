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

package testrunner

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/interceptors"
	"github.com/zigflow/zigflow/pkg/utils"
	workerconfig "github.com/zigflow/zigflow/pkg/worker"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"github.com/zigflow/zigflow/pkg/zigflow/tasks"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

// TaskQueuePrefix identifies isolated test queues in Temporal.
const TaskQueuePrefix = "zigflow-test-"

type session struct {
	client    client.Client
	worker    worker.Worker
	taskQueue string
}

func (s *session) stop() {
	s.worker.Stop()
	s.client.Close()
}

func startSession(ctx context.Context, wf *model.Workflow, cfg *Config) (*session, error) {
	taskOpts, err := sessionTaskOptions(cfg.TaskOptions)
	if err != nil {
		return nil, err
	}
	validator, err := utils.NewValidator()
	if err != nil {
		return nil, err
	}
	failures, err := validator.ValidateStruct(wf)
	if err != nil {
		return nil, err
	}
	if len(failures) > 0 {
		return nil, fmt.Errorf("workflow validation failed: %v", failures)
	}
	events, err := cloudevents.Load(cfg.CloudEventsConfig, validator, wf)
	if err != nil {
		return nil, err
	}
	options, err := workerconfig.ClientOptions(ctx, cfg.Temporal, &cfg.Client)
	if err != nil {
		return nil, err
	}
	c, err := temporal.NewConnection(options...)
	if err != nil {
		return nil, err
	}
	s := &session{client: c, taskQueue: TaskQueuePrefix + uuid.NewString()}
	s.worker = worker.New(c, s.taskQueue, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{interceptors.NewLoggerInterceptor()},
	})
	for _, activity := range tasks.ActivitiesList() {
		s.worker.RegisterActivity(activity)
	}
	envvars := cfg.Envvars
	if envvars == nil {
		envvars = utils.LoadEnvvars("ZIGGY_")
	}
	if err := zigflow.NewWorkflow(s.worker, wf, envvars, events, cfg.Telemetry, taskOpts); err != nil {
		c.Close()
		return nil, err
	}
	if cfg.Telemetry != nil {
		cfg.Telemetry.StartWorker()
	}
	if err := s.worker.Start(); err != nil {
		c.Close()
		return nil, err
	}
	return s, nil
}

func sessionTaskOptions(configured *tasks.TaskOpts) (*tasks.TaskOpts, error) {
	var opts tasks.TaskOpts
	if configured != nil {
		opts = *configured
	}
	var runOpts tasks.RunTaskOpts
	if opts.Run != nil {
		runOpts = *opts.Run
	}
	if runOpts.Runtime == "" {
		runOpts.Runtime = activities.ContainerRuntimeDocker
	}
	if _, ok := activities.ValidContainerRuntimes[runOpts.Runtime]; !ok {
		return nil, fmt.Errorf("invalid container runtime %q", runOpts.Runtime)
	}
	opts.Run = &runOpts
	return &opts, nil
}

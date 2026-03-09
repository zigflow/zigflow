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
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type runOptions struct {
	CloudEventsConfig    string
	CodecEndpoint        string
	CodecHeaders         map[string]string
	ConvertData          string
	ConvertKeyPath       string
	EnvPrefix            string
	FilePath             string
	HealthListenAddress  string
	MetricsListenAddress string
	MetricsPrefix        string
	TemporalAddress      string
	TemporalAPIKey       string
	TemporalMTLSCertPath string
	TemporalMTLSKeyPath  string
	TemporalTLSEnabled   bool
	TemporalNamespace    string
	Validate             bool

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

func startWorker(
	temporalClient client.Client,
	taskQueue string,
	workflowDefinition *model.Workflow,
	envvars map[string]any,
	events *cloudevents.Events,
	telem *telemetry.Telemetry,
) error {
	pollerAutoscaler := worker.NewPollerBehaviorAutoscaling(worker.PollerBehaviorAutoscalingOptions{})
	temporalWorker := worker.New(temporalClient, taskQueue, worker.Options{
		WorkflowTaskPollerBehavior: pollerAutoscaler,
		ActivityTaskPollerBehavior: pollerAutoscaler,
		NexusTaskPollerBehavior:    pollerAutoscaler,
	})

	if err := zigflow.NewWorkflow(temporalWorker, workflowDefinition, envvars, events, telem); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to build workflow from DSL",
		}
	}

	if telem != nil {
		telem.StartWorker()
		defer telem.Shutdown()
	}

	if err := temporalWorker.Run(worker.InterruptCh()); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to start worker",
		}
	}

	return nil
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

	workflowDefinition, err := zigflow.LoadFromFile(opts.FilePath)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to load workflow file"}
	}

	validator, err := utils.NewValidator()
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Error creating validator"}
	}

	if opts.Validate {
		if err := runValidation(validator, workflowDefinition); err != nil {
			return err
		}
	}

	log.Debug().Str("cloudEventsConfig", opts.CloudEventsConfig).Msg("Registering CloudEvents handler")
	events, err := cloudevents.Load(opts.CloudEventsConfig, validator, workflowDefinition)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error creating CloudEvents handler",
		}
	}

	codecType, _ := codec.ParseCodecType(opts.ConvertData)
	dataConverter, err := codec.NewDataConverter(codecType, opts.CodecEndpoint, opts.ConvertKeyPath, opts.CodecHeaders)
	if err != nil {
		return err
	}

	// The client and worker are heavyweight objects that should be created once per process.
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
	defer func() {
		log.Trace().Msg("Closing Temporal connection")
		temporalClient.Close()
		log.Trace().Msg("Temporal connection closed")
	}()

	taskQueue := workflowDefinition.Document.Namespace
	prefix := opts.EnvPrefix + "_"

	log.Debug().Str("prefix", prefix).Msg("Loading envvars to state")
	envvars := utils.LoadEnvvars(prefix)

	log.Debug().Msg("Starting health check service")
	temporal.NewHealthCheck(ctx, taskQueue, opts.HealthListenAddress, temporalClient)

	log.Info().Msg("Updating schedules")
	if err := zigflow.UpdateSchedules(ctx, temporalClient, workflowDefinition, envvars); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error updating Temporal schedules",
		}
	}

	log.Info().Str("task-queue", taskQueue).Msg("Starting workflow")

	return startWorker(temporalClient, taskQueue, workflowDefinition, envvars, events, opts.Telemetry)
}

func registerRunFlags(cmd *cobra.Command, opts *runOptions) {
	cmd.Flags().StringVar(
		&opts.CloudEventsConfig, "cloudevents-config",
		viper.GetString("cloudevents_config"), "Path to CloudEvents config file",
	)

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

	cmd.Flags().StringVarP(
		&opts.FilePath, "file", "f",
		viper.GetString("workflow_file"), "Path to workflow file",
	)

	viper.SetDefault("env_prefix", "ZIGGY")
	cmd.Flags().StringVar(
		&opts.EnvPrefix, "env-prefix",
		viper.GetString("env_prefix"), "Load envvars with this prefix to the workflow",
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
		Short: "Start the Zigflow workflow worker",
		Long: `Start a Zigflow workflow worker that connects to Temporal and processes
workflow executions defined in the provided workflow file.

The worker loads the workflow definition from the specified file, validates it
(by default), and registers it with Temporal using the workflow's namespace as
the task queue. The worker then polls Temporal for workflow and activity tasks
until interrupted.

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

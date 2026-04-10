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
	"fmt"
	"time"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zigflow/zigflow/pkg/codec"
	"go.temporal.io/sdk/client"
)

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

	cmd.Flags().StringVar(
		&opts.TemporalServerName, "temporal-server-name",
		viper.GetString("temporal_server_name"),
		"Override the TLS server name (SNI) used for certificate validation. "+
			"Required when the endpoint address does not match the certificate hostname, for example AWS PrivateLink.",
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

	cmd.Flags().IntVar(
		&opts.MaxConcurrentActivityExecutionSize, "max-concurrent-activity-execution-size",
		viper.GetInt("max_concurrent_activity_execution_size"),
		"Sets the maximum concurrent activity executions this worker can have.",
	)

	cmd.Flags().IntVar(
		&opts.MaxConcurrentWorkflowTaskExecutionSize, "max-concurrent-workflow-task-execution-size",
		viper.GetInt("max_concurrent_workflow_task_execution_size"),
		"Sets the maximum concurrent workflow task executions this worker can have.",
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

	cmd.Flags().Float64Var(
		&opts.TaskQueueActivitiesPerSecond, "task-queue-activities-per-second",
		viper.GetFloat64("task_queue_activities_per_second"),
		"Sets the rate limiting on number of activities that can be executed per second.",
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

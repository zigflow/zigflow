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
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zigflow/zigflow/pkg/codec"
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

func registerVersioningFlags(cmd *cobra.Command, opts *runOptions) {
	viper.SetDefault("default_versioning_type", versioningBehaviourAutoUpgrade)
	cmd.Flags().StringVar(
		&opts.DefaultVersioningBehaviour, "default-versioning-type",
		viper.GetString("default_versioning_type"), fmt.Sprintf(
			"Default versioning type: %q, %q, or %q",
			versioningBehaviourUnspecified,
			versioningBehaviourPinned,
			versioningBehaviourAutoUpgrade,
		),
	)

	cmd.Flags().StringVar(
		&opts.DeploymentBuildID, "temporal-worker-build-id",
		viper.GetString("temporal_worker_build_id"), "The build id specific to this worker",
	)

	cmd.Flags().StringVar(
		&opts.DeploymentName, "temporal-deployment-name",
		viper.GetString("temporal_deployment_name"), "The name of the deployment this worker version belongs to",
	)

	cmd.Flags().BoolVar(
		&opts.EnableVersioning, "enable-versioning",
		viper.GetBool("enable_versioning"), "Enable Temporal worker versioning",
	)
}

// registerContainerRuntimeFlags registers the flags that control how
// run.container tasks dispatch to a container runtime: which runtime to use,
// and the namespace and service account that runtime should run workloads
// under.
func registerContainerRuntimeFlags(cmd *cobra.Command, opts *runOptions) {
	viper.SetDefault("container_runtime", "docker")
	cmd.Flags().StringVar(
		&opts.ContainerRuntime, "container-runtime",
		viper.GetString("container_runtime"), "Container runtime to use for `run.container` tasks. Can be `docker` or `kubernetes`",
	)

	cmd.Flags().StringVar(
		&opts.ContainerRuntimeNamespace, "container-runtime-namespace",
		viper.GetString("container_runtime_namespace"), "Namespace to use for the container runtime",
	)

	cmd.Flags().StringVar(
		&opts.ContainerRuntimeServiceAccount, "container-runtime-service-account",
		viper.GetString("container_runtime_service_account"), "Service account to use for the container runtime",
	)
}

func registerRunFlags(cmd *cobra.Command, opts *runOptions) {
	registerWorkflowSourceFlags(cmd, opts)
	temporal.NewCobraOpts(cmd, opts.temporal)
	registerVersioningFlags(cmd, opts)
	registerContainerRuntimeFlags(cmd, opts)

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

	viper.SetDefault("convert_failure_data", true)
	cmd.Flags().BoolVar(
		&opts.ConvertFailureData, "convert-failure-data",
		viper.GetBool("convert_failure_data"), "Convert failure payloads as well as workflow data (disable for readable UI errors)",
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

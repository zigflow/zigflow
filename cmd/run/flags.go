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
	"strings"
	"time"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	temporal "github.com/zigflow/helpers"
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

func registerExternalStorageFlags(cmd *cobra.Command, opts *runOptions) {
	cmd.Flags().StringVar(
		&opts.ExternalStorage, "external-storage",
		viper.GetString("external_storage"), "External storage type",
	)

	cmd.Flags().IntVar(
		&opts.ExternalStoragePayloadSizeThreshold, "external-storage-payload-size-threshold",
		viper.GetInt("external_storage_payload_size_threshold"), "Configure size threshold to send to external storage. Defaults to 256KB",
	)

	registerS3ExternalStorageFlags(cmd, opts)
	registerRedisExternalStorageFlags(cmd, opts)
}

func registerRedisExternalStorageFlags(cmd *cobra.Command, opts *runOptions) {
	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisDriverName, "external-storage-redis-driver-name",
		viper.GetString("external_storage_redis_driver_name"), "Driver name if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisKeyPrefix, "external-storage-redis-key-prefix",
		viper.GetString("external_storage_redis_key_prefix"), "Record key prefix if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisAddress, "external-storage-redis-address",
		viper.GetString("external_storage_redis_address"), "Server address if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisUsername, "external-storage-redis-username",
		viper.GetString("external_storage_redis_username"), "Server username if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisPassword, "external-storage-redis-password",
		viper.GetString("external_storage_redis_password"), "Server password if using external storage with Redis",
	)
	gh.HideCommandOutput(cmd, "external-storage-redis-password")

	cmd.Flags().IntVar(
		&opts.ExternalStorageRedisDB, "external-storage-redis-database",
		viper.GetInt("external_storage_redis_database"), "Database if using external storage with Redis",
	)

	cmd.Flags().DurationVar(
		&opts.ExternalStorageRedisTTL, "external-storage-redis-ttl",
		viper.GetDuration("external_storage_redis_ttl"), "Record TTL if using external storage with Redis",
	)

	cmd.Flags().BoolVar(
		&opts.ExternalStorageRedisTLSEnabled, "external-storage-redis-tls-enabled",
		viper.GetBool("external_storage_redis_tls_enabled"), "Use TLS on the connection if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisTLSCA, "external-storage-redis-tls-ca",
		viper.GetString("external_storage_redis_tls_ca"), "TLS CA if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisTLSCert, "external-storage-redis-tls-cert",
		viper.GetString("external_storage_redis_tls_cert"), "TLS certificate if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisTLSKey, "external-storage-redis-tls-key",
		viper.GetString("external_storage_redis_tls_key"), "TLS key if using external storage with Redis",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageRedisTLSServerName, "external-storage-redis-tls-server-name",
		viper.GetString("external_storage_redis_tls_server_name"), "TLS server name if using external storage with Redis",
	)

	cmd.Flags().BoolVar(
		&opts.ExternalStorageRedisTLSInsecureSkipVerify, "external-storage-redis-tls-insecure-skip-verify",
		viper.GetBool("external_storage_redis_tls_insecure_skip_verify"), "TLS skip verify if using external storage with Redis",
	)
}

func registerS3ExternalStorageFlags(cmd *cobra.Command, opts *runOptions) {
	cmd.Flags().StringVar(
		&opts.ExternalStorageS3Bucket, "external-storage-s3-bucket",
		viper.GetString("external_storage_s3_bucket"), "Name of the bucket if using external storage with S3",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageS3Region, "external-storage-s3-region",
		viper.GetString("external_storage_s3_region"), "Region if using external storage with S3",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageS3DriverName, "external-storage-s3-driver-name",
		viper.GetString("external_storage_s3_driver_name"), "Driver name if using external storage with S3",
	)

	cmd.Flags().IntVar(
		&opts.ExternalStorageS3MaxPayloadSize, "external-storage-s3-max-payload-size",
		viper.GetInt("external_storage_s3_max_payload_size"), "Maximum payload size if using external storage with S3",
	)

	cmd.Flags().StringVar(
		&opts.ExternalStorageS3Endpoint, "external-storage-s3-endpoint",
		viper.GetString("external_storage_s3_endpoint"), "Endpoint if using external storage with S3",
	)

	cmd.Flags().BoolVar(
		&opts.ExternalStorageS3UsePathStyle, "external-storage-s3-use-path-style",
		viper.GetBool("external_storage_s3_use_path_style"), "Use path style if using external storage with S3",
	)

	// Also support the idiomatic AWS key for access credentials
	accessKeyID := "external_storage_s3_access_key_id"
	_ = viper.BindEnv(append([]string{accessKeyID}, []string{strings.ToUpper(accessKeyID), "AWS_ACCESS_KEY_ID"}...)...)
	cmd.Flags().StringVar(
		&opts.ExternalStorageS3AccessKeyID, "external-storage-s3-access-key-id",
		viper.GetString(accessKeyID), "Access key ID if using external storage with S3",
	)
	gh.HideCommandOutput(cmd, "external-storage-s3-access-key-id")

	secretAccessKey := "external_storage_s3_secret_access_key"
	_ = viper.BindEnv(append([]string{secretAccessKey}, []string{strings.ToUpper(secretAccessKey), "AWS_SECRET_ACCESS_KEY"}...)...)
	cmd.Flags().StringVar(
		&opts.ExternalStorageS3SecretAccessKey, "external-storage-s3-secret-access-key",
		viper.GetString(secretAccessKey), "Secret access key if using external storage with S3",
	)
	gh.HideCommandOutput(cmd, "external-storage-s3-secret-access-key")

	sessionToken := "external_storage_s3_session_token"
	_ = viper.BindEnv(append([]string{sessionToken}, []string{strings.ToUpper(sessionToken), "AWS_SESSION_TOKEN"}...)...)
	cmd.Flags().StringVar(
		&opts.ExternalStorageS3SessionToken, "external-storage-s3-session-token",
		viper.GetString(sessionToken), "Session token if using external storage with S3",
	)
	gh.HideCommandOutput(cmd, "external-storage-s3-session-token")
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
	registerExternalStorageFlags(cmd, opts)

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

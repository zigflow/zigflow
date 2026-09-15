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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/telemetry"
)

func TestNewRunCmd_Flags(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	assert.NotNil(t, cmd.Flags().Lookup("file"))
	assert.NotNil(t, cmd.Flags().Lookup("validate"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-address"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-namespace"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-server-name"))
	assert.NotNil(t, cmd.Flags().Lookup("codec-endpoint"))
	assert.NotNil(t, cmd.Flags().Lookup("codec-headers"))
	assert.NotNil(t, cmd.Flags().Lookup("convert-data"))
	assert.NotNil(t, cmd.Flags().Lookup("convert-failure-data"))
	assert.NotNil(t, cmd.Flags().Lookup("converter-key-path"))
	assert.NotNil(t, cmd.Flags().Lookup("cloudevents-config"))
	assert.NotNil(t, cmd.Flags().Lookup("env-prefix"))
	assert.NotNil(t, cmd.Flags().Lookup("health-listen-address"))
	assert.NotNil(t, cmd.Flags().Lookup("metrics-listen-address"))
	assert.NotNil(t, cmd.Flags().Lookup("dir"))
	assert.NotNil(t, cmd.Flags().Lookup("glob"))
	assert.NotNil(t, cmd.Flags().Lookup("max-concurrent-activity-execution-size"))
	assert.NotNil(t, cmd.Flags().Lookup("max-concurrent-workflow-task-execution-size"))
	assert.NotNil(t, cmd.Flags().Lookup("task-queue-activities-per-second"))
	assert.NotNil(t, cmd.Flags().Lookup("enable-versioning"))
	assert.NotNil(t, cmd.Flags().Lookup("default-versioning-type"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-worker-build-id"))
	assert.NotNil(t, cmd.Flags().Lookup("temporal-deployment-name"))
	assert.NotNil(t, cmd.Flags().Lookup("container-runtime"))
	assert.NotNil(t, cmd.Flags().Lookup("container-runtime-namespace"))
	assert.NotNil(t, cmd.Flags().Lookup("container-runtime-service-account"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-payload-size-threshold"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-s3-bucket"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-s3-region"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-s3-driver-name"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-s3-max-payload-size"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-s3-endpoint"))
	assert.NotNil(t, cmd.Flags().Lookup("external-storage-s3-use-path-style"))
	assert.NotNil(t, cmd.Flags().Lookup(testFlagS3AccessKeyID))
	assert.NotNil(t, cmd.Flags().Lookup(testFlagS3SecretAccessKey))
	assert.NotNil(t, cmd.Flags().Lookup(testFlagS3SessionToken))
}

// ---- --temporal-server-name flag ----

func TestNewRunCmd_TemporalServerNameFlag(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("temporal-server-name")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.DefValue)
}

func TestNewRunCmd_TemporalServerNameFlagBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("temporal-server-name", "my-namespace.tmprl.cloud"))

	flag := cmd.Flags().Lookup("temporal-server-name")
	assert.Equal(t, "my-namespace.tmprl.cloud", flag.Value.String())
}

func TestNewRunCmd_TemporalServerNameFlagDefaultIsEmpty(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("temporal-server-name")
	require.NotNil(t, flag)
	// When omitted the flag is empty so existing connection behaviour is unchanged.
	assert.Equal(t, "", flag.Value.String())
}

// ---- --convert-failure-data flag ----

func TestNewRunCmd_ConvertFailureDataFlagDefaultsToTrue(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("convert-failure-data")
	require.NotNil(t, flag)
	// Default is true so existing behaviour is preserved: failure payloads pass
	// through the data converter unless explicitly opted out.
	assert.Equal(t, "true", flag.DefValue)
	assert.Equal(t, "true", flag.Value.String())
}

func TestNewRunCmd_ConvertFailureDataFlagBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("convert-failure-data", "false"))
	assert.Equal(t, "false", cmd.Flags().Lookup("convert-failure-data").Value.String())
}

// ---- --watch flags ----

func TestNewRunCmd_WatchFlags(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	watchFlag := cmd.Flags().Lookup("watch")
	require.NotNil(t, watchFlag)
	assert.Equal(t, "false", watchFlag.DefValue)

	debounceFlag := cmd.Flags().Lookup("watch-debounce")
	require.NotNil(t, debounceFlag)
	assert.Equal(t, "300ms", debounceFlag.DefValue)
}

func TestNewRunCmd_WatchFlagsBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("watch", "true"))
	require.NoError(t, cmd.Flags().Set("watch-debounce", "500ms"))

	watchFlag := cmd.Flags().Lookup("watch")
	assert.Equal(t, "true", watchFlag.Value.String())

	debounceFlag := cmd.Flags().Lookup("watch-debounce")
	assert.Equal(t, "500ms", debounceFlag.Value.String())
}

// ---- worker tuning flags ----

func TestNewRunCmd_WorkerTuningFlags(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	actFlag := cmd.Flags().Lookup("max-concurrent-activity-execution-size")
	require.NotNil(t, actFlag)
	assert.Equal(t, "0", actFlag.DefValue)

	wfFlag := cmd.Flags().Lookup("max-concurrent-workflow-task-execution-size")
	require.NotNil(t, wfFlag)
	assert.Equal(t, "0", wfFlag.DefValue)

	tqFlag := cmd.Flags().Lookup("task-queue-activities-per-second")
	require.NotNil(t, tqFlag)
	assert.Equal(t, "0", tqFlag.DefValue)
}

func TestNewRunCmd_WorkerTuningFlagsBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("max-concurrent-activity-execution-size", "10"))
	require.NoError(t, cmd.Flags().Set("max-concurrent-workflow-task-execution-size", "5"))
	require.NoError(t, cmd.Flags().Set("task-queue-activities-per-second", "2.5"))

	assert.Equal(t, "10", cmd.Flags().Lookup("max-concurrent-activity-execution-size").Value.String())
	assert.Equal(t, "5", cmd.Flags().Lookup("max-concurrent-workflow-task-execution-size").Value.String())
	assert.Equal(t, "2.5", cmd.Flags().Lookup("task-queue-activities-per-second").Value.String())
}

// ---- container runtime flags ----

func TestNewRunCmd_ContainerRuntimeFlagDefaultsToDocker(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("container-runtime")
	require.NotNil(t, flag)
	assert.Equal(t, "docker", flag.DefValue, "container-runtime must default to docker")
	assert.Equal(t, "docker", flag.Value.String())
}

func TestNewRunCmd_ContainerRuntimeFlagAcceptsKubernetes(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("container-runtime", "kubernetes"))
	assert.Equal(t, "kubernetes", cmd.Flags().Lookup("container-runtime").Value.String())
}

func TestNewRunCmd_ContainerRuntimeFlagAcceptsDocker(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("container-runtime", "docker"))
	assert.Equal(t, "docker", cmd.Flags().Lookup("container-runtime").Value.String())
}

func TestNewRunCmd_ContainerRuntimeNamespaceFlagDefaultIsEmpty(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("container-runtime-namespace")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.DefValue)
}

func TestNewRunCmd_ContainerRuntimeNamespaceFlagBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("container-runtime-namespace", "my-namespace"))
	assert.Equal(t, "my-namespace", cmd.Flags().Lookup("container-runtime-namespace").Value.String())
}

func TestNewRunCmd_ContainerRuntimeServiceAccountFlagDefaultIsEmpty(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	flag := cmd.Flags().Lookup("container-runtime-service-account")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.DefValue)
}

func TestNewRunCmd_ContainerRuntimeServiceAccountFlagBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("container-runtime-service-account", "workflow-sa"))
	assert.Equal(t, "workflow-sa", cmd.Flags().Lookup("container-runtime-service-account").Value.String())
}

// ---- versioning flags ----

func TestNewRunCmd_VersioningFlagDefaults(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	enableFlag := cmd.Flags().Lookup("enable-versioning")
	require.NotNil(t, enableFlag)
	assert.Equal(t, "false", enableFlag.DefValue)

	typeFlag := cmd.Flags().Lookup("default-versioning-type")
	require.NotNil(t, typeFlag)
	assert.Equal(t, string(versioningBehaviourAutoUpgrade), typeFlag.DefValue)

	buildIDFlag := cmd.Flags().Lookup("temporal-worker-build-id")
	require.NotNil(t, buildIDFlag)
	assert.Equal(t, "", buildIDFlag.DefValue)

	deployNameFlag := cmd.Flags().Lookup("temporal-deployment-name")
	require.NotNil(t, deployNameFlag)
	assert.Equal(t, "", deployNameFlag.DefValue)
}

func TestNewRunCmd_VersioningFlagsBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	require.NoError(t, cmd.Flags().Set("enable-versioning", "true"))
	require.NoError(t, cmd.Flags().Set("default-versioning-type", "pinned"))
	require.NoError(t, cmd.Flags().Set("temporal-worker-build-id", "my-build-id"))
	require.NoError(t, cmd.Flags().Set("temporal-deployment-name", "my-deploy"))

	assert.Equal(t, "true", cmd.Flags().Lookup("enable-versioning").Value.String())
	assert.Equal(t, "pinned", cmd.Flags().Lookup("default-versioning-type").Value.String())
	assert.Equal(t, "my-build-id", cmd.Flags().Lookup("temporal-worker-build-id").Value.String())
	assert.Equal(t, "my-deploy", cmd.Flags().Lookup("temporal-deployment-name").Value.String())
}

// ---- external storage flags ----

func TestNewRunCmd_ExternalStorageFlagDefaults(t *testing.T) {
	// Clear the AWS fallback variables so the defaults are not inherited from
	// the ambient environment of whoever runs the suite.
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")

	cmd := New(func() *telemetry.Telemetry { return nil })

	tests := []struct {
		flag     string
		defValue string
	}{
		// Empty means external storage is off, so existing behaviour is
		// preserved unless it is explicitly opted into.
		{flag: "external-storage", defValue: ""},
		{flag: "external-storage-payload-size-threshold", defValue: "0"},
		{flag: "external-storage-s3-bucket", defValue: ""},
		{flag: "external-storage-s3-region", defValue: ""},
		{flag: "external-storage-s3-driver-name", defValue: ""},
		{flag: "external-storage-s3-max-payload-size", defValue: "0"},
		{flag: "external-storage-s3-endpoint", defValue: ""},
		{flag: "external-storage-s3-use-path-style", defValue: "false"},
		{flag: testFlagS3AccessKeyID, defValue: ""},
		{flag: testFlagS3SecretAccessKey, defValue: ""},
		{flag: testFlagS3SessionToken, defValue: ""},
	}

	for _, test := range tests {
		t.Run(test.flag, func(t *testing.T) {
			flag := cmd.Flags().Lookup(test.flag)
			require.NotNil(t, flag)
			assert.Equal(t, test.defValue, flag.DefValue)
			assert.Equal(t, test.defValue, flag.Value.String())
		})
	}
}

func TestNewRunCmd_ExternalStorageFlagsBoundToOpts(t *testing.T) {
	cmd := New(func() *telemetry.Telemetry { return nil })

	values := map[string]string{
		"external-storage":                        testExternalStorageS3,
		"external-storage-payload-size-threshold": "1",
		"external-storage-s3-bucket":              "zigflow",
		"external-storage-s3-region":              testAWSRegion,
		"external-storage-s3-driver-name":         "my-driver",
		"external-storage-s3-max-payload-size":    "2048",
		"external-storage-s3-endpoint":            "http://s3:9000",
		"external-storage-s3-use-path-style":      "true",
		testFlagS3AccessKeyID:                     "access-key",
		testFlagS3SecretAccessKey:                 "secret-key",
		testFlagS3SessionToken:                    "session-token",
	}

	for flag, value := range values {
		require.NoError(t, cmd.Flags().Set(flag, value))
	}

	for flag, value := range values {
		assert.Equal(t, value, cmd.Flags().Lookup(flag).Value.String(), flag)
	}
}

// The AWS_* variables are accepted as aliases so a worker can pick up the
// credentials it would already have in its environment.
func TestNewRunCmd_ExternalStorageS3CredentialsFallBackToAWSEnvvars(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "aws-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "aws-session-token")

	cmd := New(func() *telemetry.Telemetry { return nil })

	assert.Equal(t, "aws-access-key", cmd.Flags().Lookup(testFlagS3AccessKeyID).Value.String())
	assert.Equal(t, "aws-secret-key", cmd.Flags().Lookup(testFlagS3SecretAccessKey).Value.String())
	assert.Equal(t, "aws-session-token", cmd.Flags().Lookup(testFlagS3SessionToken).Value.String())
}

// The prefixed variables take precedence over the AWS_* aliases.
func TestNewRunCmd_ExternalStorageS3CredentialsPreferPrefixedEnvvars(t *testing.T) {
	t.Setenv("EXTERNAL_STORAGE_S3_ACCESS_KEY_ID", "zigflow-access-key")
	t.Setenv("AWS_ACCESS_KEY_ID", "aws-access-key")

	cmd := New(func() *telemetry.Telemetry { return nil })

	assert.Equal(t, "zigflow-access-key", cmd.Flags().Lookup(testFlagS3AccessKeyID).Value.String())
}

// Credentials resolved from the environment must not be echoed back in the
// command's help output.
func TestNewRunCmd_ExternalStorageS3CredentialDefaultsAreMasked(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "aws-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "aws-session-token")

	cmd := New(func() *telemetry.Telemetry { return nil })

	for _, flag := range []string{
		testFlagS3AccessKeyID,
		testFlagS3SecretAccessKey,
		testFlagS3SessionToken,
	} {
		t.Run(flag, func(t *testing.T) {
			assert.Equal(t, "***", cmd.Flags().Lookup(flag).DefValue)
		})
	}

	assert.NotContains(t, cmd.UsageString(), "aws-access-key")
	assert.NotContains(t, cmd.UsageString(), "aws-secret-key")
	assert.NotContains(t, cmd.UsageString(), "aws-session-token")
}

func TestNewRunCmd_ExternalStorageS3CredentialDefaultsAreNotMaskedWhenUnset(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")

	cmd := New(func() *telemetry.Telemetry { return nil })

	for _, flag := range []string{
		testFlagS3AccessKeyID,
		testFlagS3SecretAccessKey,
		testFlagS3SessionToken,
	} {
		t.Run(flag, func(t *testing.T) {
			assert.Equal(t, "", cmd.Flags().Lookup(flag).DefValue)
		})
	}
}

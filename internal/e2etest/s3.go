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

package e2etest

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// s3Image is the S3-compatible object store used by the external storage
// examples. It matches the image those examples run under Docker Compose, so
// the test exercises the same server the documented setup uses.
const s3Image = "rustfs/rustfs:latest"

// s3Port is the port the object store listens on inside the container.
const s3Port = "9000/tcp"

// Credentials the object store is started with. They are fixed rather than
// generated because the point of the test is the wiring, not the secrets.
const (
	s3AccessKeyID     = "test"
	s3SecretAccessKey = "test"
	// s3Region is required by the AWS SDK even though the server ignores it.
	s3Region = "us-east-1"
)

// S3 is a running S3-compatible object store managed by Testcontainers,
// with one bucket already created.
type S3 struct {
	testcontainers.Container

	// Endpoint is the http://host:port address the test host and a host-side
	// worker both reach the store on.
	Endpoint string
	// Bucket is the name of the bucket created at startup.
	Bucket string

	client *s3.Client
}

// StartS3 starts an S3-compatible object store and creates bucket inside it.
// The container is terminated when the test finishes.
//
// The worker under test runs as a host subprocess rather than a container, so
// it reaches the store on the same mapped host port this helper uses.
func StartS3(ctx context.Context, t *testing.T, bucket string) *S3 {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        s3Image,
		ExposedPorts: []string{s3Port},
		Env: map[string]string{
			"RUSTFS_ACCESS_KEY": s3AccessKeyID,
			"RUSTFS_SECRET_KEY": s3SecretAccessKey,
		},
		// An unauthenticated request is rejected rather than refused once the
		// server is serving, so any HTTP status means it is ready.
		WaitingFor: wait.ForHTTP("/").
			WithPort(s3Port).
			WithStatusCodeMatcher(func(int) bool { return true }).
			WithStartupTimeout(90 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start S3 container")

	t.Cleanup(func() {
		// Use a background context so cleanup still runs if the test context
		// has been cancelled.
		_ = container.Terminate(context.WithoutCancel(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err, "S3 container host")

	port, err := container.MappedPort(ctx, s3Port)
	require.NoError(t, err, "S3 mapped port")

	store := &S3{
		Container: container,
		Endpoint:  "http://" + net.JoinHostPort(host, port.Port()),
		Bucket:    bucket,
	}
	store.client = newS3Client(ctx, t, store.Endpoint)

	_, err = store.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	require.NoErrorf(t, err, "create bucket %q", bucket)

	return store
}

// newS3Client builds an SDK client addressing endpoint in path style, which is
// what an S3-compatible server that is not S3 itself requires.
func newS3Client(ctx context.Context, t *testing.T, endpoint string) *s3.Client {
	t.Helper()

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3AccessKeyID, s3SecretAccessKey, ""),
		),
	)
	require.NoError(t, err, "load AWS config")

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
}

// WorkerEnv returns the environment entries that configure a Zigflow worker to
// offload payloads to this store. payloadSizeThreshold is the size in bytes
// above which a payload is offloaded; pass 1 to offload everything.
func (s *S3) WorkerEnv(payloadSizeThreshold int) []string {
	return []string{
		"EXTERNAL_STORAGE=s3",
		fmt.Sprintf("EXTERNAL_STORAGE_PAYLOAD_SIZE_THRESHOLD=%d", payloadSizeThreshold),
		"EXTERNAL_STORAGE_S3_BUCKET=" + s.Bucket,
		"EXTERNAL_STORAGE_S3_REGION=" + s3Region,
		"EXTERNAL_STORAGE_S3_ENDPOINT=" + s.Endpoint,
		"EXTERNAL_STORAGE_S3_USE_PATH_STYLE=true",
		"AWS_ACCESS_KEY_ID=" + s3AccessKeyID,
		"AWS_SECRET_ACCESS_KEY=" + s3SecretAccessKey,
	}
}

// Region returns the region the store is addressed with.
func (s *S3) Region() string { return s3Region }

// Credentials returns the access key and secret the store was started with.
func (s *S3) Credentials() (accessKeyID, secretAccessKey string) {
	return s3AccessKeyID, s3SecretAccessKey
}

// ObjectKeys returns the keys of every object currently in the bucket.
func (s *S3) ObjectKeys(ctx context.Context, t *testing.T) []string {
	t.Helper()

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.Bucket),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		require.NoError(t, err, "list bucket objects")

		for _, obj := range page.Contents {
			keys = append(keys, aws.ToString(obj.Key))
		}
	}

	return keys
}

//go:build e2e

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

package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporalhelpers "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
)

// bucketName is the bucket the worker offloads payloads into. It matches the
// bucket name used by the example's compose stack.
const bucketName = "zigflow"

// payloadSizeThreshold offloads every payload rather than only large ones, so
// the example demonstrates external storage without needing bulky test data.
const payloadSizeThreshold = 1

// result mirrors the hello-world workflow output:
//
//	{"data": {"message": "Hello from Ziggy"}}
type result struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// TestHelloWorldExternalStorageE2E runs the hello-world-external-storage
// example end to end against a Temporal dev server and an S3-compatible object
// store that the test owns through Testcontainers. The example provides its own
// infrastructure rather than relying on the shared compose stack.
//
// It asserts both halves of the contract: the workflow result is unchanged by
// offloading, and the payloads really did leave the Temporal history for the
// bucket.
func TestHelloWorldExternalStorageE2E(t *testing.T) {
	ctx := t.Context()

	store := e2etest.StartS3(ctx, t, bucketName)
	require.Empty(t, store.ObjectKeys(ctx, t), "bucket should start empty")

	temporal := e2etest.StartTemporal(ctx, t)

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	e2etest.StartWorkerWithEnv(ctx, t, store.WorkerEnv(payloadSizeThreshold), temporal.Address, workflowFile)

	c := dialWithExternalStorage(ctx, t, temporal.Address, store)
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "hello-world")
	require.NoError(t, err, "execute workflow")

	var got result
	require.NoError(t, we.Get(runCtx, &got), "get workflow result")

	// Offloading is transparent: the caller sees the same result it would
	// without external storage configured.
	assert.Equal(t, "Hello from Ziggy", got.Data.Message)

	// The payloads must actually be in the bucket, otherwise the assertion
	// above would still pass with external storage silently disabled.
	assert.NotEmpty(t, store.ObjectKeys(ctx, t), "expected payloads to be offloaded to the bucket")
}

// dialWithExternalStorage builds a Temporal client configured with the same S3
// driver as the worker. The client needs it to resolve the storage references
// the worker writes back in place of the workflow result.
func dialWithExternalStorage(
	ctx context.Context, t *testing.T, temporalAddress string, store *e2etest.S3,
) client.Client {
	t.Helper()

	accessKeyID, secretAccessKey := store.Credentials()

	c, err := temporalhelpers.NewConnection(
		temporalhelpers.WithHostPort(temporalAddress),
		temporalhelpers.WithExternalStorageFactory(temporalhelpers.ExternalConfig{
			PayloadSizeThreshold: payloadSizeThreshold,
			Factory: temporalhelpers.ExternalConfigS3Factory(ctx, &temporalhelpers.S3Config{
				Bucket:          store.Bucket,
				Region:          store.Region(),
				Endpoint:        store.Endpoint,
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				UsePathStyle:    true,
			}),
		}),
	)
	require.NoError(t, err, "dial Temporal")

	return c
}

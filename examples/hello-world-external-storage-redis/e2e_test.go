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

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporalhelpers "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/internal/e2etest"
	"github.com/zigflow/zigflow/pkg/externalstorage"
	"go.temporal.io/sdk/client"
)

// payloadSizeThreshold offloads every payload rather than only large ones, so
// the example demonstrates external storage without needing bulky test data.
// It matches the threshold the example's compose stack and trigger both set.
const payloadSizeThreshold = 1

// result mirrors the hello-world workflow output:
//
//	{"data": {"message": "Hello from Ziggy"}}
type result struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// TestHelloWorldExternalStorageRedisE2E runs the
// hello-world-external-storage-redis example end to end against a Temporal dev
// server and a Redis server that the test owns through Testcontainers. The
// example provides its own infrastructure rather than relying on the shared
// compose stack.
//
// It asserts both halves of the contract: the workflow result is unchanged by
// offloading, and the payloads really did leave the Temporal history for Redis.
func TestHelloWorldExternalStorageRedisE2E(t *testing.T) {
	ctx := t.Context()

	store := e2etest.StartRedis(ctx, t)
	require.Empty(t, store.ClaimKeys(ctx, t), "Redis should start empty")

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

	// The payloads must actually be in Redis, otherwise the assertion above
	// would still pass with external storage silently disabled.
	assert.NotEmpty(t, store.ClaimKeys(ctx, t), "expected payloads to be offloaded to Redis")
}

// dialWithExternalStorage builds a Temporal client configured with the same
// Redis driver as the worker. The client needs it to resolve the storage
// references the worker writes back in place of the workflow result.
//
// As in the example's trigger, only the server address is set: the driver name,
// key prefix, database and TTL are all left at their defaults.
func dialWithExternalStorage(
	ctx context.Context, t *testing.T, temporalAddress string, store *e2etest.Redis,
) client.Client {
	t.Helper()

	c, err := temporalhelpers.NewConnection(
		temporalhelpers.WithHostPort(temporalAddress),
		temporalhelpers.WithExternalStorageFactory(temporalhelpers.ExternalConfig{
			PayloadSizeThreshold: payloadSizeThreshold,
			Factory: externalstorage.ExternalConfigRedisFactory(ctx, &externalstorage.RedisConfig{
				Options: &goredis.Options{
					Addr: store.Address,
				},
			}),
		}),
	)
	require.NoError(t, err, "dial Temporal")

	return c
}

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

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// redisImage is the Redis server used by the external storage example. It
// matches the image that example runs under Docker Compose, so the test
// exercises the same server the documented setup uses.
const redisImage = "redis:alpine"

// redisPort is the port Redis listens on inside the container.
const redisPort = "6379/tcp"

// redisClaimKeyPrefix is the prefix the Redis storage driver writes claim keys
// under. It is the driver default, which the example does not override.
const redisClaimKeyPrefix = "zigflow:claim"

// Redis is a running Redis server managed by Testcontainers, started with no
// authentication and an empty keyspace.
type Redis struct {
	testcontainers.Container

	// Address is the host:port address the test host and a host-side worker
	// both reach the server on.
	Address string

	client *goredis.Client
}

// StartRedis starts a Redis server. The container is terminated when the test
// finishes.
//
// The worker under test runs as a host subprocess rather than a container, so
// it reaches the server on the same mapped host port this helper uses.
func StartRedis(ctx context.Context, t *testing.T) *Redis {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        redisImage,
		ExposedPorts: []string{redisPort},
		WaitingFor:   wait.ForListeningPort(redisPort).WithStartupTimeout(90 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start Redis container")

	t.Cleanup(func() {
		// Use a background context so cleanup still runs if the test context
		// has been cancelled.
		_ = container.Terminate(context.WithoutCancel(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err, "Redis container host")

	port, err := container.MappedPort(ctx, redisPort)
	require.NoError(t, err, "Redis mapped port")

	store := &Redis{
		Container: container,
		Address:   net.JoinHostPort(host, port.Port()),
	}
	store.client = goredis.NewClient(&goredis.Options{Addr: store.Address})

	t.Cleanup(func() { _ = store.client.Close() })

	require.NoError(t, store.client.Ping(ctx).Err(), "ping Redis")

	return store
}

// WorkerEnv returns the environment entries that configure a Zigflow worker to
// offload payloads to this server. payloadSizeThreshold is the size in bytes
// above which a payload is offloaded; pass 1 to offload everything.
func (r *Redis) WorkerEnv(payloadSizeThreshold int) []string {
	return []string{
		"EXTERNAL_STORAGE=redis",
		fmt.Sprintf("EXTERNAL_STORAGE_PAYLOAD_SIZE_THRESHOLD=%d", payloadSizeThreshold),
		"EXTERNAL_STORAGE_REDIS_ADDRESS=" + r.Address,
	}
}

// ClaimKeys returns every claim key currently held by the server, that is the
// keys matching the storage driver's claim prefix.
func (r *Redis) ClaimKeys(ctx context.Context, t *testing.T) []string {
	t.Helper()

	var keys []string
	iter := r.client.Scan(ctx, 0, redisClaimKeyPrefix+":*", 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	require.NoError(t, iter.Err(), "scan Redis claim keys")

	return keys
}

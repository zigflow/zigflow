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

package externalstorage

import (
	"context"
	"strings"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	"go.temporal.io/sdk/converter"
)

const (
	testBucket    = "zigflow"
	testRegion    = "us-east-1"
	testAccessKey = "test"
	testSecretKey = "test"

	// testRedisDriverName and testRedisKeyPrefix are non-default values, used
	// to prove the configured value reaches the driver rather than the
	// driver's own default being applied regardless.
	testRedisDriverName = "my-redis-driver"
	testRedisKeyPrefix  = "tenant-a:payloads"

	// defaultRedisDriverName is the name the Redis driver falls back to when
	// none is configured.
	defaultRedisDriverName = "redis.driver"

	// redisClaimDataKey is the ClaimData entry the Redis driver records the
	// Redis key under.
	redisClaimDataKey = "key"

	testRedisUsername = "ziggy"
	testRedisPassword = "correct-horse"
)

// stubDriverSelector is a do-nothing StorageDriverSelector used to prove the
// selector is carried through to the ExternalConfig untouched.
type stubDriverSelector struct{}

func (stubDriverSelector) SelectDriver(
	_ converter.StorageDriverStoreContext,
	_ *commonpb.Payload,
) (converter.StorageDriver, error) {
	return nil, nil
}

// validS3Config returns an S3Config that the helper factory accepts. Static
// credentials are supplied so building the driver never consults the ambient
// AWS credential chain.
func validS3Config() *temporal.S3Config {
	return &temporal.S3Config{
		Bucket:          testBucket,
		Region:          testRegion,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
	}
}

// ---- ParseStorageType ----

func TestParseStorageType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StorageType
		wantErr bool
	}{
		{
			name:  "empty string is the none type",
			input: "",
			want:  StorageTypeNone,
		},
		{
			name:  "s3 is recognised",
			input: "s3",
			want:  StorageTypeS3,
		},
		{
			name:  "redis is recognised",
			input: "redis",
			want:  StorageTypeRedis,
		},
		{
			name:    "parsing is case sensitive",
			input:   "S3",
			wantErr: true,
		},
		{
			name:    "unknown type is rejected",
			input:   "gcs",
			wantErr: true,
		},
		{
			name:    "surrounding whitespace is not trimmed",
			input:   " s3",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseStorageType(test.input)
			if test.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "invalid external storage type")
				// The rejected value is quoted back so the message is actionable.
				assert.ErrorContains(t, err, `"`+test.input+`"`)
				// Every accepted type is listed, so the message says what to
				// use instead of the rejected value.
				assert.ErrorContains(t, err, `"redis"`)
				assert.ErrorContains(t, err, `"s3"`)
				assert.Equal(t, StorageType(""), got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

// ---- New ----

func TestNewStorageTypeNone(t *testing.T) {
	cfg, err := New(t.Context(), StorageTypeNone, &Config{PayloadSizeThreshold: 1024})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Factory)
	assert.Equal(t, 1024, cfg.PayloadSizeThreshold)

	// The noop factory must succeed and yield no drivers, so external storage
	// is inert rather than broken when it has not been configured.
	drivers, err := cfg.Factory()
	require.NoError(t, err)
	assert.Empty(t, drivers)
}

func TestNewNilConfig(t *testing.T) {
	cfg, err := New(t.Context(), StorageTypeNone, nil)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Factory)
	assert.Equal(t, 0, cfg.PayloadSizeThreshold)
	assert.Nil(t, cfg.StorageDriverSelector)

	drivers, err := cfg.Factory()
	require.NoError(t, err)
	assert.Empty(t, drivers)
}

func TestNewStorageTypeS3(t *testing.T) {
	cfg, err := New(t.Context(), StorageTypeS3, &Config{
		PayloadSizeThreshold: 1,
		S3Confg:              validS3Config(),
	})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Factory)
	assert.Equal(t, 1, cfg.PayloadSizeThreshold)

	drivers, err := cfg.Factory()
	require.NoError(t, err)
	assert.Len(t, drivers, 1, "expected exactly one S3 storage driver")
	assert.NotNil(t, drivers[0])
}

func TestNewStorageTypeS3MissingS3Config(t *testing.T) {
	cfg, err := New(t.Context(), StorageTypeS3, &Config{})
	// Building the config must not talk to the backend, so the missing S3
	// config is only reported when the factory is invoked.
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Factory)

	drivers, err := cfg.Factory()
	require.Error(t, err)
	assert.ErrorContains(t, err, "config missing")
	assert.Nil(t, drivers)
}

func TestNewInvalidStorageType(t *testing.T) {
	cfg, err := New(t.Context(), StorageType("gcs"), &Config{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid storage type: gcs")
	assert.Nil(t, cfg)
}

func TestNewPropagatesDriverSelector(t *testing.T) {
	selector := stubDriverSelector{}

	cfg, err := New(t.Context(), StorageTypeNone, &Config{
		PayloadSizeThreshold:  256,
		StorageDriverSelector: selector,
	})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 256, cfg.PayloadSizeThreshold)
	assert.Equal(t, selector, cfg.StorageDriverSelector)
}

func TestNewDoesNotInvokeFactoryEagerly(t *testing.T) {
	// A nil S3Confg or RedisConfig makes the factory fail when called. New
	// returning no error proves construction is lazy and free of side effects,
	// so nothing dials Redis or S3 until the connection is built.
	for _, storageType := range []StorageType{StorageTypeNone, StorageTypeRedis, StorageTypeS3} {
		t.Run(string(storageType), func(t *testing.T) {
			cfg, err := New(t.Context(), storageType, &Config{S3Confg: nil, RedisConfig: nil})
			require.NoError(t, err)
			require.NotNil(t, cfg.Factory)
		})
	}
}

// ---- Redis ----

// deadRedisAddress returns the address of a Redis server that has been shut
// down. Connecting to it is refused immediately, so a connection failure can
// be provoked without waiting for a timeout.
func deadRedisAddress(t *testing.T) string {
	t.Helper()

	server := miniredis.RunT(t)
	addr := server.Addr()
	server.Close()

	return addr
}

// storePayload writes a single payload through driver and returns the claim
// data the driver recorded for it. Going through the driver keeps the
// assertions on observable Redis state rather than on the driver's fields.
func storePayload(t *testing.T, driver converter.StorageDriver) map[string]string {
	t.Helper()

	claims, err := driver.Store(
		converter.StorageDriverStoreContext{Context: t.Context()},
		[]*commonpb.Payload{{Data: []byte("payload")}},
	)
	require.NoError(t, err)
	require.Len(t, claims, 1)

	return claims[0].ClaimData
}

// redisDriver builds the Redis factory for cfg and returns the single driver
// it produces.
func redisDriver(t *testing.T, cfg *RedisConfig) converter.StorageDriver {
	t.Helper()

	drivers, err := ExternalConfigRedisFactory(t.Context(), cfg)()
	require.NoError(t, err)
	require.Len(t, drivers, 1, "expected exactly one Redis storage driver")

	return drivers[0]
}

func TestNewStorageTypeRedis(t *testing.T) {
	server := miniredis.RunT(t)

	cfg, err := New(t.Context(), StorageTypeRedis, &Config{
		PayloadSizeThreshold: 1,
		RedisConfig: &RedisConfig{
			Options: &goredis.Options{Addr: server.Addr()},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Factory)
	assert.Equal(t, 1, cfg.PayloadSizeThreshold)

	drivers, err := cfg.Factory()
	require.NoError(t, err)
	require.Len(t, drivers, 1, "expected exactly one Redis storage driver")
	// Left unset, the driver's own default name applies.
	assert.Equal(t, defaultRedisDriverName, drivers[0].Name())
}

func TestNewStorageTypeRedisMissingRedisConfig(t *testing.T) {
	cfg, err := New(t.Context(), StorageTypeRedis, &Config{})
	// Building the config must not talk to the backend, so the missing Redis
	// config is only reported when the factory is invoked.
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Factory)

	drivers, err := cfg.Factory()
	require.Error(t, err)
	assert.ErrorContains(t, err, "config missing")
	assert.Nil(t, drivers)
}

// A configured driver name must reach the driver, otherwise a deployment that
// names its driver cannot resolve payloads stored under that name.
func TestExternalConfigRedisFactoryPropagatesDriverName(t *testing.T) {
	server := miniredis.RunT(t)

	driver := redisDriver(t, &RedisConfig{
		DriverName: testRedisDriverName,
		Options:    &goredis.Options{Addr: server.Addr()},
	})

	assert.Equal(t, testRedisDriverName, driver.Name())
}

// The key prefix and TTL must reach the driver, otherwise offloaded payloads
// land under the wrong keys or never expire.
func TestExternalConfigRedisFactoryPropagatesKeyPrefixAndTTL(t *testing.T) {
	server := miniredis.RunT(t)

	driver := redisDriver(t, &RedisConfig{
		KeyPrefix: testRedisKeyPrefix,
		Options:   &goredis.Options{Addr: server.Addr()},
		TTL:       time.Hour,
	})

	key := storePayload(t, driver)[redisClaimDataKey]
	require.NotEmpty(t, key)

	assert.True(
		t,
		strings.HasPrefix(key, testRedisKeyPrefix+":"),
		"expected key %q to use the configured prefix %q", key, testRedisKeyPrefix,
	)
	assert.Equal(t, []string{key}, server.Keys())
	assert.Equal(t, time.Hour, server.TTL(key))
}

// An unset TTL must leave the record without an expiry rather than expiring it
// immediately, so payloads outlive the workflow that wrote them.
func TestExternalConfigRedisFactoryWithoutTTLDoesNotExpire(t *testing.T) {
	server := miniredis.RunT(t)

	driver := redisDriver(t, &RedisConfig{
		Options: &goredis.Options{Addr: server.Addr()},
	})

	key := storePayload(t, driver)[redisClaimDataKey]

	assert.Equal(t, []string{key}, server.Keys())
	assert.Zero(t, server.TTL(key), "expected no expiry to be set")
}

// The Redis options are handed to the client as given, so a configured
// database is the one written to.
func TestExternalConfigRedisFactoryPropagatesDatabase(t *testing.T) {
	server := miniredis.RunT(t)

	driver := redisDriver(t, &RedisConfig{
		Options: &goredis.Options{Addr: server.Addr(), DB: 3},
	})

	key := storePayload(t, driver)[redisClaimDataKey]

	assert.Equal(t, []string{key}, server.DB(3).Keys(), "expected the payload in the configured database")
	assert.Empty(t, server.DB(0).Keys(), "expected nothing in the default database")
}

// A server that rejects the supplied credentials must fail the factory rather
// than yield a driver that cannot read or write.
func TestExternalConfigRedisFactoryPropagatesCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{
			name:     "matching credentials connect",
			username: testRedisUsername,
			password: testRedisPassword,
		},
		{
			name:     "wrong password is rejected",
			username: testRedisUsername,
			password: "wrong",
			wantErr:  true,
		},
		{
			name:    "missing credentials are rejected",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			server.RequireUserAuth(testRedisUsername, testRedisPassword)

			drivers, err := ExternalConfigRedisFactory(t.Context(), &RedisConfig{
				Options: &goredis.Options{
					Addr:     server.Addr(),
					Username: test.username,
					Password: test.password,
				},
			})()

			if test.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to connect to redis")
				assert.Nil(t, drivers)
				return
			}

			require.NoError(t, err)
			assert.Len(t, drivers, 1)
		})
	}
}

// The factory pings on construction so an unreachable server is reported when
// the worker starts rather than when the first payload is offloaded.
func TestExternalConfigRedisFactoryConnectionFailure(t *testing.T) {
	drivers, err := ExternalConfigRedisFactory(t.Context(), &RedisConfig{
		Options: &goredis.Options{Addr: deadRedisAddress(t)},
	})()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to connect to redis")
	assert.Nil(t, drivers)
}

// The context given to the factory bounds the connection check, and a config
// without options is defaulted rather than panicking.
func TestExternalConfigRedisFactoryHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	drivers, err := ExternalConfigRedisFactory(ctx, &RedisConfig{})()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to connect to redis")
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, drivers)
}

// A negative TTL is rejected by the driver, and the factory must surface that
// rather than return a driver that silently drops every payload.
func TestExternalConfigRedisFactoryNegativeTTL(t *testing.T) {
	server := miniredis.RunT(t)

	drivers, err := ExternalConfigRedisFactory(t.Context(), &RedisConfig{
		Options: &goredis.Options{Addr: server.Addr()},
		TTL:     -time.Second,
	})()

	require.Error(t, err)
	assert.ErrorContains(t, err, "TTL must not be negative")
	assert.Nil(t, drivers)
}

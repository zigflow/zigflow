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
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/helpers/externalstorage/redis"
	"go.temporal.io/sdk/converter"
)

type StorageType string

const (
	StorageTypeNone  StorageType = ""
	StorageTypeRedis StorageType = "redis"
	StorageTypeS3    StorageType = "s3"
)

var storages = map[StorageType]struct{}{
	StorageTypeNone:  {},
	StorageTypeRedis: {},
	StorageTypeS3:    {},
}

type Config struct {
	PayloadSizeThreshold  int
	StorageDriverSelector converter.StorageDriverSelector
	RedisConfig           *RedisConfig
	S3Confg               *temporal.S3Config
}

func newNoopFactory() temporal.ExternalConfigFactory {
	return func() ([]converter.StorageDriver, error) {
		return []converter.StorageDriver{}, nil
	}
}

func ParseStorageType(t string) (StorageType, error) {
	normalised := StorageType(t)

	if _, ok := storages[normalised]; ok {
		return normalised, nil
	}

	return "", fmt.Errorf(
		"invalid external storage type %q (must be %q, %q or %q)",
		t,
		StorageTypeNone,
		StorageTypeRedis,
		StorageTypeS3,
	)
}

func New(ctx context.Context, t StorageType, config *Config) (*temporal.ExternalConfig, error) {
	if config == nil {
		config = &Config{}
	}

	var factory temporal.ExternalConfigFactory
	switch t {
	case StorageTypeNone:
		factory = newNoopFactory()
	case StorageTypeRedis:
		factory = ExternalConfigRedisFactory(ctx, config.RedisConfig)
	case StorageTypeS3:
		factory = temporal.ExternalConfigS3Factory(ctx, config.S3Confg)
	default:
		return nil, fmt.Errorf("invalid storage type: %s", t)
	}

	return &temporal.ExternalConfig{
		Factory:               factory,
		PayloadSizeThreshold:  config.PayloadSizeThreshold,
		StorageDriverSelector: config.StorageDriverSelector,
	}, nil
}

func ExternalConfigRedisFactory(ctx context.Context, cfg *RedisConfig) temporal.ExternalConfigFactory {
	return func() ([]converter.StorageDriver, error) {
		if cfg == nil {
			return nil, fmt.Errorf("redisfactory: config missing as second argument")
		}

		client := goredis.NewClient(cfg.Options)

		// Manages the defer in this context
		context.AfterFunc(ctx, func() {
			_ = client.Close()
		})

		// Check the connection
		if err := client.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to redis: %w", err)
		}

		driver, err := redis.New(&redis.Options{
			Client:     client,
			DriverName: cfg.DriverName,
			KeyPrefix:  cfg.KeyPrefix,
			TTL:        cfg.TTL,
		})
		if err != nil {
			return nil, err
		}

		return []converter.StorageDriver{driver}, nil
	}
}

type RedisConfig struct {
	DriverName string
	KeyPrefix  string
	Options    *goredis.Options
	TTL        time.Duration
}

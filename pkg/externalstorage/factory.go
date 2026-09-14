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

	temporal "github.com/zigflow/helpers"
	"go.temporal.io/sdk/converter"
)

type StorageType string

const (
	StorageTypeNone StorageType = ""
	StorageTypeS3   StorageType = "s3"
)

var storages = map[StorageType]struct{}{
	StorageTypeNone: {},
	StorageTypeS3:   {},
}

type Config struct {
	PayloadSizeThreshold  int
	StorageDriverSelector converter.StorageDriverSelector
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
		"invalid external storage type %q (must be %q or %q)",
		t,
		StorageTypeNone,
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

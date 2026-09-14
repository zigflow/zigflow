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
	"testing"

	commonpb "go.temporal.io/api/common/v1"

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
	// A nil S3Confg makes the factory fail when called. New returning no error
	// proves construction is lazy and free of side effects.
	for _, storageType := range []StorageType{StorageTypeNone, StorageTypeS3} {
		t.Run(string(storageType), func(t *testing.T) {
			cfg, err := New(t.Context(), storageType, &Config{S3Confg: nil})
			require.NoError(t, err)
			require.NotNil(t, cfg.Factory)
		})
	}
}

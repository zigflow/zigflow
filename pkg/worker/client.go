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

package worker

import (
	"context"

	"github.com/rs/zerolog/log"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/codec"
	"github.com/zigflow/zigflow/pkg/ctxpropagator"
	"github.com/zigflow/zigflow/pkg/externalstorage"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// ClientConfig configures payload conversion and external storage for Zigflow workers.
// The zero value uses inline payloads with no encryption.
type ClientConfig struct {
	ConvertData                         string
	CodecEndpoint                       string
	CodecHeaders                        map[string]string
	ConvertKeyPath                      string
	DisableFailureConversion            bool
	ExternalStorage                     string
	ExternalStoragePayloadSizeThreshold int
	ExternalStorageS3Bucket             string
	ExternalStorageS3Region             string
	ExternalStorageS3DriverName         string
	ExternalStorageS3MaxPayloadSize     int
	ExternalStorageS3Endpoint           string
	ExternalStorageS3AccessKeyID        string
	ExternalStorageS3SecretAccessKey    string
	ExternalStorageS3SessionToken       string
	ExternalStorageS3UsePathStyle       bool
}

// ClientOptions builds the options shared by long-lived workers and test sessions.
// It does not connect to Temporal or start health or metrics listeners.
func ClientOptions(ctx context.Context, connection *temporal.TemporalOpts, cfg *ClientConfig) ([]temporal.Option, error) {
	if connection == nil {
		connection = &temporal.TemporalOpts{}
	}
	if cfg == nil {
		cfg = &ClientConfig{}
	}
	keyPath := cfg.ConvertKeyPath
	if keyPath == "" {
		keyPath = "keys.yaml"
	}
	codecType, err := codec.ParseCodecType(cfg.ConvertData)
	if err != nil {
		return nil, err
	}
	dataConverter, err := codec.NewDataConverter(codecType, cfg.CodecEndpoint, keyPath, cfg.CodecHeaders)
	if err != nil {
		return nil, err
	}

	storageType, err := externalstorage.ParseStorageType(cfg.ExternalStorage)
	if err != nil {
		return nil, err
	}
	externalStorage, err := externalstorage.New(ctx, storageType, &externalstorage.Config{
		PayloadSizeThreshold: cfg.ExternalStoragePayloadSizeThreshold,
		S3Confg: &temporal.S3Config{
			Bucket:          cfg.ExternalStorageS3Bucket,
			Region:          cfg.ExternalStorageS3Region,
			DriverName:      cfg.ExternalStorageS3DriverName,
			MaxPayloadSize:  cfg.ExternalStorageS3MaxPayloadSize,
			Endpoint:        cfg.ExternalStorageS3Endpoint,
			AccessKeyID:     cfg.ExternalStorageS3AccessKeyID,
			SecretAccessKey: cfg.ExternalStorageS3SecretAccessKey,
			SessionToken:    cfg.ExternalStorageS3SessionToken,
			UsePathStyle:    cfg.ExternalStorageS3UsePathStyle,
		},
	})
	if err != nil {
		return nil, err
	}

	log.Trace().Msg("Connecting to Temporal")
	connectionOpts := []temporal.Option{
		temporal.WithHostPort(connection.Address),
		temporal.WithNamespace(connection.Namespace),
		temporal.WithTLS(connection.TLSEnabled, temporal.WithTLSServerName(connection.ServerName)),
		temporal.WithAuthDetection(
			connection.APIKey,
			connection.MTLSCertPath,
			connection.MTLSKeyPath,
		),
		temporal.WithDataConverter(dataConverter),
		temporal.WithExternalStorageFactory(*externalStorage),
		func(o *client.Options) error {
			if !cfg.DisableFailureConversion {
				return temporal.WithFailureConverter(dataConverter)(o)
			}
			return nil
		},
		temporal.WithZerolog(&log.Logger),
		temporal.WithContextPropagators([]workflow.ContextPropagator{ctxpropagator.NewContextPropagator()}),
	}
	return connectionOpts, nil
}

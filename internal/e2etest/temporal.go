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
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// defaultTemporalImage is the Temporal CLI/server image used by default. It
// bundles the dev server (server start-dev) and the temporal CLI used for the
// readiness check, so a single container provides everything an example needs.
//
// The tag is intentionally unpinned: Zigflow supports the latest Temporal, and
// running against :latest lets these tests surface unexpected breakage against
// current Temporal. Use ImageEnvVar to pin a specific image for local repro.
const defaultTemporalImage = "temporalio/temporal:latest"

// ImageEnvVar names the environment variable that overrides the Temporal image.
// Set it to pin a version locally, for example:
//
//	ZIGFLOW_E2E_TEMPORAL_IMAGE=temporalio/temporal:1.7.2 go test -tags=e2e ./examples/...
const ImageEnvVar = "ZIGFLOW_E2E_TEMPORAL_IMAGE"

// temporalImage returns the Temporal image to run: the ImageEnvVar override when
// set, otherwise defaultTemporalImage.
func temporalImage() string {
	if img := os.Getenv(ImageEnvVar); img != "" {
		return img
	}
	return defaultTemporalImage
}

// temporalFrontendPort is the gRPC frontend port the Temporal client connects
// to.
const temporalFrontendPort = "7233/tcp"

// Temporal is a running Temporal dev server managed by Testcontainers.
type Temporal struct {
	testcontainers.Container

	// Address is the host:port of the Temporal frontend, suitable for passing
	// to a Temporal client or the Zigflow worker's --temporal-address flag.
	Address string
}

// StartTemporal starts a single-container Temporal dev server using
// Testcontainers and returns its frontend address. The container is terminated
// automatically when the test finishes.
//
// No shared Docker Compose stack is required: the example owns this dependency
// for the lifetime of the test.
func StartTemporal(ctx context.Context, t *testing.T) *Temporal {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        temporalImage(),
		ExposedPorts: []string{temporalFrontendPort},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
		// Mirror the readiness check used by the shared compose stack so the
		// container is only considered ready once the cluster can serve
		// requests, not merely once the port is open.
		WaitingFor: wait.ForExec([]string{"temporal", "operator", "cluster", "health"}).
			WithStartupTimeout(90 * time.Second).
			WithPollInterval(time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start Temporal container")

	t.Cleanup(func() {
		// Use a background context so cleanup still runs if the test context
		// has been cancelled.
		_ = container.Terminate(context.WithoutCancel(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err, "Temporal container host")

	port, err := container.MappedPort(ctx, temporalFrontendPort)
	require.NoError(t, err, "Temporal frontend mapped port")

	return &Temporal{
		Container: container,
		Address:   net.JoinHostPort(host, port.Port()),
	}
}

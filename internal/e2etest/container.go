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
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// goServiceImage is the minimal base image used to run a host-built static Go
// binary. It carries no toolchain: the binary is compiled on the host and
// copied in, so the container only has to execute it.
const goServiceImage = "alpine:3.20"

// GoServiceContainer is a Go program compiled on the host and run inside a
// Testcontainers container.
type GoServiceContainer struct {
	testcontainers.Container

	// Address is the host:port the test host can reach the service on.
	Address string
	// Host is the host portion of Address alone, useful for NO_PROXY so a
	// worker's HTTP(S) proxy does not also intercept calls to this service.
	Host string
	// Port is the mapped host port portion of Address alone.
	Port string
}

// StartGoServiceContainer builds the Go main package in pkgDir into a static
// Linux binary on the host, then runs that binary inside a minimal container
// with exposedPort published. It provides, under Testcontainers, a containerised
// example dependency that Docker Compose would otherwise supply (such as the
// external-calls gRPC backend), without building an image or fetching modules
// inside the container: the host toolchain and module cache perform the build,
// and the container only executes the result.
//
// exposedPort is the port the service listens on inside the container, in
// "<port>/tcp" form. args are passed to the binary. The container is terminated
// when the test finishes.
func StartGoServiceContainer(ctx context.Context, t *testing.T, pkgDir, exposedPort string, args ...string) *GoServiceContainer {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "service")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = pkgDir
	// Compile a static binary for the container's platform. The Docker daemon
	// runs the test host's architecture, so GOARCH matches runtime.GOARCH.
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+runtime.GOARCH)
	out, err := build.CombinedOutput()
	require.NoErrorf(t, err, "build Go service in %s:\n%s", pkgDir, out)

	req := testcontainers.ContainerRequest{
		Image:        goServiceImage,
		ExposedPorts: []string{exposedPort},
		Files: []testcontainers.ContainerFile{{
			HostFilePath:      binary,
			ContainerFilePath: "/service",
			FileMode:          0o755,
		}},
		Cmd:        append([]string{"/service"}, args...),
		WaitingFor: wait.ForListeningPort(exposedPort).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start Go service container")

	t.Cleanup(func() {
		// Use a background context so cleanup still runs if the test context
		// has been cancelled.
		_ = container.Terminate(context.WithoutCancel(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err, "Go service container host")

	port, err := container.MappedPort(ctx, exposedPort)
	require.NoError(t, err, "Go service mapped port")

	return &GoServiceContainer{
		Container: container,
		Address:   net.JoinHostPort(host, port.Port()),
		Host:      host,
		Port:      port.Port(),
	}
}

// LoopbackForwarder pipes TCP connections from a loopback port to a fixed
// target address.
type LoopbackForwarder struct {
	// Host is always "localhost".
	Host string
	// Port is the loopback port forwarded to the target.
	Port string
}

// StartLoopbackForwarder listens on a 127.0.0.1 port and forwards every
// connection to target (a "host:port" address). It lets a host-side worker
// reach a service through the "localhost" hostname even when that service is
// only reachable by IP, for example a Docker gateway IP under
// Docker-in-Docker. That matters because Zigflow validates a gRPC
// service.host as an RFC 1123 hostname and rejects bare IP addresses, so a
// container's gateway-IP address cannot be used there directly. The forwarder
// is closed when the test finishes.
func StartLoopbackForwarder(ctx context.Context, t *testing.T, target string) *LoopbackForwarder {
	t.Helper()

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen for loopback forwarder")

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed on cleanup
			}
			go forwardConn(ctx, conn, target)
		}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err, "loopback forwarder port")

	return &LoopbackForwarder{Host: "localhost", Port: port}
}

// forwardConn bridges a single accepted connection to a fresh dial of target,
// copying bytes in both directions until either side closes.
func forwardConn(ctx context.Context, downstream net.Conn, target string) {
	defer func() { _ = downstream.Close() }()

	var d net.Dialer
	upstream, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return
	}
	defer func() { _ = upstream.Close() }()

	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		// Closing both ends unblocks the partner copy so it can finish too.
		_ = dst.Close()
		_ = src.Close()
		done <- struct{}{}
	}
	go pipe(upstream, downstream)
	go pipe(downstream, upstream)
	<-done
}

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
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// MockResponse customises the status code returned for a route. Use it as a
// route value when an example expects a non-200 response (for example an error
// path under test); a plain value is served as 200.
type MockResponse struct {
	Status int
	Body   any
}

// HTTPMock is a local HTTPS mock that example workers reach in place of a real
// external service. It runs as an HTTP CONNECT proxy: the worker is pointed at
// it with HTTPS_PROXY, so requests to hardcoded https:// endpoints are
// intercepted and answered from fixtures without any DNS lookup or internet
// access. TLS is terminated with a generated CA the worker trusts via
// SSL_CERT_FILE.
type HTTPMock struct {
	// ProxyURL is the http://host:port address to pass as HTTPS_PROXY.
	ProxyURL string
	// CAFile is the path to a PEM file containing the mock's CA certificate,
	// suitable for SSL_CERT_FILE.
	CAFile string
}

// WorkerEnv returns the environment entries a worker needs to route its HTTPS
// calls through the mock: the proxy address, the CA to trust, and a NO_PROXY
// that keeps loopback traffic direct. Pass any additional hosts that must not
// be proxied, in particular the Temporal address host, since the proxy would
// otherwise also intercept the worker's gRPC connection to Temporal.
func (m *HTTPMock) WorkerEnv(noProxyHosts ...string) []string {
	noProxy := "127.0.0.1,localhost"
	for _, h := range noProxyHosts {
		if h != "" {
			noProxy += "," + h
		}
	}

	return []string{
		"HTTPS_PROXY=" + m.ProxyURL,
		"HTTP_PROXY=" + m.ProxyURL,
		"SSL_CERT_FILE=" + m.CAFile,
		"NO_PROXY=" + noProxy,
	}
}

// StartHTTPSMock starts an HTTPS-intercepting mock proxy. routes maps a request
// path (for example "/users/3") to the JSON value returned for that path. The
// hosts argument lists the TLS server names the mock must answer for (for
// example "jsonplaceholder.typicode.com"). The proxy and its certificate are
// torn down when the test finishes.
func StartHTTPSMock(t *testing.T, hosts []string, routes map[string]any) *HTTPMock {
	t.Helper()

	caPEM, leaf := generateMockCerts(t, hosts)

	caFile := filepath.Join(t.TempDir(), "mock-ca.pem")
	require.NoError(t, os.WriteFile(caFile, caPEM, 0o600), "write mock CA file")

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{leaf},
		MinVersion:   tls.VersionTLS12,
	}

	handler := &mockProxy{tlsConfig: tlsConfig, routes: routes, t: t}

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen for mock proxy")

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	})

	return &HTTPMock{
		ProxyURL: "http://" + listener.Addr().String(),
		CAFile:   caFile,
	}
}

// mockProxy answers CONNECT tunnels with TLS-terminated fixture responses.
type mockProxy struct {
	tlsConfig *tls.Config
	routes    map[string]any
	t         *testing.T
}

func (p *mockProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		// Plain HTTP fixtures, in case a future example calls an http:// URL.
		p.writeFixture(w, r)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer func() { _ = clientConn.Close() }()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	tlsConn := tls.Server(clientConn, p.tlsConfig)
	if err := tlsConn.HandshakeContext(r.Context()); err != nil {
		return
	}
	defer func() { _ = tlsConn.Close() }()

	reader := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}
		body, status := p.lookup(req.URL.Path)
		if err := writeRawResponse(tlsConn, body, status); err != nil {
			return
		}
	}
}

// writeFixture serves a fixture over a plain http.ResponseWriter.
func (p *mockProxy) writeFixture(w http.ResponseWriter, r *http.Request) {
	body, status := p.lookup(r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeRawResponse writes a minimal HTTP/1.1 JSON response to a raw connection.
func writeRawResponse(w io.Writer, body []byte, status int) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "HTTP/1.1 %d %s\r\n", status, http.StatusText(status))
	fmt.Fprintf(&b, "Content-Type: application/json\r\n")
	fmt.Fprintf(&b, "Content-Length: %d\r\n", len(body))
	b.WriteString("Connection: keep-alive\r\n\r\n")
	b.Write(body)

	_, err := w.Write(b.Bytes())
	return err
}

// lookup returns the JSON body and status for a path. A route value may be a
// MockResponse to control the status code; any other value is served as 200.
func (p *mockProxy) lookup(path string) (body []byte, status int) {
	value, ok := p.routes[path]
	if !ok {
		p.t.Logf("[httpmock] no fixture for path %q", path)
		return []byte(`{"error":"not found"}`), http.StatusNotFound
	}

	status = http.StatusOK
	if mr, isMock := value.(MockResponse); isMock {
		status = mr.Status
		value = mr.Body
	}

	body, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"error":"encode"}`), http.StatusInternalServerError
	}
	return body, status
}

// generateMockCerts creates a self-signed CA and a leaf certificate valid for
// the given hosts, signed by that CA. It returns the CA in PEM form (for
// SSL_CERT_FILE) and the leaf as a usable tls.Certificate.
func generateMockCerts(t *testing.T, hosts []string) (caPEM []byte, leaf tls.Certificate) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generate CA key")

	notBefore := time.Now().Add(-time.Hour)
	notAfter := time.Now().Add(24 * time.Hour)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Zigflow e2e mock CA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err, "create CA certificate")

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err, "parse CA certificate")

	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generate leaf key")

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: hosts[0]},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     hosts,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err, "create leaf certificate")

	leaf = tls.Certificate{
		Certificate: [][]byte{leafDER, caDER},
		PrivateKey:  leafKey,
	}

	return caPEM, leaf
}

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

package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// maxDownloadSize is the maximum number of bytes read from an HTTP response body.
const maxDownloadSize int64 = 10 * 1024 * 1024 // 10 MiB

func readFileContents(parsed *url.URL, rawURL string) ([]byte, error) {
	path := parsed.Path

	// On Windows, strip leading slash from /C:/path
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	// Non-standard, but allow relative paths
	if strings.HasPrefix(rawURL, "file://./") {
		path = strings.TrimPrefix(rawURL, "file://")
	}

	//nolint:gosec // path originates from trusted config, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return data, nil
}

func readHTTPContents(ctx context.Context, rawURL string, client ...*http.Client) ([]byte, error) {
	if len(client) == 0 {
		client = append(client, &http.Client{Timeout: 5 * time.Second})
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client[0].Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	if resp.ContentLength > maxDownloadSize {
		return nil, fmt.Errorf("response too large: Content-Length %d exceeds limit of %d bytes", resp.ContentLength, maxDownloadSize)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if int64(len(body)) > maxDownloadSize {
		return nil, fmt.Errorf("response too large: exceeds limit of %d bytes", maxDownloadSize)
	}
	return body, nil
}

func ReadURLContents(ctx context.Context, rawURL string, client ...*http.Client) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	switch parsed.Scheme {
	case "http", "https":
		return readHTTPContents(ctx, rawURL, client...)

	case "file":
		return readFileContents(parsed, rawURL)

	case "":
		return nil, fmt.Errorf("missing url scheme (expected http, https, or file)")

	default:
		return nil, fmt.Errorf("unsupported scheme: %q", parsed.Scheme)
	}
}

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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadURLContentsUnsupportedScheme(t *testing.T) {
	_, err := ReadURLContents(context.Background(), "ftp://example.com/file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scheme")
}

func TestReadURLContentsMissingScheme(t *testing.T) {
	_, err := ReadURLContents(context.Background(), "no-scheme-here")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing url scheme")
}

func TestReadURLContentsHTTPSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "hello")
	}))
	defer srv.Close()

	body, err := ReadURLContents(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), body)
}

func TestReadURLContentsHTTPNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := ReadURLContents(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}

// TestReadURLContentsContentLengthTooLarge verifies that a response whose
// Content-Length header exceeds maxDownloadSize is rejected before reading the body.
func TestReadURLContentsContentLengthTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", maxDownloadSize+1))
		_, _ = fmt.Fprint(w, "x") // tiny body — the check must fire on the header
	}))
	defer srv.Close()

	_, err := ReadURLContents(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response too large")
	assert.Contains(t, err.Error(), "Content-Length")
}

// TestReadURLContentsBodyTooLarge verifies that a response with no Content-Length
// but a body exceeding maxDownloadSize is rejected after bounded reading.
func TestReadURLContentsBodyTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write maxDownloadSize+1 bytes without a Content-Length header so the
		// early-rejection path is bypassed and the LimitReader path is exercised.
		data := make([]byte, maxDownloadSize+1)
		w.Header().Set("Transfer-Encoding", "chunked")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	_, err := ReadURLContents(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response too large")
	assert.NotContains(t, err.Error(), "Content-Length")
}

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
	"bytes"
	"io"
	"testing"

	"go.temporal.io/sdk/log"
)

type probeLogger struct{}

func (probeLogger) Debug(string, ...any) {}
func (probeLogger) Info(string, ...any)  {}
func (probeLogger) Warn(string, ...any)  {}
func (probeLogger) Error(string, ...any) {}

var _ log.Logger = probeLogger{}

// io.Writer: "Write must return a non-nil error if it returns n < len(p)."
func TestLogWriterHonoursIoWriterContract(t *testing.T) {
	w := LogWriter{Logger: probeLogger{}, Level: "info"}
	for _, p := range [][]byte{[]byte("\n"), []byte("   "), []byte(" \t\n ")} {
		n, err := w.Write(p)
		if err == nil && n != len(p) {
			t.Errorf("Write(%q) = (%d, nil); io.Writer requires n == len(p) (%d) when err is nil",
				p, n, len(p))
		}
	}
}

// The real-world consequence: run:script pipes stdout/stderr through
// io.MultiWriter, which fails the whole copy on a short write.
func TestLogWriterUnderMultiWriter(t *testing.T) {
	var buf bytes.Buffer
	mw := io.MultiWriter(&buf, LogWriter{Logger: probeLogger{}, Level: "info"})
	if _, err := mw.Write([]byte("\n")); err != nil {
		t.Fatalf("MultiWriter rejected a whitespace-only chunk: %v "+
			"(this is the intermittent 'short write' from run:script)", err)
	}
}

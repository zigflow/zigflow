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
	"strings"

	"go.temporal.io/sdk/log"
)

// LogWriter converts the Temporal log.Logger to an io.Writer for streaming
type LogWriter struct {
	Logger log.Logger
	Level  string
	Msg    string
	Fields []any
}

func (w LogWriter) AddFields(args []any) LogWriter {
	w.Fields = append(w.Fields, args...)
	return w
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	// This may include multiline strings
	line := string(p)

	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	msg := "New line"
	if w.Msg != "" {
		msg = w.Msg
	}

	f := append([]any{"output", line}, w.Fields...)

	switch strings.ToLower(w.Level) {
	case "debug":
		w.Logger.Debug(msg, f...)
	case "warn":
		w.Logger.Warn(msg, f...)
	case "error":
		w.Logger.Error(msg, f...)
	default:
		w.Logger.Info(msg, f...)
	}

	return len(p), nil
}

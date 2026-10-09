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

package testrunner

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// FormatHuman writes a concise pass/fail summary for CLI output.
func FormatHuman(w io.Writer, res *Result) error {
	if res == nil {
		return fmt.Errorf("result is nil")
	}

	marker := "✓"
	if res.Status != StatusCompleted {
		marker = "✗"
	}

	line := fmt.Sprintf(
		"%s %s   %s   %s\n",
		marker,
		res.WorkflowType,
		res.Status,
		formatDuration(res.Duration),
	)
	if _, err := io.WriteString(w, line); err != nil {
		return err
	}

	if res.Output != nil && !isEmptyOutput(res.Output) {
		if _, err := io.WriteString(w, "\noutput:\n"); err != nil {
			return err
		}
		if err := writeIndentedJSON(w, res.Output); err != nil {
			return err
		}
	}

	if res.Status != StatusCompleted && res.Err != nil {
		if _, err := fmt.Fprintf(w, "\n  error: %v\n", res.Err); err != nil {
			return err
		}
	}
	if res.TemporalUIURL != "" {
		if _, err := fmt.Fprintf(w, "\n  inspect: %s\n", res.TemporalUIURL); err != nil {
			return err
		}
	}
	return nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func writeIndentedJSON(w io.Writer, v any) error {
	raw, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return err
	}
	lines := strings.SplitSeq(string(raw), "\n")
	for line := range lines {
		if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
			return err
		}
	}
	return nil
}

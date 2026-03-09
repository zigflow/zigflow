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

package cloudevents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	sdk "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

// FileSender implements protocol.Sender to write CloudEvents to files.
type FileSender struct {
	targetDir string
}

// convertJSONNumbers recursively converts json.Number values to appropriate types
// to prevent scientific notation and precision loss when marshalling to YAML.
func convertJSONNumbers(v any) any {
	switch val := v.(type) {
	case json.Number:
		// Try to parse as int64 first (for whole numbers)
		if i, err := val.Int64(); err == nil {
			return i
		}
		// If it has a decimal point or is too large for int64, keep as string
		return val.String()
	case map[string]any:
		// Recursively process map values
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = convertJSONNumbers(v)
		}
		return result
	case []any:
		// Recursively process slice elements
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = convertJSONNumbers(v)
		}
		return result
	default:
		return v
	}
}

// NewFileSender creates a new FileSender that writes events to the specified directory.
func NewFileSender(targetDir string) (*FileSender, error) {
	// Ensure the target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	return &FileSender{
		targetDir: targetDir,
	}, nil
}

// Send writes the CloudEvent to a file named after the event ID.
// If the file already exists, the event is appended to the file.
func (s *FileSender) Send(ctx context.Context, m binding.Message, transformers ...binding.Transformer) error {
	defer func() {
		if err := m.Finish(nil); err != nil {
			log.Warn().Err(err).Msg("Failed to finish message")
		}
	}()

	// Convert the binding.Message to a CloudEvent
	event, err := binding.ToEvent(ctx, m, transformers...)
	if err != nil {
		return fmt.Errorf("failed to convert message to event: %w", err)
	}

	// Use the event ID as the filename
	filename := filepath.Join(s.targetDir, event.ID()+".yaml")

	// Build a complete representation of the event
	eventMap := map[string]any{
		"specversion": event.SpecVersion(),
		"id":          event.ID(),
		"source":      event.Source(),
		"type":        event.Type(),
		"time":        event.Time(),
	}

	// Add the subject
	if s := event.Subject(); s != "" {
		eventMap["subject"] = s
	}

	// Add the data content type
	if t := event.DataContentType(); t != "" {
		eventMap["datacontenttype"] = t
	}

	// Add the data, converting to JSON if appropriate
	// The event.DataAs function doesn't allow passing any type for strings
	if d := event.Data(); d != nil {
		var data any
		if event.DataContentType() == sdk.ApplicationJSON {
			// Convert JSON to a map using a decoder with UseNumber() to preserve
			// numeric precision and avoid scientific notation for large numbers
			data = map[string]any{}
			decoder := json.NewDecoder(bytes.NewReader(d))
			decoder.UseNumber()
			if err := decoder.Decode(&data); err != nil {
				return fmt.Errorf("error converting event data to json: %w", err)
			}
			// Convert json.Number values to appropriate types
			data = convertJSONNumbers(data)
		} else {
			// Output the string if type is unknown
			data = string(d)
		}
		eventMap["data"] = data
	}

	// Add the extensions
	maps.Copy(eventMap, event.Extensions())

	// Convert any json.Number values in the entire map before YAML marshalling
	eventMap = convertJSONNumbers(eventMap).(map[string]any)

	// Marshal the event map to YAML
	data, err := yaml.Marshal(eventMap)
	if err != nil {
		return fmt.Errorf("failed to marshal event to YAML: %w", err)
	}

	// Open file in append mode, create if it doesn't exist
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		err = f.Close()
	}()

	// Add YAML document separator
	if _, err := f.WriteString("---\n"); err != nil {
		return fmt.Errorf("failed to write document separator: %w", err)
	}

	// Write the event YAML
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write event to file: %w", err)
	}

	log.Debug().
		Str("event_id", event.ID()).
		Str("file", filename).
		Msg("CloudEvent written to file")

	return err
}

// Ensure FileSender implements protocol.Sender
var _ protocol.Sender = (*FileSender)(nil)

func (c *ClientConfig) loadFileClient() (sdk.Client, error) {
	targetDir := c.Target
	if targetDir == "" {
		return nil, fmt.Errorf("file client %q: target directory is required", c.Name)
	}

	sender, err := NewFileSender(targetDir)
	if err != nil {
		return nil, fmt.Errorf("file client %q: %w", c.Name, err)
	}

	log.Debug().
		Str("name", c.Name).
		Str("target", targetDir).
		Msg("Creating new File CloudEvent client")

	return sdk.NewClient(sender, sdk.WithTimeNow(), sdk.WithUUIDs())
}

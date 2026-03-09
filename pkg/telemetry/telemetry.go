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

package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog/log"
)

const (
	// Project API keys (starting "phc_") are safe to be stored publicly
	// @link https://posthog.com/docs/privacy
	//nolint:gosec
	apiKey           = "phc_aAZLi0FMmGUug73jLYdTIkFjns49I9YcpOUs6TztZ0B"
	endpoint         = "https://cli.zigflow.dev"
	heartbeatTimeout = time.Minute
)

type TelemetryEvents string

const (
	HeartbeatEvent   TelemetryEvents = "heartbeat_run_count"
	StartWorkerEvent TelemetryEvents = "worker_started"
)

type Telemetry struct {
	isDisabled    bool
	heartbeatOnce sync.Once
	startTime     time.Time
	stopCh        chan struct{}

	arch        string
	distinctID  string
	isContainer bool
	os          string
	runCount    uint64
	version     string
}

// Increment the number of workflow runs
func (t *Telemetry) IncrementRun() {
	if t.isDisabled {
		return
	}

	newValue := atomic.AddUint64(&t.runCount, 1)
	log.Trace().Uint64("count", newValue).Msg("Incrementing telemetry run count")

	t.heartbeatOnce.Do(func() {
		go t.startHeartbeat()
	})
}

// A new worker is started
func (t *Telemetry) StartWorker() {
	if t.isDisabled {
		return
	}

	if err := t.capture(StartWorkerEvent, t.baseProps()); err != nil {
		log.Trace().Err(err).Msg("Failed to send worker started")
	}
}

// Shutdown the long-running heartbeat
func (t *Telemetry) Shutdown() {
	if t == nil || t.isDisabled {
		return
	}

	select {
	case <-t.stopCh:
		return
	default:
		close(t.stopCh)
	}
}

// baseProps returns the common set of properties sent with every event.
func (t *Telemetry) baseProps() posthog.Properties {
	return posthog.NewProperties().
		Set("arch", t.arch).
		Set("version", t.version).
		Set("is_container", t.isContainer).
		Set("os", t.os)
}

// capture creates a PostHog client, enqueues the event, and closes the client.
// Returns immediately if telemetry is disabled.
func (t *Telemetry) capture(event TelemetryEvents, properties posthog.Properties) error {
	if t.isDisabled {
		return nil
	}

	client, err := posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: endpoint})
	if err != nil {
		return fmt.Errorf("error creating posthog connection: %w", err)
	}

	defer func() {
		_ = client.Close()
	}()

	// Log for transparency
	log.Trace().
		Str("id", t.distinctID).
		Str("event", string(event)).
		Any("properties", properties).
		Msg("Sending anonymous telemetry")

	// Send the data
	if err := client.Enqueue(posthog.Capture{
		DistinctId: t.distinctID,
		Event:      string(event),
		Properties: properties,
	}); err != nil {
		return fmt.Errorf("error sending posthog telemetry: %w", err)
	}

	return nil
}

// Start a new worker's heartbeat. This is to avoid sending lots of duplicate
// or zero data in the telemetry
func (t *Telemetry) startHeartbeat() {
	ticker := time.NewTicker(heartbeatTimeout)
	defer ticker.Stop()

	var lastValue uint64

	for {
		select {
		case <-ticker.C:
			// Only send if data is different
			current := atomic.LoadUint64(&t.runCount)
			if lastValue != current {
				props := t.baseProps().
					Set("workflow_run_count", current).
					Set("uptime_seconds", time.Since(t.startTime).Seconds())

				if err := t.capture(HeartbeatEvent, props); err != nil {
					log.Trace().Err(err).Msg("Failed to send heartbeat telemetry")
				}

				lastValue = current
			} else {
				log.Trace().
					Uint64("last_value", lastValue).
					Uint64("run_count", current).
					Msg("Last value and run count the same - telemetry not sent")
			}

		case <-t.stopCh:
			return
		}
	}
}

func New(version string, disabled bool) (*Telemetry, error) {
	t := &Telemetry{
		arch:       runtime.GOARCH,
		isDisabled: disabled || version == gh.Development,
		os:         runtime.GOOS,
		startTime:  time.Now(),
		stopCh:     make(chan struct{}),
		version:    version,
	}

	if t.isDisabled {
		return t, nil
	}

	distinctID, isContainer, err := resolveID()
	if err != nil {
		return nil, err
	}

	t.distinctID = distinctID
	t.isContainer = isContainer

	return t, nil
}

// getID looks for an ID in ~/.config/zigflow or creates it
func getID() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	// Build the path to the file
	appDir := filepath.Join(configDir, "zigflow")
	idFile := filepath.Join(appDir, "id")

	// Ensure the directory exists
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	// Try reading existing ID
	if data, err := os.ReadFile(idFile); err == nil {
		return string(data), nil
	}

	newID := uuid.NewString()

	// Persist it
	if err := os.WriteFile(idFile, []byte(newID), 0o600); err != nil {
		return "", fmt.Errorf("write id file: %w", err)
	}

	return newID, nil
}

func resolveID() (distinctID string, isContainer bool, err error) {
	if id, ok := os.LookupEnv("HOSTNAME"); ok {
		// If HOSTNAME envvar exists, assume it's a containerised environment
		// Avoid spaffing anything sensitive from hostname
		sum := sha256.Sum256([]byte(id))
		distinctID = hex.EncodeToString(sum[:])
		isContainer = true
	} else {
		distinctID, err = getID()
		if err != nil {
			err = fmt.Errorf("error generating distinct id: %w", err)
			return
		}
	}

	return
}

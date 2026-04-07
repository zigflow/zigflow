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
	"fmt"
	"os"
	"text/template"
	"time"

	sdk "github.com/cloudevents/sdk-go/v2"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/observability"
	"github.com/zigflow/zigflow/pkg/utils"
	"sigs.k8s.io/yaml"
)

type Events struct {
	Clients []*ClientConfig `json:"clients" validate:"dive"`

	workflow *model.Workflow
}

func (e *Events) loadClients() error {
	for _, c := range e.Clients {
		if err := c.load(); err != nil {
			return fmt.Errorf("error loading client %s: %w", c.Name, err)
		}
	}

	return nil
}

type Option func(*sdk.Event)

func WithEvent(f func(*sdk.Event)) Option {
	return func(e *sdk.Event) {
		f(e)
	}
}

func (e *Events) Emit(ctx context.Context, eventType string, opts ...Option) {
	event := sdk.NewEvent()

	for _, o := range opts {
		o(&event)
	}

	// Hard-code important things
	event.SetSpecVersion(sdk.VersionV1)
	// Format is zigflow.dev/<taskQueue>/<workflowType>
	event.SetSource(fmt.Sprintf("zigflow.dev/%s/%s", e.workflow.Document.Namespace, e.workflow.Document.Name))
	event.SetTime(time.Now())
	event.SetType(fmt.Sprintf("dev.zigflow.%s", eventType))

	for _, c := range e.Clients {
		if c.Disabled {
			continue
		}

		l := log.With().Str("name", c.Name).Str("type", eventType).Logger()

		start := time.Now()
		observability.EventsEmittedTotal.WithLabelValues(c.Name, eventType).Inc()

		if result := c.client.Send(ctx, event); sdk.IsUndelivered(result) {
			l.Error().Any("result", result).Msg("CloudEvent not delivered")
			observability.EventsUndeliveredTotal.WithLabelValues(c.Name, eventType).Inc()
		}
		observability.EventsUndeliveredTotal.WithLabelValues(c.Name, eventType).Inc()

		dur := time.Since(start)
		observability.EventEmitDuration.
			WithLabelValues(c.Name).
			Observe(dur.Seconds())

		l.Debug().Str("name", c.Name).Dur("duration", dur).Msg("New event triggered")
	}
}

func Load(path string, validator *utils.Validator, workflow *model.Workflow) (*Events, error) {
	cfg := Events{
		workflow: workflow,
	}

	// Allow empty string to be ignored
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}

		rendered, err := renderTemplate(raw)
		if err != nil {
			return nil, fmt.Errorf("render template: %w", err)
		}

		if err := yaml.Unmarshal(rendered, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal yaml: %w", err)
		}

		if res, err := validator.ValidateStruct(cfg); err != nil {
			return nil, fmt.Errorf("error creating validation stack: %w", err)
		} else if res != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	if err := cfg.loadClients(); err != nil {
		return nil, fmt.Errorf("error loading event clients: %w", err)
	}

	return &cfg, nil
}

func envMap() map[string]string {
	out := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				out[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return out
}

func renderTemplate(input []byte) ([]byte, error) {
	tmpl, err := template.New("config").
		Option("missingkey=error").
		Parse(string(input))
	if err != nil {
		return nil, fmt.Errorf("error creating template parser: %w", err)
	}

	data := map[string]any{
		"env": envMap(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return buf.Bytes(), nil
}

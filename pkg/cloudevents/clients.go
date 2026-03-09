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
	"fmt"

	sdk "github.com/cloudevents/sdk-go/v2"
)

type ClientConfig struct {
	Name     string         `json:"name" validate:"required"`
	Disabled bool           `json:"disabled"`
	Protocol string         `json:"protocol" validate:"required_if=Enabled true,oneof=file http"`
	Target   string         `json:"target" validate:"required_if=Enabled true"`
	Options  map[string]any `json:"options,omitempty"`

	client sdk.Client `json:"-"`
}

func (c *ClientConfig) load() error {
	if c.Disabled {
		return nil
	}

	var client sdk.Client
	var err error
	switch c.Protocol {
	case "file":
		client, err = c.loadFileClient()
	case "http":
		client, err = c.loadHTTPClient()
	default:
		return fmt.Errorf("unsupported protocol %q for client %q", c.Protocol, c.Name)
	}

	if err != nil {
		return fmt.Errorf("error loading client: %w", err)
	}

	// Attach the client to the struct
	c.client = client

	return nil
}

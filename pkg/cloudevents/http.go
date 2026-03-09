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
	"net/http"
	"strings"
	"time"

	sdk "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/rs/zerolog/log"
)

var validHTTPMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodConnect: {},
	http.MethodOptions: {},
	http.MethodTrace:   {},
}

func (c *ClientConfig) loadHTTPClient() (sdk.Client, error) {
	opts := []cehttp.Option{
		cehttp.WithTarget(c.Target),
	}

	method := http.MethodPost
	if m, ok := c.Options["method"].(string); ok {
		m = strings.ToUpper(m)

		if m != "" {
			if _, ok := validHTTPMethods[m]; !ok {
				return nil, fmt.Errorf("unknown http method: %s", m)
			}

			method = m
		}
	}
	opts = append(opts, cehttp.WithMethod(method))

	timeout := time.Second * 1
	if v, ok := c.Options["timeout"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	opts = append(opts, cehttp.WithClient(http.Client{
		Timeout: timeout,
	}))

	if headers, ok := c.Options["headers"].(map[string]any); ok {
		for k, v := range headers {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("http client %q: header %q must be a string", c.Name, k)
			}
			opts = append(opts, cehttp.WithHeader(k, s))
		}
	}

	log.Debug().
		Str("name", c.Name).
		Str("target", c.Target).
		Str("method", method).
		Dur("timeout", timeout).
		Msg("Creating new HTTP CloudEvent client")

	return sdk.NewClientHTTP(opts...)
}

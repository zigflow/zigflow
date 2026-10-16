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
	"strings"

	zigtemporal "github.com/zigflow/helpers"
	"go.temporal.io/sdk/contrib/envconfig"
)

// mergeTemporalEnv fills empty TemporalOpts fields from the Temporal SDK
// environment configuration (TEMPORAL_ADDRESS, TEMPORAL_NAMESPACE, and so on).
func mergeTemporalEnv(opts *zigtemporal.TemporalOpts) error {
	if opts == nil {
		return nil
	}

	envOpts, err := envconfig.LoadDefaultClientOptions()
	if err != nil {
		return err
	}

	if opts.Address == "" && envOpts.HostPort != "" {
		opts.Address = normalizeHostPort(envOpts.HostPort)
	}
	if opts.Namespace == "" && envOpts.Namespace != "" {
		opts.Namespace = envOpts.Namespace
	}

	return nil
}

func normalizeHostPort(hostPort string) string {
	return strings.Replace(hostPort, "0.0.0.0", "127.0.0.1", 1)
}

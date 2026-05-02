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
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
)

func ResolveAuthenticationPolicy(endpoint *model.Endpoint, doc *model.Workflow) error {
	if endpoint != nil &&
		endpoint.EndpointConfig != nil &&
		endpoint.EndpointConfig.Authentication != nil &&
		endpoint.EndpointConfig.Authentication.Use != nil {
		// The auth is a "use" reference, so we need to populate it
		use := *endpoint.EndpointConfig.Authentication.Use

		if doc.Use != nil && doc.Use.Authentications != nil {
			if auth, ok := doc.Use.Authentications[use]; ok {
				endpoint.EndpointConfig.Authentication = &model.ReferenceableAuthenticationPolicy{
					AuthenticationPolicy: auth,
				}

				return nil
			} else {
				return fmt.Errorf("authentication not found for use reference: %s", use)
			}
		} else {
			return fmt.Errorf("no authentications defined in the workflow for use reference: %s", use)
		}
	}

	return nil
}

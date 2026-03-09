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
	"encoding/json"
	"fmt"
)

// ToType this works in a similar way to mapstructure.Decode, but with JSON. This
// is because many of the Serverless Workflow types have custom marshal/unmarshal
// JSON functions.
func ToType[T any](m any, result T) error {
	payload, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error marshalling data to json: %w", err)
	}

	if err := json.Unmarshal(payload, &result); err != nil {
		return fmt.Errorf("error unmarshalling json to type: %w", err)
	}

	return nil
}

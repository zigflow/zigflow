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

package codec

import (
	"fmt"

	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"go.temporal.io/sdk/converter"
)

func NewAESConverter(keyPath string) (converter.DataConverter, error) {
	keys, err := aes.ReadKeyFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read key file %q: %w", keyPath, err)
	}
	return aes.DataConverter(keys), nil
}

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

package metadata

import (
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
)

func GetMaxHistoryLength(doc *model.Workflow) (int, error) {
	if v, ok := doc.Document.Metadata[MaxHistoryLengthAttribute]; ok {
		if !utils.CanBeInt(v) {
			return 0, fmt.Errorf("document.metadata.%s value must be an integer", MaxHistoryLengthAttribute)
		}
		return int(v.(float64)), nil
	}
	return 0, nil
}

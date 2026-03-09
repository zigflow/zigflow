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

package cmd

import (
	"context"
	"fmt"

	basic "app/v1"
)

// Command1 implementation. This just returns the input data
func (c *Commands) Command1(ctx context.Context, request *basic.Command1Request) (*basic.Command1Response, error) {
	res, err := c.client.Run(request.Input)
	if err != nil {
		return nil, fmt.Errorf("error running db: %w", err)
	}

	return &basic.Command1Response{
		Output: res,
	}, nil
}

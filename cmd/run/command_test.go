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

package run

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicMessage(t *testing.T) {
	tests := []struct {
		Name     string
		Input    any
		Expected string
	}{
		{
			Name:     "error value",
			Input:    errors.New("something went wrong"),
			Expected: "something went wrong",
		},
		{
			Name:     "string value",
			Input:    "a plain string",
			Expected: "a plain string",
		},
		{
			Name:     "other value",
			Input:    42,
			Expected: fmt.Sprintf("%+v", 42),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, panicMessage(test.Input))
		})
	}
}

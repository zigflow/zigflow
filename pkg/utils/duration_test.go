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

package utils_test

import (
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
)

func TestToDuration(t *testing.T) {
	tests := []struct {
		Name     string
		Duration model.DurationInline
		Expected time.Duration
	}{
		{
			Name:     "nil",
			Expected: 0,
		},
		{
			Name: "10 second",
			Duration: model.DurationInline{
				Seconds: 10,
			},
			Expected: time.Second * 10,
		},
		{
			Name: "1 minute",
			Duration: model.DurationInline{
				Minutes: 1,
			},
			Expected: time.Minute,
		},
		{
			Name: "Complete",
			Duration: model.DurationInline{
				Days:         4,
				Hours:        6,
				Minutes:      43,
				Seconds:      32,
				Milliseconds: 472,
			},
			Expected: (time.Hour * 24 * 4) + (time.Hour * 6) + (time.Minute * 43) + (time.Second * 32) + (time.Millisecond * 472),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, utils.ToDuration(&model.Duration{
				Value: test.Duration,
			}))
		})
	}
}

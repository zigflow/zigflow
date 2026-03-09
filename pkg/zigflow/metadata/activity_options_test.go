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

package metadata_test

import (
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/temporal"
)

func TestConvertRetryPolicy(t *testing.T) {
	tests := []struct {
		Name        string
		RetryPolicy *metadata.RetryPolicy
		Starting    *temporal.RetryPolicy
		Expected    *temporal.RetryPolicy
	}{
		{
			Name:        "Empty",
			RetryPolicy: &metadata.RetryPolicy{},
			Expected:    &temporal.RetryPolicy{},
		},
		{
			Name: "Full",
			RetryPolicy: &metadata.RetryPolicy{
				InitialInterval:        &model.Duration{Value: model.DurationInline{Seconds: 1}},
				BackoffCoefficient:     utils.Ptr(2.0),
				MaximumAttempts:        utils.Ptr[int32](3),
				MaximumInterval:        &model.Duration{Value: model.DurationInline{Seconds: 3}},
				NonRetryableErrorTypes: []string{"error1"},
			},
			Expected: &temporal.RetryPolicy{
				InitialInterval:        time.Second,
				BackoffCoefficient:     2.0,
				MaximumAttempts:        3,
				MaximumInterval:        time.Second * 3,
				NonRetryableErrorTypes: []string{"error1"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.RetryPolicy.ToTemporal(test.Starting))
		})
	}
}

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

package e2etest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// AssertNoChildWorkflowStarts checks the completed parent history for child
// workflow start commands.
func AssertNoChildWorkflowStarts(
	ctx context.Context,
	t *testing.T,
	c client.Client,
	run client.WorkflowRun,
) {
	t.Helper()

	history := c.GetWorkflowHistory(
		ctx,
		run.GetID(),
		run.GetRunID(),
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)

	childTypes := make([]string, 0)
	for history.HasNext() {
		event, err := history.Next()
		require.NoError(t, err, "read workflow history")

		attributes := event.GetStartChildWorkflowExecutionInitiatedEventAttributes()
		if attributes != nil {
			childTypes = append(childTypes, attributes.GetWorkflowType().GetName())
		}
	}

	assert.Empty(t, childTypes, "parent history should not contain child workflow starts")
}

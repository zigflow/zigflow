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

package tasks

import (
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
)

// assertRegistersOncePerWorker exercises the registration-dedup contract that
// every activity-dispatching task builder must honour: repeated Build() calls
// against the same worker register the per-task activity exactly once. Both
// CallHTTP and CallGRPC tests share this helper because the assertion shape
// is identical; new task builders should add a one-line test calling it.
func assertRegistersOncePerWorker(
	t *testing.T,
	workflowName, taskName string,
	newBuilder func(w *WorkflowRegistryMock, doc *model.Workflow, taskName string) (TaskBuilder, error),
) {
	t.Helper()

	doc := &model.Workflow{Document: model.Document{Name: workflowName}}

	w := new(WorkflowRegistryMock)
	w.
		On("RegisterActivityWithOptions", mock.Anything, activity.RegisterOptions{
			Name: workflowName + "." + taskName,
		}).
		Once()

	b, err := newBuilder(w, doc, taskName)
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err = b.Build()
		assert.NoError(t, err)
	}

	w.AssertExpectations(t)
}

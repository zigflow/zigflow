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

package tasks_test

import (
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/tasks"
	"go.temporal.io/sdk/testsuite"
)

var (
	// testWorkflow is a shared workflow instance for testing purposes.
	testWorkflow = &model.Workflow{
		Document: model.Document{
			Namespace: "some-namespace",
			Name:      "some-name",
		},
	}

	// testEvents is a shared events instance for testing purposes.
	testEvents, _ = cloudevents.Load("", nil, testWorkflow)
)

func TestWaitTaskBuilder(t *testing.T) {
	tests := []struct {
		Name     string
		Duration model.DurationInline
	}{
		{
			Name: "10 second delay",
			Duration: model.DurationInline{
				Seconds: 10,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			start := time.Now().UTC()
			env.SetStartTime(start)

			dur := &model.Duration{
				Value: test.Duration,
			}

			w, err := tasks.NewWaitTaskBuilder(nil, &model.WaitTask{
				Wait: dur,
			}, test.Name, nil, testEvents)
			assert.NoError(t, err)

			wf, err := w.Build()
			assert.NoError(t, err)

			env.RegisterWorkflow(wf)

			env.ExecuteWorkflow(wf, nil, nil)

			assert.NoError(t, env.GetWorkflowError())

			got := env.Now().UTC()
			want := start.Add(utils.ToDuration(dur))

			assert.True(t, got.Equal(want))
		})
	}
}

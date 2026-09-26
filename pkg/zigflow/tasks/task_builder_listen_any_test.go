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
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestListenTaskBuilderAnySignalCompletes is a regression test for
// https://github.com/zigflow/zigflow/issues/598: a `listen` task with
// `to.any` must complete as soon as any one of its signals is received,
// rather than waiting for the timeout.
func TestListenTaskBuilderAnySignalCompletes(t *testing.T) {
	const (
		signalApprove = "approve"
		signalReject  = "reject"
		listenTimeout = time.Hour
		signalAfter   = time.Second
	)

	newSignal := func(id string) *model.EventFilter {
		return &model.EventFilter{
			With: &model.EventProperties{
				ID:   id,
				Type: string(ListenTaskTypeSignal),
			},
		}
	}

	for _, signal := range []string{signalApprove, signalReject} {
		t.Run(signal, func(t *testing.T) {
			builder, err := NewListenTaskBuilder(nil, &model.ListenTask{
				Metadata: map[string]any{
					"timeout": listenTimeout.String(),
				},
				Listen: model.ListenTaskConfiguration{
					To: &model.EventConsumptionStrategy{
						Any: []*model.EventFilter{
							newSignal(signalApprove),
							newSignal(signalReject),
						},
					},
				},
			}, "listen", testWorkflow, testEvents, nil)
			require.NoError(t, err)
			require.NoError(t, builder.Validate())

			fn, err := builder.Build()
			require.NoError(t, err)

			var (
				listenErr error
				nextRan   bool
				elapsed   time.Duration
			)

			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
				start := workflow.Now(ctx)

				_, listenErr = fn(ctx, nil, utils.NewState())
				if listenErr != nil {
					return listenErr
				}

				// The following step only runs once the listener returns.
				nextRan = true
				elapsed = workflow.Now(ctx).Sub(start)
				return nil
			}, workflow.RegisterOptions{Name: "listen-any"})

			env.RegisterDelayedCallback(func() {
				env.SignalWorkflow(signal, map[string]any{testConstValue: signal})
			}, signalAfter)

			env.ExecuteWorkflow("listen-any")

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, listenErr, "listen task must return successfully once a to.any signal is received")
			require.NoError(t, env.GetWorkflowError())
			assert.True(t, nextRan, "the step following the listen task must run")
			assert.Less(t, elapsed, listenTimeout, "the workflow must complete before the listen timeout")
		})
	}
}

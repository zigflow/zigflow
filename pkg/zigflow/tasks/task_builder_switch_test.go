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
	"errors"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/flow"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestSwitchTaskBuilderBuildRejectsMultipleDefaults(t *testing.T) {
	task := &model.SwitchTask{
		Switch: []model.SwitchItem{
			{
				"defaultOne": {
					Then: &model.FlowDirective{Value: "child-a"},
				},
			},
			{
				"defaultTwo": {
					Then: &model.FlowDirective{Value: "child-b"},
				},
			},
		},
	}

	builder, err := NewSwitchTaskBuilder(nil, task, "switch-task", nil, testEvents, nil)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.Nil(t, fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple switch statements without when")
}

// runSwitch executes the built switch function inside a Temporal test
// workflow environment and returns the error the function produced.
// Because the switch task itself no longer dispatches anything, we
// surface its return value to the test through a closure rather than
// the workflow boundary (which would wrap the sentinel error).
func runSwitch(t *testing.T, task *model.SwitchTask, state *utils.State) error {
	t.Helper()

	builder, err := NewSwitchTaskBuilder(nil, task, "switch-task", nil, testEvents, nil)
	assert.NoError(t, err)

	fn, err := builder.Build()
	assert.NoError(t, err)

	var capturedErr error
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		_, fnErr := fn(ctx, nil, state)
		capturedErr = fnErr
		return nil, nil
	}, workflow.RegisterOptions{Name: "switch-host"})

	env.ExecuteWorkflow("switch-host")
	assert.NoError(t, env.GetWorkflowError())
	return capturedErr
}

func TestSwitchTaskBuilderEmitsFlowDirective(t *testing.T) {
	tests := []struct {
		name      string
		cases     []model.SwitchItem
		runFlag   bool
		wantErr   error  // sentinel matched with errors.Is; nil means no error expected
		wantRedir string // when non-empty, expect flow.RedirectError with this Target
	}{
		{
			name: "continue directive returns flow.ErrContinue",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
						Then: &model.FlowDirective{Value: string(model.FlowDirectiveContinue)},
					},
				},
			},
			runFlag: true,
			wantErr: flow.ErrContinue,
		},
		{
			name: "exit directive returns flow.ErrExit",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
						Then: &model.FlowDirective{Value: string(model.FlowDirectiveExit)},
					},
				},
			},
			runFlag: true,
			wantErr: flow.ErrExit,
		},
		{
			name: "end directive returns flow.ErrEnd",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
						Then: &model.FlowDirective{Value: string(model.FlowDirectiveEnd)},
					},
				},
			},
			runFlag: true,
			wantErr: flow.ErrEnd,
		},
		{
			name: "named target returns flow.RedirectError",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
						Then: &model.FlowDirective{Value: "handler-task"},
					},
				},
			},
			runFlag:   true,
			wantRedir: "handler-task",
		},
		{
			name: "matching case without then returns nil",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
					},
				},
			},
			runFlag: true,
		},
		{
			name: "no matching case and no default returns nil",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
						Then: &model.FlowDirective{Value: string(model.FlowDirectiveEnd)},
					},
				},
			},
			runFlag: false,
		},
		{
			name: "default case is taken when no other case matches",
			cases: []model.SwitchItem{
				{
					testConstSwitchMatch: {
						When: model.NewRuntimeExpression(testConstDataFlag),
						Then: &model.FlowDirective{Value: string(model.FlowDirectiveContinue)},
					},
				},
				{
					"fallback": {
						Then: &model.FlowDirective{Value: string(model.FlowDirectiveEnd)},
					},
				},
			},
			runFlag: false,
			wantErr: flow.ErrEnd,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := utils.NewState()
			state.AddData(map[string]any{testConstFlag: tc.runFlag})

			err := runSwitch(t, &model.SwitchTask{Switch: tc.cases}, state)

			switch {
			case tc.wantRedir != "":
				var redirect flow.RedirectError
				if assert.True(t, errors.As(err, &redirect), "expected flow.RedirectError, got %v", err) {
					assert.Equal(t, tc.wantRedir, redirect.Target)
				}
			case tc.wantErr != nil:
				assert.ErrorIs(t, err, tc.wantErr)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

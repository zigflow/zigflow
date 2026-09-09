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

package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/ctxpropagator"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// addWorkflowNowTestWorkflow exercises AddWorkflowNow the way task execution
// does: once after AddWorkflowInfo, then again later in the run, mimicking
// it being called at each task's execution point.
func addWorkflowNowTestWorkflow(ctx workflow.Context) (map[string]string, error) {
	s := NewState()
	s.AddWorkflowInfo(ctx)
	s.AddWorkflowNow(ctx)

	first, _ := s.Data[stateWorkflow].(map[string]any)
	firstNow, _ := first[stateNow].(string)

	if err := workflow.Sleep(ctx, time.Second); err != nil {
		return nil, err
	}

	s.AddWorkflowNow(ctx)
	second, _ := s.Data[stateWorkflow].(map[string]any)
	secondNow, _ := second[stateNow].(string)

	return map[string]string{
		"first_now":  firstNow,
		"second_now": secondNow,
	}, nil
}

func TestAddWorkflowNow_SetsRFC3339Timestamp(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(addWorkflowNowTestWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]string
	require.NoError(t, env.GetWorkflowResult(&result))

	firstNow, ok := result["first_now"]
	require.True(t, ok, "$data.workflow.now should be present after AddWorkflowNow")
	_, err := time.Parse(time.RFC3339, firstNow)
	assert.NoError(t, err, "$data.workflow.now must be RFC3339-formatted")

	secondNow, ok := result["second_now"]
	require.True(t, ok, "$data.workflow.now should still be present on a later call")
	_, err = time.Parse(time.RFC3339, secondNow)
	assert.NoError(t, err, "$data.workflow.now must be RFC3339-formatted")

	// The test environment auto-skips time across the Sleep, so calling
	// AddWorkflowNow again should reflect a later (or equal) instant,
	// confirming repeated calls keep the field fresh.
	assert.GreaterOrEqual(t, secondNow, firstNow)
}

func TestAddWorkflowNow_BeforeAddWorkflowInfo(t *testing.T) {
	// AddWorkflowNow is normally called after AddWorkflowInfo, but it must
	// not panic if the ordering is violated, and it should end up seeding
	// the full workflow info map (not just "now").
	workflowFn := func(ctx workflow.Context) (map[string]any, error) {
		s := NewState()
		s.AddWorkflowNow(ctx)

		wf, ok := s.Data[stateWorkflow].(map[string]any)
		require.True(t, ok)

		return wf, nil
	}

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(workflowFn)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var wf map[string]any
	require.NoError(t, env.GetWorkflowResult(&wf))

	now, ok := wf[stateNow].(string)
	require.True(t, ok, "$data.workflow.now should be present")
	_, err := time.Parse(time.RFC3339, now)
	assert.NoError(t, err)

	// Confirm AddWorkflowInfo's fields were seeded too, not just "now".
	_, ok = wf["workflow_execution_id"]
	assert.True(t, ok, "AddWorkflowNow should seed the rest of $data.workflow when missing")
}

func TestState_GetAsMap_ExposesWorkflowNow(t *testing.T) {
	// Runtime expressions read the state via GetAsMap ($data.workflow.now),
	// so verify the field survives that path, not just the raw Data map.
	workflowFn := func(ctx workflow.Context) (map[string]any, error) {
		s := NewState()
		s.AddWorkflowInfo(ctx)
		s.AddWorkflowNow(ctx)
		return s.GetAsMap(), nil
	}

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(workflowFn)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var asMap map[string]any
	require.NoError(t, env.GetWorkflowResult(&asMap))

	data, ok := asMap["$data"].(map[string]any)
	require.True(t, ok)

	wf, ok := data[stateWorkflow].(map[string]any)
	require.True(t, ok, "$data.workflow should be present")

	now, ok := wf[stateNow].(string)
	require.True(t, ok, "$data.workflow.now should be present")

	_, err := time.Parse(time.RFC3339, now)
	assert.NoError(t, err, "$data.workflow.now must be RFC3339-formatted")
}

func addActivityInfoTestActivity(ctx context.Context) (map[string]any, error) {
	s := NewState()
	s.AddActivityInfo(ctx)

	activityData, _ := s.Data["activity"].(map[string]any)
	return activityData, nil
}

func TestAddActivityInfo_SetsRFC3339Now(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(addActivityInfoTestActivity)

	val, err := env.ExecuteActivity(addActivityInfoTestActivity)
	require.NoError(t, err)

	var activityData map[string]any
	require.NoError(t, val.Get(&activityData))

	now, ok := activityData[stateNow].(string)
	require.True(t, ok, "$data.activity.now should be present")

	_, err = time.Parse(time.RFC3339, now)
	assert.NoError(t, err, "$data.activity.now must be RFC3339-formatted")
}

const testCorrelationID = "corr-123"

func TestAddContextPropagator(t *testing.T) {
	tests := []struct {
		name       string
		propagated any
		setValue   bool
		want       map[string]any
	}{
		{
			name:       "copies propagated values onto the state",
			propagated: map[string]any{ctxpropagator.CorrelationID: testCorrelationID, "tenant": "acme"},
			setValue:   true,
			want:       map[string]any{ctxpropagator.CorrelationID: testCorrelationID, "tenant": "acme"},
		},
		{
			name:     "leaves an empty map when nothing was propagated",
			setValue: false,
			want:     map[string]any{},
		},
		{
			name:       "ignores a propagated value that is not a map",
			propagated: "not-a-map",
			setValue:   true,
			want:       map[string]any{},
		},
		{
			name:       "ignores a propagated map of the wrong type",
			propagated: map[string]string{ctxpropagator.CorrelationID: testCorrelationID},
			setValue:   true,
			want:       map[string]any{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflowFn := func(ctx workflow.Context) (map[string]any, error) {
				if tc.setValue {
					ctx = workflow.WithValue(ctx, ctxpropagator.PropagateKey, tc.propagated)
				}

				// An unexpected value must neither panic nor leave the
				// state unusable, so read it back through GetAsMap.
				return NewState().AddContextPropagator(ctx).GetAsMap(), nil
			}

			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			env.ExecuteWorkflow(workflowFn)

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())

			got := map[string]any{}
			require.NoError(t, env.GetWorkflowResult(&got))

			assert.Equal(t, tc.want, got["$propagated"])
			assert.Equal(t, map[string]any{}, got["$data"])
			assert.Equal(t, map[string]any{}, got["$env"])
		})
	}
}

func TestNewState_ContextPropagatorDefaultsToEmptyMap(t *testing.T) {
	assert.Equal(t, map[string]any{}, NewState().ContextPropagator)
}

func TestState_GetAsMap_ExposesPropagated(t *testing.T) {
	// Runtime expressions read propagated values as $propagated.
	workflowFn := func(ctx workflow.Context) (map[string]any, error) {
		ctx = workflow.WithValue(ctx, ctxpropagator.PropagateKey, map[string]any{
			ctxpropagator.CorrelationID: testCorrelationID,
		})

		return NewState().AddContextPropagator(ctx).GetAsMap(), nil
	}

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(workflowFn)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var asMap map[string]any
	require.NoError(t, env.GetWorkflowResult(&asMap))

	assert.Equal(t, map[string]any{ctxpropagator.CorrelationID: testCorrelationID}, asMap["$propagated"])
}

func TestState_GetAsMap_PropagatedIsEmptyWhenUnset(t *testing.T) {
	assert.Equal(t, map[string]any{}, NewState().GetAsMap()["$propagated"])
}

func TestState_Clone_PreservesContextPropagator(t *testing.T) {
	s := NewState()
	s.ContextPropagator = map[string]any{ctxpropagator.CorrelationID: testCorrelationID}

	clone := s.Clone()
	require.Equal(t, s.ContextPropagator, clone.ContextPropagator)

	// The clone owns its propagated values: mutating them must not reach
	// back into the state it was cloned from.
	clone.ContextPropagator[ctxpropagator.CorrelationID] = "corr-mutated"
	clone.ContextPropagator["added"] = true

	assert.Equal(t, map[string]any{ctxpropagator.CorrelationID: testCorrelationID}, s.ContextPropagator)
}

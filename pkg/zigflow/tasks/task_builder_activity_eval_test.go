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
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
)

func TestEvaluateTaskForActivity_ResolvesEnvExpressions(t *testing.T) {
	task := &model.CallGRPC{
		Call: "grpc",
		With: model.GRPCArguments{
			Arguments: map[string]any{
				"input": "${ $env." + testConstGRPCInputEnv + " }",
			},
		},
	}

	state := utils.NewState()
	state.Env[testConstGRPCInputEnv] = testConstHello

	resolved, err := evaluateTaskForActivity(task, state)
	require.NoError(t, err)
	assert.Equal(t, testConstHello, resolved.With.Arguments["input"])
}

func TestEvaluateTaskForActivity_SkipsActivityScopedExpressions(t *testing.T) {
	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image: "alpine",
				Environment: map[string]string{
					"WORKFLOW_ID": `${ $data.activity.workflow_execution_id }`,
				},
			},
		},
	}

	state := utils.NewState()

	resolved, err := evaluateTaskForActivity(task, state)
	require.NoError(t, err)
	assert.Equal(
		t,
		`${ $data.activity.workflow_execution_id }`,
		resolved.Run.Container.Environment["WORKFLOW_ID"],
	)
}

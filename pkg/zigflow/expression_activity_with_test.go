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

package zigflow_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

// Expressions inside activity `with` payloads are collected during validation
// (issue #462): they are walked as part of the task body, so invalid and
// non-deterministic expressions there fail `zigflow validate`, while ordinary
// and activity-runtime expressions pass.
func TestLoadFromBytes_ValidatesActivityWithExpressions(t *testing.T) {
	t.Run("invalid expression in grpc with arguments is rejected", func(t *testing.T) {
		yaml := workflowHeader + `do:
  - grpc:
      call: grpc
      with:
        proto:
          endpoint: file:///tmp/basic.proto
        service:
          name: providers.v1.BasicService
          host: grpc
          port: 3000
        method: Command1
        arguments:
          input: ${ $env.GRPC_INPUT | }
`
		_, err := zigflow.LoadFromBytes([]byte(yaml))
		require.Error(t, err, "invalid expression in with must be rejected")
		assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression)
	})

	t.Run("non-deterministic expression in grpc with arguments is rejected", func(t *testing.T) {
		yaml := workflowHeader + `do:
  - grpc:
      call: grpc
      with:
        proto:
          endpoint: file:///tmp/basic.proto
        service:
          name: providers.v1.BasicService
          host: grpc
          port: 3000
        method: Command1
        arguments:
          input: ${ uuid }
`
		_, err := zigflow.LoadFromBytes([]byte(yaml))
		require.Error(t, err, "non-deterministic expression in with must be rejected")
		assert.ErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
	})

	t.Run("ordinary and activity-runtime expressions in http with body are allowed", func(t *testing.T) {
		yaml := workflowHeader + `do:
  - withdraw:
      call: http
      with:
        method: post
        endpoint: http://server:3000/withdraw
        body:
          amount: ${ $input.amount }
          attempt: ${ $data.activity.attempt }
`
		wf, err := zigflow.LoadFromBytes([]byte(yaml))
		require.NoError(t, err, "ordinary and activity-runtime with expressions must validate")
		require.NotNil(t, wf)
	})
}

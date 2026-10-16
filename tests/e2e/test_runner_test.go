//go:build e2e

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

package e2e

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/testrunner"
)

// Each definition uses the same workflow type and a different output. A shared
// queue could execute the wrong definition and still report a successful run.
func TestE2EZigflowTestConcurrentIsolation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, listener.Close()) })
	connection := &temporal.TemporalOpts{
		HealthListenAddress:  listener.Addr().String(),
		MetricsListenAddress: listener.Addr().String(),
	}
	original := *connection
	cfg := testrunner.Config{
		Temporal: connection,
		Timeout:  time.Minute,
		Envvars:  map[string]any{"SOURCE": "sdk"},
	}
	const count = 8
	queues := make(chan string, count)
	// The outer subtest waits for every parallel run before inspecting the results.
	t.Run("runs", func(t *testing.T) {
		for i := range count {
			t.Run(fmt.Sprintf("definition-%d", i), func(t *testing.T) {
				t.Parallel()
				definition := fmt.Sprintf(`document:
  dsl: 1.0.0
  taskQueue: shared-document-queue
  workflowType: same-test-type
  version: 0.0.1
do:
  - child:
      fork:
        branches:
          - pause:
              wait:
                seconds: 1
  - identify:
      set:
        built_from: definition-%d
        source: ${ $env.SOURCE }
`, i)
				result, err := testrunner.RunBytes(t.Context(), []byte(definition), &cfg)
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, testrunner.StatusCompleted, result.Status, "%v", result.Err)
				assert.Equal(t, map[string]any{
					"built_from": fmt.Sprintf("definition-%d", i),
					"source":     "sdk",
				}, result.Output)
				require.Contains(t, result.TaskQueue, testrunner.TaskQueuePrefix)
				_, err = uuid.Parse(result.TaskQueue[len(testrunner.TaskQueuePrefix):])
				require.NoError(t, err)
				queues <- result.TaskQueue
			})
		}
	})
	close(queues)
	seen := map[string]bool{}
	for queue := range queues {
		assert.False(t, seen[queue], "runs must not share a task queue")
		seen[queue] = true
	}
	assert.Len(t, seen, count)
	assert.Equal(t, original, *connection, "shared SDK configuration must remain unchanged")
}

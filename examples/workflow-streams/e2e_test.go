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

package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/internal/e2etest"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/converter"
)

// TestWorkflowStreamsE2E runs the workflow-streams example and subscribes to
// every topic on its Workflow Stream. Each io.temporal.streams.<topic> emit
// task must arrive once, in order, on the topic named by the event type
// suffix. The subscription ends when the workflow completes.
func TestWorkflowStreamsE2E(t *testing.T) {
	ctx := t.Context()

	temporal := e2etest.StartTemporal(ctx, t)

	workflowFile, err := filepath.Abs("workflow.yaml")
	require.NoError(t, err)

	e2etest.StartWorker(ctx, t, temporal.Address, workflowFile)

	c, err := client.Dial(client.Options{HostPort: temporal.Address})
	require.NoError(t, err, "dial Temporal")
	defer c.Close()

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	we, err := c.ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}, "workflow-streams", map[string]any{})
	require.NoError(t, err, "execute workflow")

	// No topic filter, so an item published on an unexpected topic (for
	// example the full event type) would also be received and fail the test.
	stream := workflowstreams.NewClient(c, we.GetID(), workflowstreams.Options{})
	dc := converter.GetDefaultDataConverter()

	topics := make([]string, 0, 2)
	data := make([]any, 0, 2)
	for item, err := range stream.Subscribe(runCtx, workflowstreams.SubscribeOptions{}) {
		require.NoError(t, err, "subscribe to workflow stream")

		var value any
		require.NoError(t, dc.FromPayload(item.Data, &value), "decode stream item")
		t.Logf("stream item: topic=%s offset=%d data=%v", item.Topic, item.Offset, value)

		topics = append(topics, item.Topic)
		data = append(data, value)
	}

	require.NoError(t, we.Get(runCtx, nil), "workflow should complete")

	require.Equal(t, []string{"topic1", "topic2"}, topics, "stream topics")
	assert.Equal(t, map[string]any{
		"number":  3.141,
		"message": "Hello world",
		"happy":   true,
	}, data[0], "topic1 data")
	assert.Nil(t, data[1], "topic2 has no data")
}

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
	"time"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/workflow"
)

type fakeWorkflowContext struct{}

func (fakeWorkflowContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (fakeWorkflowContext) Done() workflow.Channel      { return nil }
func (fakeWorkflowContext) Err() error                  { return nil }
func (fakeWorkflowContext) Value(key interface{}) interface{} {
	return nil
}

func TestForkTaskBuilderAwaitCondition(t *testing.T) {
	builder := &ForkTaskBuilder{}

	tests := []struct {
		name        string
		replyErr    error
		isCompeting bool
		winningCtx  workflow.Context
		hasReplied  []bool
		expect      bool
	}{
		{
			name:     "reply error short circuits",
			replyErr: errors.New("boom"),
			expect:   true,
		},
		{
			name:        "competing fork waits for winner",
			isCompeting: true,
			expect:      false,
		},
		{
			name:        "competing fork with winner returns true",
			isCompeting: true,
			winningCtx:  fakeWorkflowContext{},
			expect:      true,
		},
		{
			name:       "non competing waits for all replies",
			hasReplied: []bool{true, false},
			expect:     false,
		},
		{
			name:       "non competing completes when all replied",
			hasReplied: []bool{true, true},
			expect:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cond := builder.awaitCondition(tc.replyErr, tc.isCompeting, tc.winningCtx, tc.hasReplied)
			assert.Equal(t, tc.expect, cond())
		})
	}
}

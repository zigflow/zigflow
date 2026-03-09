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
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/workflow"
)

type CancellableFuture struct {
	Cancel  workflow.CancelFunc
	Context workflow.Context
	Future  workflow.ChildWorkflowFuture
}

type CancellableFutures struct {
	m map[string]CancellableFuture
}

func (c *CancellableFutures) Add(key string, future CancellableFuture) {
	if c.m == nil {
		c.m = map[string]CancellableFuture{}
	}

	c.m[key] = future
}

func (c *CancellableFutures) CancelOthers(passedContext workflow.Context) {
	passedID := workflow.GetChildWorkflowOptions(passedContext).WorkflowID
	for _, f := range c.m {
		currentID := workflow.GetChildWorkflowOptions(f.Context).WorkflowID

		if currentID != passedID {
			log.Debug().
				Str("childWorkflowID", workflow.GetChildWorkflowOptions(f.Context).WorkflowID).
				Msg("Cancelling losing child workflow")

			f.Cancel()
		}
	}
}

func (c *CancellableFutures) Length() int {
	return len(c.m)
}

func (c *CancellableFutures) List() map[string]CancellableFuture {
	return c.m
}

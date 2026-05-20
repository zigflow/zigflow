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
	"fmt"

	"go.temporal.io/sdk/activity"
)

type SummarisePartialResultRequest struct {
	Question     string           `json:"question"`
	Observations []map[string]any `json:"observations"`
}

type SummarisePartialResultResponse struct {
	Answer string `json:"answer"`
}

// summarisePartialResult produces a deterministic best-effort answer when the
// agent loop runs out of iterations without reaching a final answer. It is
// intentionally simple so the example stays runnable without a model.
func summarisePartialResult(
	ctx context.Context,
	req SummarisePartialResultRequest,
) (SummarisePartialResultResponse, error) {
	activity.GetLogger(ctx).Info(
		"Summarising partial agent result",
		"question", req.Question,
		"observations", len(req.Observations),
	)

	answer := fmt.Sprintf(
		"I could not complete the agent loop, but I gathered %d observation(s) for: %s",
		len(req.Observations),
		req.Question,
	)

	return SummarisePartialResultResponse{Answer: answer}, nil
}

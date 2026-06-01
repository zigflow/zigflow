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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
)

// evaluateTaskForActivity resolves runtime expressions in a task definition
// before it is passed to a Temporal activity. Expressions that reference
// activity-scoped state are left for the activity to evaluate.
func evaluateTaskForActivity[T model.Task](task T, state *utils.State) (T, error) {
	var zero T

	payload, err := json.Marshal(task)
	if err != nil {
		return zero, fmt.Errorf("marshal activity task: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return zero, fmt.Errorf("unmarshal activity task: %w", err)
	}

	evaluated, err := utils.TraverseAndEvaluateObjWithOpts(
		model.NewObjectOrRuntimeExpr(data),
		nil,
		state,
		utils.TraverseEvaluateOpts{SkipExpression: skipActivityScopedExpression},
	)
	if err != nil {
		return zero, fmt.Errorf("evaluate activity task expressions: %w", err)
	}

	evaluatedMap, ok := evaluated.(map[string]any)
	if !ok {
		return zero, fmt.Errorf("evaluate activity task expressions: expected map, got %T", evaluated)
	}

	resolved, err := json.Marshal(evaluatedMap)
	if err != nil {
		return zero, fmt.Errorf("marshal evaluated activity task: %w", err)
	}

	var result T
	if err := json.Unmarshal(resolved, &result); err != nil {
		return zero, fmt.Errorf("unmarshal evaluated activity task: %w", err)
	}

	return result, nil
}

func skipActivityScopedExpression(expr string) bool {
	return strings.Contains(expr, "$data.activity")
}

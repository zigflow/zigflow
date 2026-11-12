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

package testrunner

import (
	"encoding/json"
	"errors"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"go.temporal.io/sdk/temporal"
)

const failureOutputMessageKey = "message"

// failureOutputForRun returns a JSON-friendly value describing a failed workflow
// run. When Temporal did not return a workflow result, structured Open Workflow
// Specification errors are extracted from the error chain when present.
func failureOutputForRun(workflowOutput any, runErr error) any {
	if !isEmptyOutput(workflowOutput) {
		return workflowOutput
	}
	return failureOutputFromError(runErr)
}

func failureOutputFromError(err error) any {
	if err == nil {
		return nil
	}

	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		if owsErr, ok := errors.AsType[*model.Error](cur); ok {
			return owsErrorAsMap(owsErr)
		}

		if appErr, ok := errors.AsType[*temporal.ApplicationError](cur); ok {
			var owsPtr *model.Error
			if appErr.Details(&owsPtr) == nil && owsPtr != nil {
				return owsErrorAsMap(owsPtr)
			}
			return map[string]any{
				"type":                  appErr.Type(),
				failureOutputMessageKey: appErr.Message(),
			}
		}
	}

	return map[string]any{
		failureOutputMessageKey: err.Error(),
	}
}

func owsErrorAsMap(e *model.Error) map[string]any {
	if e == nil {
		return nil
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return map[string]any{failureOutputMessageKey: e.Error()}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{failureOutputMessageKey: e.Error()}
	}
	return out
}

func isEmptyOutput(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case map[string]any:
		return len(x) == 0
	default:
		return false
	}
}

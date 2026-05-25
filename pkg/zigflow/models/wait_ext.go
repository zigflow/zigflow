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

package models

import (
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/zigflow/extensions"
)

const (
	keyWait         = "wait"
	keyUntil        = "until"
	keyDays         = "days"
	keyHours        = "hours"
	keyMinutes      = "minutes"
	keySeconds      = "seconds"
	keyMilliseconds = "milliseconds"
)

// WaitExtTask is the Zigflow extension of the Serverless Workflow wait
// task. It is registered with the SDK under
// extensions.ZigflowExtKeyPrefix + "wait" and constructed when the loader
// has renamed a wait task to its internal key. Expression resolution
// happens in the dedicated builder at workflow execution time.
type WaitExtTask struct {
	model.TaskBase `json:",inline"`
	Wait           *WaitExtBody `json:"__zigflow_ext_wait" validate:"required"`
}

func (w *WaitExtTask) GetBase() *model.TaskBase {
	return &w.TaskBase
}

// WaitExtBody carries either an absolute `until` moment or one or more
// duration fields. The schema's OneOf enforces that the two forms cannot
// be combined. Numeric fields are typed `any` because each may be a
// literal integer or a runtime expression string; evaluation and strict
// numeric coercion happen in the builder.
type WaitExtBody struct {
	Until        string `json:"until,omitempty"`
	Days         any    `json:"days,omitempty"`
	Hours        any    `json:"hours,omitempty"`
	Minutes      any    `json:"minutes,omitempty"`
	Seconds      any    `json:"seconds,omitempty"`
	Milliseconds any    `json:"milliseconds,omitempty"`
}

type waitExtension struct{}

func (waitExtension) TaskType() string { return keyWait }

// Claims a wait body when it carries an `until` field or a string-valued
// duration field. Vanilla literal-numeric wait bodies are left for the SDK.
func (waitExtension) Claims(body any) bool {
	m, ok := body.(map[string]any)
	if !ok {
		return false
	}
	if _, hasUntil := m[keyUntil]; hasUntil {
		return true
	}
	for _, k := range []string{keyDays, keyHours, keyMinutes, keySeconds, keyMilliseconds} {
		if v, ok := m[k]; ok {
			if _, isString := v.(string); isString {
				return true
			}
		}
	}
	return false
}

func init() {
	extensions.RegisterExtension(waitExtension{}, func() model.Task {
		return &WaitExtTask{}
	})
}

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

// Package extensions provides a generic pre-Open Workflow Specification SDK
// normalisation hook so that Zigflow can extend its semantics of tasks.
// An Extension inspects a task body before the SDK parses it and
// decides whether to claim ownership. A claimed task is renamed from its
// Open Workflow Specification task type (e.g. "wait") to a Zigflow-internal key
// formed by prefixing the task type with ZigflowExtKeyPrefix (e.g.
// "__zigflow_ext_wait"), so the SDK's task registry constructs the
// Zigflow Go type rather than the SDK's native one. Tasks that no
// extension claims are passed through to the SDK untouched.
package extensions

import (
	"fmt"

	"github.com/open-workflow-specification/sdk-go/v4/model"
)

// ZigflowExtKeyPrefix is the fixed prefix applied to an Open Workflow Specification
// task type to produce the Zigflow-internal task-type key for an
// extension. Use this constant when registering the Zigflow Go type with
// the SDK's task registry so the registered key matches the one the
// extensions package writes during normalisation.
const ZigflowExtKeyPrefix = "__zigflow_ext_"

// Extension is a pre-SDK normalisation hook. When Claims returns true,
// the loader renames the task body key from TaskType() to
// ZigflowExtKeyPrefix + TaskType() so the SDK constructs the Zigflow Go
// type registered under the renamed key.
type Extension interface {
	// TaskType is the YAML key this extension watches (e.g. "wait").
	// Extensions extending a spec task must use the spec key; extensions
	// introducing a Zigflow-only task type may use any key.
	TaskType() string

	// Claims reports whether this extension takes ownership of the task
	// body. Body is the raw value under TaskType().
	Claims(body any) bool
}

// registry holds the registered extensions. It is built at package init
// time by extension init() blocks and never mutated after that.
var registry []Extension

// Register adds an extension to the global registry. Registration is
// intended to happen exactly once per extension, from a package init()
// block. Registering a duplicate TaskType panics at init time, mirroring
// the Open Workflow Specification SDK's own behaviour for task-type collisions.
//
// Register only touches Zigflow's own registry. Real extensions usually
// want their Go type constructed by the SDK as well; use RegisterExtension
// to do both registrations in one call.
func Register(e Extension) {
	taskType := e.TaskType()

	if taskType == "" {
		panic("extensions: TaskType must not be empty")
	}

	for _, existing := range registry {
		if existing.TaskType() == taskType {
			panic(fmt.Sprintf("extensions: task type %q already registered by %T", taskType, existing))
		}
	}

	registry = append(registry, e)
}

// RegisterExtension is the convenience wrapper for the common pattern of
// registering both halves of a Zigflow extension from a package init():
// the SDK side (so the SDK constructs the right Go type when it sees the
// Zigflow internal key) and Zigflow's own normalise-side registry. Use
// this from extension init() blocks; use the bare Register in tests that
// drive the normalise step with stub extensions without polluting the
// SDK's global task registry.
func RegisterExtension(e Extension, newTask func() model.Task) {
	sdkKey := ZigflowExtKeyPrefix + e.TaskType()
	if err := model.RegisterTask(sdkKey, newTask); err != nil {
		panic(fmt.Sprintf("extensions: failed to register %q with the SDK: %v", sdkKey, err))
	}
	Register(e)
}

// Normalise runs the registered extensions against the given task body in
// registration order. The first extension that claims the task renames its
// body key from the task type to the Zigflow-internal key and returns.
// Tasks not claimed by any extension are untouched.
//
// Normalise is safe to call on any task body: extensions whose TaskType
// is not present in the task simply skip it.
func Normalise(task map[string]any) {
	for _, e := range registry {
		taskType := e.TaskType()
		body, ok := task[taskType]
		if !ok {
			continue
		}
		if e.Claims(body) {
			task[ZigflowExtKeyPrefix+taskType] = body
			delete(task, taskType)
			return
		}
	}
}

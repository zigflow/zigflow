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

package schema

// BuildDefs returns the $defs map to embed in the top-level schema. Each key
// becomes referenceable as #/$defs/<key> from anywhere in the schema.
func BuildDefs() map[string]*Schema {
	return map[string]*Schema{
		// Shared primitives
		"duration": durationSchema(),

		// Task structure
		"taskList": taskListSchema(),
		"taskItem": taskItemSchema(),
		"task":     taskUnionSchema(),
		"taskBase": taskBaseSchema(),

		// Concrete task types
		"callActivityTask": callActivityTaskSchema(),
		"callHttpTask":     callHTTPTaskSchema(),
		"callGrpcTask":     callGRPCTaskSchema(),
		"doTask":           doTaskSchema(),
		"forTask":          forTaskSchema(),
		"forkTask":         forkTaskSchema(),
		"listenTask":       listenTaskSchema(),
		"raiseTask":        raiseTaskSchema(),
		"runTask":          runTaskSchema(),
		"setTask":          setTaskSchema(),
		"switchTask":       switchTaskSchema(),
		"tryTask":          tryTaskSchema(),
		"waitTask":         waitTaskSchema(),

		// Metadata and Temporal options
		"metadata":        MetadataSchema(),
		"activityOptions": activityOptionsSchema(),
		"retryPolicy":     retryPolicySchema(),
	}
}

// durationSchema returns the shared schema for a duration value accepted by the
// Serverless Workflow runtime. A duration is either an ISO 8601 string (e.g.
// "PT30S") or an inline object with named time unit fields. This matches
// model.Duration.UnmarshalJSON which accepts both forms.
func durationSchema() *Schema {
	return WithDescription(
		AnyOf(
			WithDescription(String(), "ISO 8601 duration string (e.g. PT30S, P1DT2H)."),
			WithDescription(
				Object(map[string]*Schema{
					"days":         Integer(),
					"hours":        Integer(),
					"minutes":      Integer(),
					"seconds":      Integer(),
					"milliseconds": Integer(),
				}),
				"Inline duration with named time unit fields.",
			),
		),
		"A duration value. Either an ISO 8601 string or an object with named time unit fields.",
	)
}

// taskListSchema returns the schema for the `do` array used in workflow
// definitions and nested task containers (do, for, fork, try).
func taskListSchema() *Schema {
	return WithDescription(
		Array(Ref("#/$defs/taskItem")),
		"An ordered list of named tasks.",
	)
}

// taskItemSchema represents a single entry in a task list. Each item is an
// object with exactly one property: the task name maps to the task definition.
func taskItemSchema() *Schema {
	return &Schema{
		Type:                 "object",
		MinProperties:        intPtr(1),
		MaxProperties:        intPtr(1),
		AdditionalProperties: Ref("#/$defs/task"),
		Description:          "A named task. The single property key is the task name; the value is the task definition.",
	}
}

// taskUnionSchema returns the oneOf union of all supported Zigflow task types.
// A task is identified by its discriminator property (call, do, for, etc.).
func taskUnionSchema() *Schema {
	return WithDescription(
		OneOf(
			Ref("#/$defs/callActivityTask"),
			Ref("#/$defs/callHttpTask"),
			Ref("#/$defs/callGrpcTask"),
			Ref("#/$defs/doTask"),
			Ref("#/$defs/forTask"),
			Ref("#/$defs/forkTask"),
			Ref("#/$defs/listenTask"),
			Ref("#/$defs/raiseTask"),
			Ref("#/$defs/runTask"),
			Ref("#/$defs/setTask"),
			Ref("#/$defs/switchTask"),
			Ref("#/$defs/tryTask"),
			Ref("#/$defs/waitTask"),
		),
		"A Zigflow task. Exactly one task type must be present.",
	)
}

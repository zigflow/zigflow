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

// MetadataSchema returns the schema for the Zigflow metadata object. Metadata
// can appear on both workflow documents and individual tasks. Known keys are
// typed; arbitrary additional keys are permitted to support user-defined values.
func MetadataSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"activityOptions": Ref("#/$defs/activityOptions"),
			// heartbeat accepts a Duration (string or inline object).
			// The runtime converts it via utils.ToDuration which handles both forms.
			"heartbeat":           WithDescription(Ref("#/$defs/duration"), "How often to record an activity heartbeat."),
			"searchAttributes":    searchAttributesSchema(),
			"canMaxHistoryLength": WithDescription(Integer(), "Maximum Temporal workflow history length before Continue-As-New is triggered."),
		},
		AdditionalProperties: true,
	}
}

// activityOptionsSchema returns the schema for Temporal activity option overrides.
// Timeout fields accept a Duration (string or inline object) matching model.Duration.
// These map directly to the ActivityOptions struct in pkg/zigflow/metadata.
func activityOptionsSchema() *Schema {
	return Object(map[string]*Schema{
		"heartbeatTimeout":       WithDescription(Ref("#/$defs/duration"), "Duration before a heartbeat times out."),
		"scheduleToCloseTimeout": WithDescription(Ref("#/$defs/duration"), "Duration from schedule to close."),
		"scheduleToStartTimeout": WithDescription(Ref("#/$defs/duration"), "Duration from schedule to start."),
		"startToCloseTimeout":    WithDescription(Ref("#/$defs/duration"), "Duration from start to close."),
		"retryPolicy":            Ref("#/$defs/retryPolicy"),
		"disableEagerExecution":  WithDescription(Boolean(), "Disable Temporal eager activity execution."),
		"summary":                WithDescription(String(), "Human-readable summary shown in the Temporal UI."),
		"priority":               activityPrioritySchema(),
	})
}

func activityPrioritySchema() *Schema {
	return WithDescription(
		Object(map[string]*Schema{
			"priorityKey":    WithDescription(Integer(), "Temporal priority key for task queue ordering."),
			"fairnessKey":    WithDescription(String(), "Fairness key for weighted fair scheduling."),
			"fairnessWeight": WithDescription(Number(), "Fairness weight applied alongside fairnessKey."),
		}),
		"Temporal activity priority configuration.",
	)
}

// retryPolicySchema returns the schema for a Temporal retry policy. Interval
// fields accept a Duration (string or inline object) matching model.Duration.
// Fields map directly to the RetryPolicy struct in pkg/zigflow/metadata.
func retryPolicySchema() *Schema {
	return WithDescription(
		Object(map[string]*Schema{
			"initialInterval":        WithDescription(Ref("#/$defs/duration"), "Duration for the first retry interval (e.g. PT1S or {seconds: 1})."),
			"backoffCoefficient":     WithDescription(Number(), "Multiplier applied to the retry interval after each failure."),
			"maximumInterval":        WithDescription(Ref("#/$defs/duration"), "Cap on the retry interval."),
			"maximumAttempts":        WithDescription(Integer(), "Maximum number of retry attempts before the activity fails permanently."),
			"nonRetryableErrorTypes": WithDescription(Array(String()), "Error type names that should not be retried."),
		}),
		"Temporal retry policy for an activity.",
	)
}

// searchAttributesSchema returns the schema for the searchAttributes metadata
// key. The value is a map of attribute name to a SearchAttribute object.
func searchAttributesSchema() *Schema {
	return WithDescription(
		&Schema{
			Type:                 "object",
			AdditionalProperties: searchAttributeSchema(),
		},
		"Map of Temporal search attribute names to their type and value.",
	)
}

func searchAttributeSchema() *Schema {
	return WithDescription(
		Object(
			map[string]*Schema{
				"type": WithDescription(
					StringEnum("datetime", "keywordlist", "keyword", "text", "int", "double", "bool"),
					"Temporal search attribute type.",
				),
				"value": WithDescription(
					Any(),
					"Attribute value. Set to null to unset the attribute.",
				),
			},
			"type",
		),
		"A single Temporal search attribute.",
	)
}

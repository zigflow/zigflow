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

import "fmt"

const (
	// SchemaVersion is the JSON Schema draft used by the generated schema.
	SchemaVersion = "https://json-schema.org/draft/2020-12/schema"

	// SchemaID is the canonical URI for the generated Zigflow workflow schema.
	SchemaID = "https://zigflow.dev/schemas/%s/workflow.%s"

	// SchemaTitle is the title of the generated schema.
	SchemaTitle = "Zigflow"

	// SchemaDescription is the description of the generated schema.
	SchemaDescription = "JSON Schema for a Zigflow workflow definition."
)

// BuildSchema returns the complete JSON Schema for a Zigflow workflow document.
// The result can be marshalled directly to JSON.
func BuildSchema(version, format string) *Schema {
	return &Schema{
		SchemaURI:   SchemaVersion,
		ID:          fmt.Sprintf(SchemaID, version, format),
		Title:       SchemaTitle,
		Description: SchemaDescription,
		Type:        "object",
		Required:    []string{"document", "do"},
		Properties:  workflowProperties(),
		Defs:        BuildDefs(),
	}
}

func workflowProperties() map[string]*Schema {
	return map[string]*Schema{
		"document": documentSchema(),
		"do":       WithDescription(Ref("#/$defs/taskList"), "The ordered list of tasks the workflow executes."),
		"input":    WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Transforms the data passed into the workflow."),
		"output":   WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Transforms the data returned from the workflow."),
		"timeout":  workflowTimeoutSchema(),
		"schedule": scheduleSchema(),
	}
}

func documentSchema() *Schema {
	semverPattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
		`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

	slugPattern := `^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`

	return WithDescription(
		&Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"dsl": {
					Type:        "string",
					Pattern:     semverPattern,
					Title:       "WorkflowDSL",
					Description: "The Serverless Workflow DSL version this workflow targets.",
				},
				"namespace": {
					Type:        "string",
					Pattern:     slugPattern,
					Title:       "WorkflowNamespace",
					Description: "The workflow namespace. Must be a lowercase DNS label.",
				},
				"name": {
					Type:        "string",
					Pattern:     slugPattern,
					Title:       "WorkflowName",
					Description: "The workflow name. Must be a lowercase DNS label.",
				},
				"version": {
					Type:        "string",
					Pattern:     semverPattern,
					Title:       "WorkflowVersion",
					Description: "The workflow's semantic version.",
				},
				"title":    WithDescription(String(), "Optional human-readable title."),
				"summary":  WithDescription(String(), "Optional Markdown summary."),
				"tags":     WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Arbitrary key-value tags."),
				"metadata": WithDescription(Ref("#/$defs/metadata"), "Workflow-level metadata applied to all tasks unless overridden."),
			},
			Required:              []string{"dsl", "namespace", "name", "version"},
			UnevaluatedProperties: boolPtr(false),
		},
		"Workflow document block containing identity and version fields.",
	)
}

// scheduleSchema returns the schema for the top-level schedule block.
// Zigflow supports cron (cron expression string) and every (interval duration).
// schedule.after is explicitly rejected by the runtime ("schedule.after not supported").
// schedule.on is not handled and must be rejected here to prevent silent no-ops.
// additionalProperties: false enforces the supported subset without needing
// explicit not:{required:[after]} / not:{required:[on]} constraints.
func scheduleSchema() *Schema {
	s := WithDescription(
		Object(
			map[string]*Schema{
				"cron":  WithDescription(String(), "Cron expression defining when the workflow runs (e.g. 0 * * * *)."),
				"every": WithDescription(Ref("#/$defs/duration"), "Interval between workflow executions."),
			},
		),
		"Workflow schedule. Only cron and every are supported; after and on are not.",
	)
	s.AdditionalProperties = false
	return s
}

func workflowTimeoutSchema() *Schema {
	// The inline branch maps to model.Timeout which has a single required field After.
	// additionalProperties: false rejects unknown keys; required enforces After is present.
	inlineTimeout := Object(
		map[string]*Schema{
			"after": WithDescription(Ref("#/$defs/duration"), "Duration after which the workflow times out."),
		},
		"after",
	)
	inlineTimeout.AdditionalProperties = false

	return WithDescription(
		AnyOf(
			WithDescription(String(), "Reference to a named timeout defined in the use block."),
			WithDescription(inlineTimeout, "Inline timeout with an after duration."),
		),
		"Workflow-level timeout. Applies as the default startToCloseTimeout for all activities.",
	)
}

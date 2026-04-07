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

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

func BuildSchema(version, format string) (*jsonschema.Schema, error) {
	schemaID := fmt.Sprintf(SchemaIDVersioned, version, format)
	if version == "development" {
		schemaID = fmt.Sprintf(SchemaID, format)
	}
	schema := &jsonschema.Schema{
		ID:          schemaID,
		Schema:      SchemaVersion,
		Title:       SchemaTitle,
		Description: SchemaDescription,
		Type:        "object",
		Required:    []string{"do", "document"},
		Properties:  schemaProperties,
		Defs:        buildDefinitions(),
	}

	// Validate the generated schema
	if _, err := schema.Resolve(nil); err != nil {
		return nil, fmt.Errorf("error resolving schema: %w", err)
	}

	return schema, nil
}

var schemaProperties = map[string]*jsonschema.Schema{
	"do": {
		Ref:         SchemaRef("taskList"),
		Title:       "Do",
		Description: "Defines the task(s) the workflow must perform.",
	},
	"document": {
		Type:                  "object",
		Title:                 "Document",
		Description:           "Documents the workflow",
		Required:              []string{"dsl", "taskQueue", "workflowType", "version"},
		UnevaluatedProperties: falseSchema(),
		Properties: map[string]*jsonschema.Schema{
			"dsl": {
				Type:        "string",
				Title:       "WorkflowDSL",
				Pattern:     semVerPattern,
				Description: "The version of the DSL used by the workflow.",
			},
			"metadata": {
				Type:                 "object",
				Title:                "WorkflowMetadata",
				Description:          "Holds additional information about the workflow.",
				AdditionalProperties: trueSchema(),
			},
			"taskQueue": {
				Type:        "string",
				Title:       "WorkflowTaskQueue",
				Pattern:     dnsLabelPattern,
				Description: "The Temporal task queue the workflow runs on.",
			},
			"workflowType": {
				Type:        "string",
				Title:       "WorkflowType",
				Pattern:     dnsLabelPattern,
				Description: "The Temporal workflow type name.",
			},
			"summary": {
				Type:        "string",
				Title:       "WorkflowSummary",
				Description: "The workflow's Markdown summary.",
			},
			"tags": {
				Type:                 "object",
				Title:                "WorkflowTags",
				Description:          "A key/value mapping of the workflow's tags, if any.",
				AdditionalProperties: &jsonschema.Schema{},
			},
			"title": {
				Type:        "string",
				Title:       "WorkflowTitle",
				Description: "The workflow's title.",
			},
			"version": {
				Type:        "string",
				Title:       "WorkflowVersion",
				Pattern:     semVerPattern,
				Description: "The workflow's semantic version.",
			},
		},
	},
	"input": {
		Ref:         SchemaRef("input"),
		Title:       "Input",
		Description: "Configures the workflow's input.",
	},
	"output": {
		Ref:         SchemaRef("output"),
		Title:       "Output",
		Description: "Configures the workflow's output.",
	},
	"schedule": {
		Type:                  "object",
		Title:                 "Schedule",
		Description:           "Schedules the workflow.",
		UnevaluatedProperties: falseSchema(),
		Properties: map[string]*jsonschema.Schema{
			"cron": {
				Type:        "string",
				Title:       "ScheduleCron",
				Description: "Specifies the schedule using a cron expression, e.g., '0 0 * * *' for daily at midnight.",
			},
			"every": {
				Ref:         SchemaRef("duration"),
				Title:       "ScheduleEvery",
				Description: "Specifies the duration of the interval at which the workflow should be executed.",
			},
		},
	},
	"timeout": {
		Title:      "DoTimeout",
		Deprecated: true,
		OneOf: []*jsonschema.Schema{
			{
				Ref:         SchemaRef("timeout"),
				Title:       "TimeoutDefinition",
				Description: "The workflow's timeout configuration, if any.",
			},
		},
	},
}

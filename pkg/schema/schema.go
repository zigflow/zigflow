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
		Type:        typeObject,
		Required:    []string{propDo, propDocument},
		Properties:  schemaProperties,
		Defs:        buildDefinitions(),
		If: &jsonschema.Schema{
			Required: []string{"schedule"},
		},
		Then: &jsonschema.Schema{
			Required: []string{propDocument},
			Properties: map[string]*jsonschema.Schema{
				propDocument: {
					Required: []string{propMetadata},
					Properties: map[string]*jsonschema.Schema{
						propMetadata: {
							Required: []string{propScheduleWorkflowName},
						},
					},
				},
			},
		},
	}

	// Validate the generated schema
	if _, err := schema.Resolve(nil); err != nil {
		return nil, fmt.Errorf("error resolving schema: %w", err)
	}

	return schema, nil
}

var schemaProperties = map[string]*jsonschema.Schema{
	propDo: {
		Ref:         SchemaRef("taskList"),
		Title:       "Do",
		Description: "Defines the task(s) the workflow must perform.",
	},
	propDocument: {
		Type:                  typeObject,
		Title:                 "Document",
		Description:           "Documents the workflow",
		Required:              []string{propDSL, propTaskQueue, propWorkflowType, propVersion},
		UnevaluatedProperties: falseSchema(),
		Properties: map[string]*jsonschema.Schema{
			propDSL: {
				Type:        typeString,
				Title:       "WorkflowDSL",
				Pattern:     semVerPattern,
				Description: "The version of the DSL used by the workflow.",
			},
			propMetadata: {
				Type:                 typeObject,
				Title:                "WorkflowMetadata",
				Description:          "Holds additional information about the workflow.",
				AdditionalProperties: trueSchema(),
				AllOf: []*jsonschema.Schema{
					{Ref: SchemaRef(defCommonMetadata)},
					{Ref: SchemaRef(defDocumentMetadata)},
				},
			},
			propTaskQueue: {
				Type:        typeString,
				Title:       "WorkflowTaskQueue",
				Pattern:     dnsLabelPattern,
				Description: "The Temporal task queue the workflow runs on.",
			},
			propWorkflowType: {
				Type:        typeString,
				Title:       "WorkflowType",
				Pattern:     dnsLabelPattern,
				Description: "The Temporal workflow type name.",
			},
			propSummary: {
				Type:        typeString,
				Title:       "WorkflowSummary",
				Description: "The workflow's Markdown summary.",
			},
			"tags": {
				Type:                 typeObject,
				Title:                "WorkflowTags",
				Description:          "A key/value mapping of the workflow's tags, if any.",
				AdditionalProperties: &jsonschema.Schema{},
			},
			"title": {
				Type:        typeString,
				Title:       "WorkflowTitle",
				Description: "The workflow's title.",
			},
			propVersion: {
				Type:        typeString,
				Title:       "WorkflowVersion",
				Pattern:     semVerPattern,
				Description: "The workflow's semantic version.",
			},
		},
	},
	propInput: {
		Ref:         SchemaRef(propInput),
		Title:       "Input",
		Description: "Configures the workflow's input.",
	},
	propOutput: {
		Ref:         SchemaRef(defOutput),
		Title:       "Output",
		Description: "Configures the workflow's output.",
	},
	"schedule": {
		Type:                  typeObject,
		Title:                 "Schedule",
		Description:           "Schedules the workflow.",
		UnevaluatedProperties: falseSchema(),
		Properties: map[string]*jsonschema.Schema{
			propCron: {
				Type:        typeString,
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
	defTimeout: {
		Title:      "DoTimeout",
		Deprecated: true,
		OneOf: []*jsonschema.Schema{
			{
				Ref:         SchemaRef(defTimeout),
				Title:       "TimeoutDefinition",
				Description: "The workflow's timeout configuration, if any.",
			},
		},
	},
}

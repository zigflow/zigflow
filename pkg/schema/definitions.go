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
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/zigflow/zigflow/pkg/utils"
)

func buildDefinitions() map[string]*jsonschema.Schema {
	return map[string]*jsonschema.Schema{
		"callTask":                 callTaskDefinition,
		"commonMetadata":           commonMetadataDefinition,
		"containerLifetime":        containerLifetimeDefinition,
		"doTask":                   doTaskDefinition,
		"documentMetadata":         documentMetadataDefinition,
		"duration":                 durationDefinition,
		"endpoint":                 endpointDefinition,
		"error":                    errorDefinition,
		"eventConsumptionStrategy": eventConsumptionStrategyDefinition,
		"eventFilter":              eventFilterDefinition,
		"eventProperties":          eventPropertiesDefinition,
		"export":                   exportDefinition,
		"externalResource":         externalResourceDefinition,
		"flowDirective":            flowDirectiveDefinition,
		"forkTask":                 forkTaskDefinition,
		"forTask":                  forTaskDefinition,
		"input":                    inputDefinition,
		"listenTask":               listenTaskDefinition,
		"output":                   outputDefinition,
		"raiseTask":                raiseTaskDefinition,
		"runTask":                  runTaskDefinition,
		"runtimeExpression":        runtimeExpressionDefinition,
		"schema":                   schemaDefinition,
		"setTask":                  setTaskDefinition,
		"subscriptionIterator":     subscriptionIteratorDefinition,
		"switchTask":               switchTaskDefinition,
		"task":                     taskDefinition,
		"taskBase":                 taskBaseDefinition,
		"taskList":                 taskListDefinition,
		"taskMetadata":             taskMetadataDefinition,
		"timeout":                  timeoutDefinition,
		"tryTask":                  tryTaskDefinition,
		"uriTemplate":              uriTemplateDefinition,
		"waitTask":                 waitTaskDefinition,
	}
}

var callTaskDefinition = &jsonschema.Schema{
	Title:       "CallTask",
	Description: "Defines the call to perform.",
	OneOf: []*jsonschema.Schema{
		{
			Type:                  "object",
			Title:                 "CallActivity",
			Description:           "Defines the external Temporal activity call to perform.",
			UnevaluatedProperties: falseSchema(),
			Required:              []string{"call", "with"},
			AllOf: []*jsonschema.Schema{
				{Ref: SchemaRef("taskBase")},
				{
					Properties: map[string]*jsonschema.Schema{
						"call": {
							Type:  "string",
							Const: utils.Ptr[any]("activity"),
						},
						"with": {
							Type:                  "object",
							Title:                 "ActivityArguments",
							Description:           "The activity call arguments.",
							UnevaluatedProperties: falseSchema(),
							Required:              []string{"name", "taskQueue"},
							Properties: map[string]*jsonschema.Schema{
								"name": {
									Type:        "string",
									Title:       "WithActivityName",
									Description: "The name of the activity to call on the defined Activity service.",
								},
								"taskQueue": {
									Type:        "string",
									Title:       "WithActivityTaskQueue",
									Description: "The name of the task queue to call on the defined Activity service.",
								},
								"arguments": {
									Type: "array",
								},
							},
						},
					},
				},
			},
		},
		{
			Type:                  "object",
			Title:                 "CallGRPC",
			Description:           "Defines the GRPC call to perform.",
			UnevaluatedProperties: falseSchema(),
			Required:              []string{"call", "with"},
			AllOf: []*jsonschema.Schema{
				{Ref: SchemaRef("taskBase")},
				{
					Properties: map[string]*jsonschema.Schema{
						"call": {
							Type:  "string",
							Const: utils.Ptr[any]("grpc"),
						},
						"with": {
							Type:                  "object",
							Title:                 "GRPCArguments",
							Description:           "The GRPC call arguments.",
							UnevaluatedProperties: falseSchema(),
							Required:              []string{"proto", "service", "method"},
							Properties: map[string]*jsonschema.Schema{
								"arguments": {
									Type:                 "object",
									Title:                "WithGRPCArguments",
									Description:          "The arguments, if any, to call the method with.",
									AdditionalProperties: trueSchema(),
								},
								"method": {
									Type:        "string",
									Title:       "WithGRPCMethod",
									Description: "The name of the method to call on the defined GRPC service.",
								},
								"proto": {
									Ref:         SchemaRef("externalResource"),
									Title:       "WithGRPCProto",
									Description: "The proto resource that describes the GRPC service to call.",
								},
								"service": {
									Type:                  "object",
									Title:                 "WithGRPCService",
									UnevaluatedProperties: falseSchema(),
									Required:              []string{"name", "host"},
									Properties: map[string]*jsonschema.Schema{
										"host": {
											Type:        "string",
											Title:       "WithGRPCServiceHost",
											Description: "The hostname of the GRPC service to call.",
											Pattern:     domainNamePattern,
										},
										"name": {
											Type:        "string",
											Title:       "WithGRPCServiceName",
											Description: "The name of the GRPC service to call.",
										},
										"port": {
											Type:        "integer",
											Title:       "WithGRPCServicePort",
											Description: "The port number of the GRPC service to call.",
											Minimum:     utils.Ptr(float64(0)),
											Maximum:     utils.Ptr(float64(65535)),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Type:                  "object",
			Title:                 "CallHTTP",
			Description:           "Defines the HTTP call to perform.",
			UnevaluatedProperties: falseSchema(),
			Required:              []string{"call", "with"},
			AllOf: []*jsonschema.Schema{
				{Ref: SchemaRef("taskBase")},
				{
					Properties: map[string]*jsonschema.Schema{
						"call": {
							Type:  "string",
							Const: utils.Ptr[any]("http"),
						},
						"with": {
							Type:                  "object",
							Title:                 "HTTPArguments",
							Description:           "The HTTP call arguments.",
							UnevaluatedProperties: falseSchema(),
							Required:              []string{"method", "endpoint"},
							Properties: map[string]*jsonschema.Schema{
								"body": {
									Title:       "HTTPBody",
									Description: "The body, if any, of the HTTP request to perform.",
								},
								"endpoint": {
									Ref:         SchemaRef("endpoint"),
									Title:       "HTTPEndpoint",
									Description: "The HTTP endpoint to send the request to.",
								},
								"headers": {
									Title:       "HTTPHeaders",
									Description: "A name/value mapping of the headers, if any, of the HTTP request to perform.",
									OneOf: []*jsonschema.Schema{
										{
											Type: "object",
											AdditionalProperties: &jsonschema.Schema{
												Type: "string",
											},
										},
										{Ref: SchemaRef("runtimeExpression")},
									},
								},
								"method": {
									Type:        "string",
									Title:       "HTTPMethod",
									Description: "The HTTP method of the HTTP request to perform.",
								},
								"output": {
									Type:        "string",
									Title:       "HTTPOutput",
									Description: "The http call output format. Defaults to 'content'.",
									Enum:        []any{"raw", "content", "response"},
								},
								"query": {
									Title:                "HTTPQuery",
									Description:          "A name/value mapping of the query parameters, if any, of the HTTP request to perform.",
									AdditionalProperties: trueSchema(),
									OneOf: []*jsonschema.Schema{
										{
											Type: "object",
											AdditionalProperties: &jsonschema.Schema{
												Type: "string",
											},
										},
										{Ref: SchemaRef("runtimeExpression")},
									},
								},
								"redirect": {
									Type:        "boolean",
									Title:       "HttpRedirect",
									Description: "Specifies whether redirection status codes (`300-399`) should be treated as errors.",
								},
							},
						},
					},
				},
			},
		},
	},
}

var commonMetadataDefinition = &jsonschema.Schema{
	Type:                 "object",
	Title:                "CommonMetadata",
	AdditionalProperties: trueSchema(),
	Properties: map[string]*jsonschema.Schema{
		"activityOptions": {
			Type:                 "object",
			Title:                "ActivityOptionsMetadata",
			AdditionalProperties: trueSchema(),
			Properties: map[string]*jsonschema.Schema{
				"disableEagerExecution": {
					Type: "boolean",
					Description: "If true, eager execution will not be requested, regardless of worker settings. " +
						"If false, eager execution may still be disabled at the worker level or may not be requested due to lack of available slots.",
				},
				"heartbeatTimeout": {
					Ref:         SchemaRef("duration"),
					Title:       "HeartbeatTimeout",
					Description: "Heartbeat interval. A heartbeat must be set and be called before the interval passes.",
				},
				"priority": {
					Type:                 "object",
					Title:                "Priority",
					Description:          "Configure an activity's priority and fairness",
					AdditionalProperties: falseSchema(),
					Properties: map[string]*jsonschema.Schema{
						"fairnessKey": {
							Type:        "string",
							Title:       "FairnessKey",
							Description: "A short string that's used as a key for a fairness balancing mechanism",
						},
						"fairnessWeight": {
							Type:        "number",
							Title:       "FairnessWeight",
							Description: "Weight of a task can come from multiple sources for flexibility",
						},
						"priorityKey": {
							Type:        "integer",
							Title:       "PriorityKey",
							Description: "A positive integer from 1 to n, where smaller integers correspond to higher priorities (tasks run sooner)",
							Minimum:     utils.Ptr[float64](1),
						},
					},
				},
				"retryPolicy": {
					Type:                 "object",
					Title:                "RetryPolicy",
					Description:          "Specifies how to retry an Activity if an error occurs",
					AdditionalProperties: falseSchema(),
					Properties: map[string]*jsonschema.Schema{
						"backoffCoefficient": {
							Type:        "number",
							Title:       "BackoffCoefficient",
							Description: "Coefficient used to calculate the next retry backoff interval.",
							Default:     json.RawMessage("2.0"),
						},
						"initialInterval": {
							Ref:         SchemaRef("duration"),
							Title:       "InitialInterval",
							Description: "Backoff interval for the first retry. If BackoffCoefficient is 1.0 then it is used for all retries.",
						},
						"maximumAttempts": {
							Type:        "integer",
							Title:       "MaximumAttempts",
							Description: "Maximum number of attempts. When exceeded the retries stop even if not expired yet.",
							Default:     json.RawMessage("5"),
						},
						"maximumInterval": {
							Ref:         SchemaRef("duration"),
							Title:       "MaximumInterval",
							Description: "Maximum backoff interval between retries.",
						},
						"nonRetryableErrorTypes": {
							Type:        "array",
							Title:       "NonRetryableErrorTypes",
							Description: "Temporal server will stop retry if error type matches this list.",
							Default:     json.RawMessage("[]"),
							Items: &jsonschema.Schema{
								Type: "string",
							},
						},
					},
				},
				"scheduleToCloseTimeout": {
					Ref:         SchemaRef("duration"),
					Title:       "ScheduleToCloseTimeout",
					Description: "Total time that a workflow is willing to wait for an Activity to complete.",
				},
				"scheduleToStartTimeout": {
					Ref:   SchemaRef("duration"),
					Title: "ScheduleToStartTimeout",
					Description: "Time that the Activity Task can stay in the Task Queue before it is picked up by a Worker. " +
						"Do not specify this timeout unless using host specific Task Queues for Activity Tasks are being used for routing.",
				},
				"startToCloseTimeout": {
					Ref:         SchemaRef("duration"),
					Title:       "StartToCloseTimeout",
					Description: "Maximum time of a single Activity execution attempt.",
					Default:     json.RawMessage(`{"seconds": 15}`),
				},
				"summary": {
					Type:        "string",
					Description: "Add a summary to the Temporal workflow UI.",
				},
			},
		},
	},
}

var containerLifetimeDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "ContainerLifetime",
	Description:           "The configuration of a container's lifetime",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"cleanup"},
	Properties: map[string]*jsonschema.Schema{
		"cleanup": {
			Type:        "string",
			Title:       "ContainerCleanupPolicy",
			Description: "The container cleanup policy to use",
			Enum:        []any{"always", "never"},
			Default:     json.RawMessage(`"always"`),
		},
	},
}

var doTaskDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "DoTask",
	Description:           "Allows to execute a list of tasks in sequence.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"do"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"do": {
					Ref:         SchemaRef("taskList"),
					Title:       "DoTaskConfiguration",
					Description: "The configuration of the tasks to perform sequentially.",
				},
			},
		},
	},
}

var documentMetadataDefinition = &jsonschema.Schema{
	Type:                 "object",
	Title:                "DocumentMetadata",
	AdditionalProperties: trueSchema(),
	Properties: map[string]*jsonschema.Schema{
		"canMaxHistoryLength": {
			Type:        "integer",
			Title:       "ContinueAsNewMaxHistoryLength",
			Description: "Allows you to test the Continue-As-New functionality by specifying the max history length before triggering.",
		},
		"scheduleWorkflowName": {
			Type:        "string",
			Title:       "ScheduleWorkflowName",
			Description: "Set the workflow name to trigger - this will either be the document.workflowType or the Do task",
			MinLength:   utils.Ptr(1),
		},
		"scheduleId": {
			Type:        "string",
			Title:       "ScheduleID",
			Description: "Set the schedule ID. If not set, this will to zigflow_<workflow.document.workflowType>",
		},
		"scheduleInput": {
			Type:        "array",
			Title:       "ScheduleInput",
			Description: "Set the input.",
		},
	},
}

var durationDefinition = &jsonschema.Schema{
	OneOf: []*jsonschema.Schema{
		{
			Type:                  "object",
			MinProperties:         utils.Ptr(1),
			UnevaluatedProperties: falseSchema(),
			Properties: map[string]*jsonschema.Schema{
				"days": {
					Type:        "integer",
					Title:       "DurationDays",
					Description: "Number of days, if any.",
				},
				"hours": {
					Type:        "integer",
					Title:       "DurationHours",
					Description: "Number of hours, if any.",
				},
				"minutes": {
					Type:        "integer",
					Title:       "DurationMinutes",
					Description: "Number of minutes, if any.",
				},
				"seconds": {
					Type:        "integer",
					Title:       "DurationSeconds",
					Description: "Number of seconds, if any.",
				},
				"milliseconds": {
					Type:        "integer",
					Title:       "DurationMilliseconds",
					Description: "Number of milliseconds, if any.",
				},
			},
		},
	},
}

var endpointDefinition = &jsonschema.Schema{
	Title:       "Endpoint",
	Description: "Represents an endpoint.",
	OneOf: []*jsonschema.Schema{
		{Ref: SchemaRef("runtimeExpression")},
		{Ref: SchemaRef("uriTemplate")},
		{
			Type:                  "object",
			Title:                 "EndpointConfiguration",
			UnevaluatedProperties: falseSchema(),
			Required:              []string{"uri"},
			Properties: map[string]*jsonschema.Schema{
				"uri": {
					Title:       "EndpointUri",
					Description: "The endpoint's URI.",
					OneOf: []*jsonschema.Schema{
						{
							Ref:         SchemaRef("uriTemplate"),
							Title:       "LiteralEndpointURI",
							Description: "The literal endpoint's URI.",
						},
						{
							Ref:         SchemaRef("runtimeExpression"),
							Title:       "ExpressionEndpointURI",
							Description: "An expression based endpoint's URI.",
						},
					},
				},
			},
		},
	},
}

var errorDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "Error",
	Description:           "Represents an error.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"type", "status"},
	Properties: map[string]*jsonschema.Schema{
		"detail": {
			Title:       "ErrorDetails",
			Description: "A human-readable explanation specific to this occurrence of the error.",
			AnyOf: []*jsonschema.Schema{
				{
					Ref:   SchemaRef("runtimeExpression"),
					Title: "ExpressionErrorDetails",
				},
				{
					Type:  "string",
					Title: "LiteralErrorDetails",
				},
			},
		},
		"instance": {
			Title:       "ErrorInstance",
			Description: "A JSON Pointer used to reference the component the error originates from.",
			OneOf: []*jsonschema.Schema{
				{
					Type:        "string",
					Title:       "LiteralErrorInstance",
					Description: "The literal error instance.",
					Format:      "json-pointer",
				},
				{
					Ref:         SchemaRef("runtimeExpression"),
					Title:       "ExpressionErrorInstance",
					Description: "An expression based error instance.",
				},
			},
		},
		"status": {
			Type:        "integer",
			Title:       "ErrorStatus",
			Description: "The status code generated by the origin for this occurrence of the error.",
		},
		"title": {
			Title:       "ErrorTitle",
			Description: "A short, human-readable summary of the error.",
			AnyOf: []*jsonschema.Schema{
				{
					Ref:   SchemaRef("runtimeExpression"),
					Title: "ExpressionErrorTitle",
				},
				{
					Type:  "string",
					Title: "LiteralErrorTitle",
				},
			},
		},
		"type": {
			Title:       "ErrorType",
			Description: "A URI reference that identifies the error type.",
			OneOf: []*jsonschema.Schema{
				{
					Ref:         SchemaRef("uriTemplate"),
					Title:       "LiteralErrorType",
					Description: "The literal error type.",
				},
				{
					Ref:         SchemaRef("runtimeExpression"),
					Title:       "ExpressionErrorType",
					Description: "An expression based error type.",
				},
			},
		},
	},
}

var eventConsumptionStrategyDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "EventConsumptionStrategy",
	Description:           "Describe the event consumption strategy to adopt.",
	UnevaluatedProperties: falseSchema(),
	OneOf: []*jsonschema.Schema{
		{
			Title:    "AllEventConsumptionStrategy",
			Required: []string{"all"},
			Properties: map[string]*jsonschema.Schema{
				"all": {
					Type:        "array",
					Title:       "AllEventConsumptionStrategyConfiguration",
					Description: "A list containing all the events that must be consumed.",
					Items: &jsonschema.Schema{
						Ref: SchemaRef("eventFilter"),
					},
				},
			},
		},
		{
			Title:    "AnyEventConsumptionStrategy",
			Required: []string{"any"},
			Properties: map[string]*jsonschema.Schema{
				"any": {
					Type:        "array",
					Title:       "AnyEventConsumptionStrategyConfiguration",
					Description: "A list containing any of the events to consume.",
					Items: &jsonschema.Schema{
						Ref: SchemaRef("eventFilter"),
					},
				},
			},
		},
		{
			Title:    "OneEventConsumptionStrategy",
			Required: []string{"one"},
			Properties: map[string]*jsonschema.Schema{
				"one": {
					Ref:         SchemaRef("eventFilter"),
					Title:       "OneEventConsumptionStrategyConfiguration",
					Description: "The single event to consume.",
				},
			},
		},
	},
}

var eventFilterDefinition = &jsonschema.Schema{
	Type:  "object",
	Title: "EventFilter",
	Description: "An event filter is a mechanism used to selectively process or handle events " +
		"based on predefined criteria, such as event type, source, or specific attributes.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"with"},
	Properties: map[string]*jsonschema.Schema{
		"with": {
			Ref:   SchemaRef("eventProperties"),
			Title: "WithEvent",
			Description: "An event filter is a mechanism used to selectively process or handle events " +
				"based on predefined criteria, such as event type, source, or specific attributes.",
			MinProperties: utils.Ptr(1),
		},
	},
}

var eventPropertiesDefinition = &jsonschema.Schema{
	Type:                 "object",
	Title:                "EventProperties",
	Description:          "Describes the properties of an event.",
	AdditionalProperties: trueSchema(),
	Properties: map[string]*jsonschema.Schema{
		"data": {
			Title:       "EventData",
			Description: "The event's payload data",
			AnyOf: []*jsonschema.Schema{
				{Ref: SchemaRef("runtimeExpression")},
				trueSchema(),
			},
		},
		"datacontenttype": {
			Type:  "string",
			Title: "EventDataContentType",
			Description: "Content type of data value. This attribute enables data to carry any type of content, " +
				"whereby format and encoding might differ from that of the chosen event format.",
		},
		"dataschema": {
			Title:       "EventDataschema",
			Description: "The schema describing the event format.",
			OneOf: []*jsonschema.Schema{
				{
					Ref:         SchemaRef("uriTemplate"),
					Title:       "LiteralDataSchema",
					Description: "The literal event data schema.",
				},
				{
					Ref:         SchemaRef("runtimeExpression"),
					Title:       "ExpressionDataSchema",
					Description: "An expression based event data schema.",
				},
			},
		},
		"id": {
			Type:        "string",
			Title:       "EventId",
			Description: "The event's unique identifier.",
		},
		"source": {
			Title:       "EventSource",
			Description: "Identifies the context in which an event happened.",
			OneOf: []*jsonschema.Schema{
				{Ref: SchemaRef("uriTemplate")},
				{Ref: SchemaRef("runtimeExpression")},
			},
		},
		"subject": {
			Type:        "string",
			Title:       "EventSubject",
			Description: "The subject of the event.",
		},
		"time": {
			Title:       "EventTime",
			Description: "When the event occurred.",
			OneOf: []*jsonschema.Schema{
				{
					Type:   "string",
					Title:  "LiteralTime",
					Format: "date-time",
				},
				{Ref: SchemaRef("runtimeExpression")},
			},
		},
		"type": {
			Type:        "string",
			Title:       "EventType",
			Description: "This attribute contains a value describing the type of event related to the originating occurrence.",
		},
	},
}

var exportDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "Export",
	Description:           "Set the content of the context.",
	UnevaluatedProperties: falseSchema(),
	Properties: map[string]*jsonschema.Schema{
		"schema": {
			Ref:         SchemaRef("schema"),
			Title:       "ExportSchema",
			Description: "The schema used to describe and validate the workflow context.",
		},
		"as": {
			Title:       "ExportAs",
			Description: "A runtime expression, if any, used to export the output data to the context.",
			OneOf: []*jsonschema.Schema{
				{Type: "string"},
				{Type: "object"},
			},
		},
	},
}

var externalResourceDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "ExternalResource",
	Description:           "Represents an external resource.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"endpoint"},
	Properties: map[string]*jsonschema.Schema{
		"endpoint": {
			Ref:         SchemaRef("endpoint"),
			Title:       "ExternalResourceEndpoint",
			Description: "The endpoint of the external resource.",
		},
		"name": {
			Type:        "string",
			Title:       "ExternalResourceName",
			Description: "The name of the external resource, if any.",
		},
	},
}

var flowDirectiveDefinition = &jsonschema.Schema{
	Title:       "FlowDirective",
	Description: "Represents different transition options for a workflow.",
	AnyOf: []*jsonschema.Schema{
		{
			Title:   "FlowDirectiveEnum",
			Type:    "string",
			Enum:    []any{"continue", "exit", "end"},
			Default: json.RawMessage(`"continue"`),
		},
		{
			Type: "string",
		},
	},
}

var forkTaskDefinition = &jsonschema.Schema{
	Type:  "object",
	Title: "ForkTask",
	Description: "Allows workflows to execute multiple tasks concurrently and optionally race them against each other, " +
		"with a single possible winner, which sets the task's output.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"fork"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"fork": {
					Type:                  "object",
					Title:                 "ForkTaskConfiguration",
					Description:           "The configuration of the branches to perform concurrently.",
					UnevaluatedProperties: falseSchema(),
					Required:              []string{"branches"},
					Properties: map[string]*jsonschema.Schema{
						"branches": {
							Ref:   SchemaRef("taskList"),
							Title: "ForkBranches",
						},
						"compete": {
							Type:  "boolean",
							Title: "ForkCompete",
							Description: "Indicates whether or not the concurrent tasks are racing against each other, " +
								"with a single possible winner, which sets the composite task's output.",
							Default: json.RawMessage(`false`),
						},
					},
				},
			},
		},
	},
}

var forTaskDefinition = &jsonschema.Schema{
	Type:  "object",
	Title: "ForTask",
	Description: "Allows workflows to iterate over a collection of items, executing a defined set of subtasks for each item " +
		"in the collection. This task type is instrumental in handling scenarios such as batch processing, " +
		"data transformation, and repetitive operations across datasets.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"for", "do"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"do": {
					Ref:   SchemaRef("taskList"),
					Title: "ForTaskDo",
				},
				"for": {
					Type:                  "object",
					Title:                 "ForTaskConfiguration",
					Description:           "The definition of the loop that iterates over a range of values.",
					UnevaluatedProperties: falseSchema(),
					Required:              []string{"in"},
					Properties: map[string]*jsonschema.Schema{
						"at": {
							Type:        "string",
							Title:       "ForAt",
							Description: "The name of the variable used to store the index of the current item being enumerated.",
							Default:     json.RawMessage(`"index"`),
						},
						"each": {
							Type:        "string",
							Title:       "ForEach",
							Description: "The name of the variable used to store the current item being enumerated.",
							Default:     json.RawMessage(`"item"`),
						},
						"in": {
							Type:        "string",
							Title:       "ForIn",
							Description: "A runtime expression used to get the collection to enumerate.",
						},
					},
				},
				"while": {
					Type:        "string",
					Title:       "While",
					Description: "A runtime expression that represents the condition, if any, that must be met for the iteration to continue.",
				},
			},
		},
	},
}

var inputDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "Input",
	Description:           "Configures the input of a workflow or task.",
	UnevaluatedProperties: falseSchema(),
	Properties: map[string]*jsonschema.Schema{
		"schema": {
			Ref:         SchemaRef("schema"),
			Title:       "InputSchema",
			Description: "The schema used to describe and validate the input of the workflow or task.",
		},
	},
}

var listenTaskDefinition = &jsonschema.Schema{
	Type:  "object",
	Title: "ListenTask",
	Description: "Provides a mechanism for workflows to await and react to external events, " +
		"enabling event-driven behaviour within workflow systems.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"listen"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"listen": {
					Type:                  "object",
					Title:                 "ListenTaskConfiguration",
					Description:           "The configuration of the listener to use.",
					UnevaluatedProperties: falseSchema(),
					Required:              []string{"to"},
					Properties: map[string]*jsonschema.Schema{
						"read": {
							Type:        "string",
							Title:       "ListenAndReadAs",
							Description: "Specifies how events are read during the listen operation.",
							Enum:        []any{"data", "envelope", "raw"},
							Default:     json.RawMessage(`"data"`),
						},
						"to": {
							Ref:         SchemaRef("eventConsumptionStrategy"),
							Title:       "ListenTo",
							Description: "Defines the event(s) to listen to.",
						},
					},
				},
			},
		},
	},
}

var outputDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "Output",
	Description:           "Configures the output of a workflow or task.",
	UnevaluatedProperties: falseSchema(),
	Properties: map[string]*jsonschema.Schema{
		"schema": {
			Ref:         SchemaRef("schema"),
			Title:       "OutputSchema",
			Description: "The schema used to describe and validate the output of the workflow or task.",
		},
		"as": {
			Title:       "OutputAs",
			Description: "A runtime expression, if any, used to mutate and/or filter the output of the workflow or task.",
			OneOf: []*jsonschema.Schema{
				{Type: "string"},
				{Type: "object"},
			},
		},
	},
}

var raiseTaskDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "RaiseTask",
	Description:           "Intentionally triggers and propagates errors.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"raise"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"raise": {
					Type:                  "object",
					Title:                 "RaiseTaskConfiguration",
					Description:           "The definition of the error to raise.",
					UnevaluatedProperties: falseSchema(),
					Required:              []string{"error"},
					Properties: map[string]*jsonschema.Schema{
						"error": {
							Title: "RaiseTaskError",
							OneOf: []*jsonschema.Schema{
								{
									Ref:         SchemaRef("error"),
									Title:       "RaiseErrorDefinition",
									Description: "Defines the error to raise.",
								},
								{
									Type:        "string",
									Title:       "RaiseErrorReference",
									Description: "The name of the error to raise",
								},
							},
						},
					},
				},
			},
		},
	},
}

var runTaskDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "RunTask",
	Description:           "Provides the capability to execute external containers, shell commands, scripts, or workflows.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"run"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"run": {
					Type:                  "object",
					Title:                 "RunTaskConfiguration",
					Description:           "The configuration of the process to execute.",
					UnevaluatedProperties: falseSchema(),
					Properties: map[string]*jsonschema.Schema{
						"await": {
							Type:        "boolean",
							Title:       "AwaitProcessCompletion",
							Description: "Whether to await the process completion before continuing.",
							Default:     json.RawMessage(`true`),
						},
					},
					OneOf: []*jsonschema.Schema{
						{
							Title:       "RunContainer",
							Description: "Enables the execution of external processes encapsulated within a containerized environment.",
							Required:    []string{"container"},
							Properties: map[string]*jsonschema.Schema{
								"container": {
									Type:                  "object",
									Title:                 "Container",
									Description:           "The configuration of the container to run.",
									UnevaluatedProperties: falseSchema(),
									Required:              []string{"image"},
									Properties: map[string]*jsonschema.Schema{
										"arguments": {
											Type:        "array",
											Title:       "ContainerArguments",
											Description: "A list of the arguments, if any, passed as argv to the command or default container CMD",
											Items:       &jsonschema.Schema{Type: "string"},
										},
										"command": {
											Type:        "string",
											Title:       "ContainerCommand",
											Description: "The command, if any, to execute on the container.",
										},
										"environment": {
											Type:        "object",
											Title:       "ContainerEnvironment",
											Description: "A key/value mapping of the environment variables, if any, to use when running the configured process.",
										},
										"image": {
											Type:        "string",
											Title:       "ContainerImage",
											Description: "The name of the container image to run.",
										},
										"lifetime": {
											Ref:         SchemaRef("containerLifetime"),
											Title:       "ContainerLifetime",
											Description: "An object, if any, used to configure the container's lifetime",
										},
										"name": {
											Type:        "string",
											Title:       "ContainerName",
											Description: "A runtime expression, if any, used to give specific name to the container.",
										},
										"volumes": {
											Type:        "object",
											Title:       "ContainerVolumes",
											Description: "The container's volume mappings, if any.",
										},
									},
								},
							},
						},
						{
							Title: "RunScript",
							Description: "Enables the execution of custom scripts or code within a workflow, empowering workflows to perform " +
								"specialised logic, data processing, or integration tasks by executing user-defined scripts " +
								"written in various programming languages.",
							Required: []string{"script"},
							Properties: map[string]*jsonschema.Schema{
								"script": {
									Type:                  "object",
									Title:                 "Script",
									Description:           "The configuration of the script to run.",
									UnevaluatedProperties: falseSchema(),
									Required:              []string{"language"},
									OneOf: []*jsonschema.Schema{
										{
											Title:       "InlineScript",
											Type:        "object",
											Description: "The script's code.",
											Required:    []string{"code"},
											Properties: map[string]*jsonschema.Schema{
												"code": {
													Type:  "string",
													Title: "InlineScriptCode",
												},
											},
										},
									},
									Properties: map[string]*jsonschema.Schema{
										"arguments": {
											Type:        "array",
											Title:       "ScriptArguments",
											Description: "A list of the arguments, if any, to the script as argv",
											Items:       &jsonschema.Schema{Type: "string"},
										},
										"environment": {
											Type:  "object",
											Title: "ScriptEnvironment",
											Description: "A key/value mapping of the environment variables, if any, " +
												"to use when running the configured script process.",
											AdditionalProperties: trueSchema(),
										},
										"language": {
											Type:        "string",
											Title:       "ScriptLanguage",
											Description: "The language of the script to run.",
											Enum:        []any{"js", "python"},
										},
									},
								},
							},
						},
						{
							Title: "RunShell",
							Description: "Enables the execution of shell commands within a workflow, enabling workflows to interact with the " +
								"underlying operating system and perform system-level operations, such as file manipulation, " +
								"environment configuration, or system administration tasks.",
							Required: []string{"shell"},
							Properties: map[string]*jsonschema.Schema{
								"shell": {
									Type:                  "object",
									Title:                 "Shell",
									Description:           "The configuration of the shell command to run.",
									UnevaluatedProperties: falseSchema(),
									Required:              []string{"command"},
									Properties: map[string]*jsonschema.Schema{
										"arguments": {
											Type:        "array",
											Title:       "ShellArguments",
											Description: "A list of the arguments, if any, to the shell command as argv",
											Items:       &jsonschema.Schema{Type: "string"},
										},
										"command": {
											Type:        "string",
											Title:       "ShellCommand",
											Description: "The shell command to run.",
										},
										"environment": {
											Type:                 "object",
											Title:                "ShellEnvironment",
											Description:          "A key/value mapping of the environment variables, if any, to use when running the configured process.",
											AdditionalProperties: trueSchema(),
										},
									},
								},
							},
						},
						{
							Title: "RunWorkflow",
							Description: "Enables the invocation and execution of nested workflows within a parent workflow, facilitating " +
								"modularization, reusability, and abstraction of complex logic or business processes " +
								"by encapsulating them into standalone workflow units.",
							Required: []string{"workflow"},
							Properties: map[string]*jsonschema.Schema{
								"workflow": {
									Type:                  "object",
									Title:                 "SubflowConfiguration",
									Description:           "The configuration of the workflow to run.",
									UnevaluatedProperties: falseSchema(),
									Required:              []string{"type"},
									Properties: map[string]*jsonschema.Schema{
										"input": {
											Type:  "object",
											Title: "SubflowInput",
											Description: "The data, if any, to pass as input to the workflow to execute. " +
												"The value should be validated against the target workflow's input schema, if specified.",
											AdditionalProperties: trueSchema(),
										},
										"type": {
											Type:        "string",
											Title:       "SubflowType",
											Description: "The workflow type to run.",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

var runtimeExpressionDefinition = &jsonschema.Schema{
	Type:        "string",
	Title:       "RuntimeExpression",
	Description: "A runtime expression.",
	Pattern:     runtimeExpressionString,
}

var schemaDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "Schema",
	Description:           "Represents the definition of a schema.",
	UnevaluatedProperties: falseSchema(),
	Properties: map[string]*jsonschema.Schema{
		"format": {
			Type:        "string",
			Title:       "SchemaFormat",
			Default:     json.RawMessage(`"json"`),
			Description: "The schema's format. Defaults to 'json'. The (optional) version of the format can be set using `{format}:{version}`.",
		},
	},
	OneOf: []*jsonschema.Schema{
		{
			Title:    "SchemaInline",
			Required: []string{"document"},
			Properties: map[string]*jsonschema.Schema{
				"document": {
					Description: "The schema's inline definition.",
				},
			},
		},
	},
}

var setTaskDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "SetTask",
	Description:           "A task used to set data.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"set"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"set": {
					Title:       "SetTaskConfiguration",
					Description: "The data to set.",
					OneOf: []*jsonschema.Schema{
						{
							Type:                 "object",
							MinProperties:        utils.Ptr(1),
							AdditionalProperties: trueSchema(),
						},
						{Type: "string"},
					},
				},
			},
		},
	},
}

var subscriptionIteratorDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "SubscriptionIterator",
	Description:           "Configures the iteration over each item (event or message) consumed by a subscription.",
	UnevaluatedProperties: falseSchema(),
	Properties: map[string]*jsonschema.Schema{
		"at": {
			Type:        "string",
			Title:       "SubscriptionIteratorIndex",
			Description: "The name of the variable used to store the index of the current item being enumerated.",
			Default:     json.RawMessage(`"index"`),
		},
		"do": {
			Ref:         SchemaRef("taskList"),
			Title:       "SubscriptionIteratorTasks",
			Description: "The tasks to perform for each consumed item.",
		},
		"export": {
			Ref:         SchemaRef("export"),
			Title:       "SubscriptionIteratorExport",
			Description: "An object, if any, used to customise the content of the workflow context.",
		},
		"item": {
			Type:        "string",
			Title:       "SubscriptionIteratorItem",
			Description: "The name of the variable used to store the current item being enumerated.",
			Default:     json.RawMessage(`"item"`),
		},
		"output": {
			Ref:         SchemaRef("output"),
			Title:       "SubscriptionIteratorOutput",
			Description: "An object, if any, used to customise the item's output and to document its schema.",
		},
	},
}

var switchTaskDefinition = &jsonschema.Schema{
	Type:  "object",
	Title: "SwitchTask",
	Description: "Enables conditional branching within workflows, allowing them to dynamically select " +
		"different paths based on specified conditions or criteria.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"switch"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"switch": {
					Type:        "array",
					Title:       "SwitchTaskConfiguration",
					Description: "The definition of the switch to use.",
					MinItems:    utils.Ptr(1),
					Items: &jsonschema.Schema{
						Type:          "object",
						Title:         "SwitchItem",
						MinProperties: utils.Ptr(1),
						MaxProperties: utils.Ptr(1),
						AdditionalProperties: &jsonschema.Schema{
							Type:  "object",
							Title: "SwitchCase",
							Description: "The definition of a case within a switch task, defining a condition " +
								"and corresponding tasks to execute if the condition is met.",
							UnevaluatedProperties: falseSchema(),
							Required:              []string{"then"},
							Properties: map[string]*jsonschema.Schema{
								"then": {
									Ref:         SchemaRef("flowDirective"),
									Title:       "SwitchCaseOutcome",
									Description: "The flow directive to execute when the case matches.",
								},
								"when": {
									Type:        "string",
									Title:       "SwitchCaseCondition",
									Description: "A runtime expression used to determine whether or not the case matches.",
								},
							},
						},
					},
				},
			},
		},
	},
}

var taskDefinition = &jsonschema.Schema{
	Title:                 "Task",
	Description:           "A discrete unit of work that contributes to achieving the overall objectives defined by the workflow.",
	UnevaluatedProperties: falseSchema(),
	OneOf: []*jsonschema.Schema{
		{Ref: SchemaRef("callTask")},
		{Ref: SchemaRef("doTask")},
		{Ref: SchemaRef("forTask")},
		{Ref: SchemaRef("forkTask")},
		{Ref: SchemaRef("listenTask")},
		{Ref: SchemaRef("raiseTask")},
		{Ref: SchemaRef("runTask")},
		{Ref: SchemaRef("setTask")},
		{Ref: SchemaRef("switchTask")},
		{Ref: SchemaRef("tryTask")},
		{Ref: SchemaRef("waitTask")},
	},
}

var taskBaseDefinition = &jsonschema.Schema{
	Type:        "object",
	Title:       "TaskBase",
	Description: "An object inherited by all tasks.",
	Properties: map[string]*jsonschema.Schema{
		"if": {
			Type:        "string",
			Title:       "TaskBaseIf",
			Description: "A runtime expression, if any, used to determine whether or not the task should be run.",
		},
		"input": {
			Ref:         SchemaRef("input"),
			Title:       "TaskBaseInput",
			Description: "Configure the task's input.",
		},
		"output": {
			Ref:         SchemaRef("output"),
			Title:       "TaskBaseOutput",
			Description: "Configure the task's output.",
		},
		"export": {
			Ref:         SchemaRef("export"),
			Title:       "TaskBaseExport",
			Description: "Export task output to context.",
		},
		"then": {
			Ref:         SchemaRef("flowDirective"),
			Title:       "TaskBaseThen",
			Description: "The flow directive to be performed upon completion of the task.",
		},
		"metadata": {
			Type:                 "object",
			Title:                "TaskBaseMetadata",
			Description:          "Holds additional information about the task.",
			AdditionalProperties: trueSchema(),
			AllOf: []*jsonschema.Schema{
				{Ref: SchemaRef("commonMetadata")},
				{Ref: SchemaRef("taskMetadata")},
			},
		},
	},
}

var taskListDefinition = &jsonschema.Schema{
	Type:        "array",
	Title:       "TaskList",
	Description: "List of named tasks to perform.",
	Items: &jsonschema.Schema{
		Type:          "object",
		Title:         "TaskItem",
		MinProperties: utils.Ptr(1),
		MaxProperties: utils.Ptr(1),
		AdditionalProperties: &jsonschema.Schema{
			Ref: SchemaRef("task"),
		},
	},
}

var taskMetadataDefinition = &jsonschema.Schema{
	Type:                 "object",
	Title:                "TaskMetadata",
	AdditionalProperties: trueSchema(),
	Properties: map[string]*jsonschema.Schema{
		"__zigflow_id": {
			Type:  "string",
			Title: "ZigflowID",
			Description: "A system-generated unique identifier for the task. " +
				"This value is assigned automatically and should not be modified by users.",
		},
		"heartbeat": {
			Ref:         SchemaRef("duration"),
			Title:       "Heartbeat",
			Description: "Heartbeats will be triggered after this time period.",
		},
	},
}

var timeoutDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "Timeout",
	Description:           "The definition of a timeout.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"after"},
	Properties: map[string]*jsonschema.Schema{
		"after": {
			Ref:         SchemaRef("duration"),
			Title:       "TimeoutAfter",
			Description: "The duration after which to timeout.",
		},
	},
}

var tryTaskDefinition = &jsonschema.Schema{
	Type:  "object",
	Title: "TryTask",
	Description: "Serves as a mechanism within workflows to handle errors gracefully, " +
		"potentially retrying failed tasks before proceeding with alternate ones.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"try", "catch"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"catch": {
					Type:                  "object",
					Title:                 "TryTaskCatch",
					Description:           "The object used to define the errors to catch.",
					UnevaluatedProperties: falseSchema(),
					Required:              []string{"do"},
					Properties: map[string]*jsonschema.Schema{
						"do": {
							Ref:         SchemaRef("taskList"),
							Title:       "TryTaskCatchDo",
							Description: "The definition of the task(s) to run when catching an error.",
						},
					},
				},
				"try": {
					Ref:         SchemaRef("taskList"),
					Title:       "TryTaskConfiguration",
					Description: "The task(s) to perform.",
				},
			},
		},
	},
}

var uriTemplateDefinition = &jsonschema.Schema{
	Title: "UriTemplate",
	AnyOf: []*jsonschema.Schema{
		{
			Type:    "string",
			Title:   "LiteralUriTemplate",
			Format:  "uri-template",
			Pattern: urlPattern,
		},
		{
			Type:    "string",
			Title:   "LiteralUri",
			Format:  "uri",
			Pattern: urlPattern,
		},
	},
}

var waitTaskDefinition = &jsonschema.Schema{
	Type:                  "object",
	Title:                 "WaitTask",
	Description:           "Allows workflows to pause or delay their execution for a specified period of time.",
	UnevaluatedProperties: falseSchema(),
	Required:              []string{"wait"},
	AllOf: []*jsonschema.Schema{
		{Ref: SchemaRef("taskBase")},
		{
			Properties: map[string]*jsonschema.Schema{
				"wait": {
					Ref:         SchemaRef("duration"),
					Title:       "WaitTaskConfiguration",
					Description: "The amount of time to wait.",
				},
			},
		},
	},
}

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

const (
	// SchemaVersion is the JSON Schema draft used by the generated schema.
	SchemaVersion = "https://json-schema.org/draft/2020-12/schema"

	// SchemaIDVersioned is the canonical versioned URI for the generated Zigflow workflow schema.
	SchemaIDVersioned = "https://zigflow.dev/schemas/%s/schema.%s"

	// SchemaIDVersioned is the canonical unversioned URI for the generated Zigflow workflow schema.
	SchemaID = "https://zigflow.dev/schema.%s"

	// SchemaTitle is the title of the generated schema.
	SchemaTitle = "Zigflow"

	// SchemaDescription is the description of the generated schema.
	SchemaDescription = "JSON Schema for a Zigflow workflow definition."

	// dnsLabelPattern ensures that a string meets the RFC1123 DNS label spec https://datatracker.ietf.org/doc/html/rfc1123#section-2.1
	dnsLabelPattern = `^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`

	// domainNamePattern ensures that a string meets the RFC1123 DNS label as an full host/domain name
	domainNamePattern = `^[a-zA-Z0-9](?:[a-zA-Z0-9-.]{0,61}[a-zA-Z0-9])?$`

	// runtimeExpressionString ensures a valid runtime expression
	runtimeExpressionString = `^\s*\$\{.+\}\s*$`

	// semVerPattern is the official regex from https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
	semVerPattern = `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

	// urlPattern ensures a valid URL
	urlPattern = `^[A-Za-z][A-Za-z0-9+\-.]*://.*`

	ref = "#/$defs/%s"

	// Definition keys used in buildDefinitions() and SchemaRef().
	defCommonMetadata   = "commonMetadata"
	defDocumentMetadata = "documentMetadata"

	// Schema property names used across schema definitions and tests.
	propDo              = "do"
	propDocument        = "document"
	propCall            = "call"
	propArguments       = "arguments"
	propActivityOptions = "activityOptions"
	propDisableEager    = "disableEagerExecution"
	propCanMaxHistory   = "canMaxHistoryLength"
	propZigflowID       = "__zigflow_id"
	propCommand         = "command"
	propCron            = "cron"

	// JSON Schema type values.
	typeArray   = "array"
	typeBoolean = "boolean"
	typeInteger = "integer"
	typeObject  = "object"
	typeString  = "string"

	// Definition keys used in buildDefinitions() and SchemaRef().
	defOutput       = "output"
	defSchema       = "schema"
	defTaskMetadata = "taskMetadata"
	defTimeout      = "timeout"

	// Additional schema property names.
	propEndpoint             = "endpoint"
	propError                = "error"
	propExport               = "export"
	propInput                = "input"
	propMethod               = "method"
	propMetadata             = "metadata"
	propHeartbeatTimeout     = "heartbeatTimeout"
	propMaximumAttempts      = "maximumAttempts"
	propHeartbeat            = "heartbeat"
	propEnvironment          = "environment"
	propDSL                  = "dsl"
	propMinutes              = "minutes"
	propWith                 = "with"
	propName                 = "name"
	propOutput               = "output"
	propSchema               = "schema"
	propRetryPolicy          = "retryPolicy"
	propTaskQueue            = "taskQueue"
	propType                 = "type"
	propScheduleWorkflowName = "scheduleWorkflowName"
	propScheduleID           = "scheduleId"
	propScheduleInput        = "scheduleInput"
	propSeconds              = "seconds"
	propStartToCloseTimeout  = "startToCloseTimeout"
	propSummary              = "summary"
	propURI                  = "uri"
	propSource               = "source"
	propSet                  = "set"
	propThen                 = "then"
	propWorkflowType         = "workflowType"
	propVersion              = "version"
)

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

package mcp

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/examples"
)

// docsBaseURL is the canonical website root for documentation pages. A page at
// path <p> is docsBaseURL + <p>, for example docsBaseURL + "concepts/common-mistakes".
const docsBaseURL = "https://zigflow.dev/docs/"

// Task type names referenced by the curated table. They are real Zigflow task
// types and are guarded against the schema by TestExplainError_RelatedTaskTypesAreReal.
const (
	taskTypeCall = "call"
	taskTypeSet  = "set"
	taskTypeWait = "wait"
)

// doc builds a canonical documentation URL from a site-relative path.
func doc(path string) string { return docsBaseURL + path }

type ExplainErrorInput struct {
	//nolint:lll // Struct tag contains schema description used by MCP tooling.
	ErrorMessage string `json:"error_message" jsonschema:"A Zigflow validation error message to explain, for example a message returned by validate_workflow."`
}

type ExplainErrorError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type ExplainErrorOutput struct {
	// Error echoes the validation error message that was explained.
	Error string `json:"error,omitempty"`
	// Matched reports whether a curated explanation was found. When false the
	// response is a graceful fallback rather than an invented explanation.
	Matched bool `json:"matched"`
	// Explanation is a concise, implementation-backed description of why the
	// validation failed.
	Explanation string `json:"explanation,omitempty"`
	// SuggestedFix is a concrete, deterministic action to resolve the error.
	SuggestedFix string `json:"suggestedFix,omitempty"`
	// Documentation lists canonical documentation URLs for the error.
	Documentation []string `json:"documentation,omitempty"`
	// RelatedTaskTypes names the task types most relevant to the error. Each
	// value is a real Zigflow task type.
	RelatedTaskTypes []string `json:"relatedTaskTypes,omitempty"`
	// Examples are bundled, validated workflows that demonstrate the fix.
	Examples []TaskExampleRef `json:"examples,omitempty"`
	// Errors carries input errors, mirroring the other MCP tools.
	Errors []ExplainErrorError `json:"errors,omitempty"`
}

// explanationRule is one entry in the curated explanation table. A rule matches
// when every contains substring is present (case-insensitive), every anyGroups
// group has at least one member present, and the optional regexp matches. Rules
// are evaluated in order and the first match wins, so more specific rules must
// precede more general ones.
type explanationRule struct {
	id        string
	contains  []string
	anyGroups [][]string
	re        *regexp.Regexp

	explanation      string
	suggestedFix     string
	documentation    []string
	relatedTaskTypes []string
	exampleNames     []string
}

func (r *explanationRule) matches(lower, raw string) bool {
	defined := false

	for _, sub := range r.contains {
		defined = true
		if !strings.Contains(lower, strings.ToLower(sub)) {
			return false
		}
	}

	for _, group := range r.anyGroups {
		defined = true
		found := false
		for _, sub := range group {
			if strings.Contains(lower, strings.ToLower(sub)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if r.re != nil {
		defined = true
		if !r.re.MatchString(raw) {
			return false
		}
	}

	return defined
}

// explanationRules is the curated, deterministic explanation table. It is kept
// small and matches against substrings that Zigflow validation genuinely
// produces (verified against validate_workflow output) plus tolerant forms of
// the same failures. Entries are ordered most-specific first. The table is
// guarded by tests so the documentation links, related task types and example
// references cannot drift from real data.
var explanationRules = []explanationRule{
	{
		// RFC 1123 DNS label failure on taskQueue or workflowType. Matches both
		// the raw schema message (field path + pattern + regex) and a normalised
		// "must match RFC1123 DNS label" form.
		id: "rfc1123-name",
		anyGroups: [][]string{
			{"taskqueue", "workflowtype"},
			{"rfc1123", "rfc 1123", "dns label", "pattern", "[a-za-z0-9]"},
		},
		explanation: "Zigflow requires document.taskQueue and document.workflowType to be RFC 1123 DNS labels: " +
			"letters, digits and hyphens, starting and ending with a letter or digit. Underscores, dots and spaces " +
			"are rejected. This is a Zigflow validation rule, not a Temporal one.",
		suggestedFix: "Rename the value using letters, digits and hyphens only, for example my-queue-v2 instead of " +
			"my_queue.v2.",
		documentation: []string{doc("concepts/common-mistakes"), doc("dsl/schema")},
	},
	{
		// Determinism pass rejects generated values outside a set task.
		id:       "non-deterministic-expression",
		contains: []string{"non-deterministic"},
		explanation: "Generated values such as uuid, timestamp and timestamp_iso8601 are non-deterministic. Temporal " +
			"replays workflow history, so a value produced outside a recorded side effect changes on replay. These " +
			"values are only allowed inside a set task, where the result is captured in history.",
		suggestedFix: "Move the expression into a set task and reference the stored result through $data in later " +
			"tasks.",
		documentation:    []string{doc("concepts/data-and-expressions"), doc("concepts/common-mistakes"), doc("dsl/tasks/set")},
		relatedTaskTypes: []string{taskTypeSet, taskTypeWait},
	},
	{
		// Expression references a variable that does not exist.
		id:       "invalid-runtime-variable",
		contains: []string{"variable not defined"},
		explanation: "Only five runtime variables exist in an expression: $context, $data, $env, $input and $output. " +
			"Names such as $workflow, $task, $steps, $vars or $now do not exist and fail when the expression is " +
			"compiled.",
		suggestedFix: "Replace the unknown variable with one of $context, $data, $env, $input or $output. For example " +
			"use $data.workflow for workflow metadata, or compute ${ timestamp } inside a set task instead of $now.",
		documentation: []string{doc("concepts/data-and-expressions"), doc("concepts/common-mistakes")},
	},
	{
		// jq could not be parsed or compiled (after the variable-not-defined rule
		// above, which is the more specific compile failure).
		id:        "invalid-expression-syntax",
		anyGroups: [][]string{{"failed to parse expression", "failed to compile expression", "unexpected token"}},
		explanation: "Runtime expressions are jq (via gojq) wrapped in ${ ... }. This expression could not be parsed " +
			"or compiled. Common causes are the wrong wrapper ({{ }}, ${{ }} or a bare jq path) or invalid jq syntax.",
		suggestedFix:  "Wrap expressions in ${ ... } and use valid jq, for example ${ $input.name }.",
		documentation: []string{doc("concepts/data-and-expressions"), doc("concepts/common-mistakes")},
	},
	{
		id:        "unsupported-dsl-version",
		anyGroups: [][]string{{"unsupported dsl"}},
		explanation: "The document.dsl version is not supported by this build of Zigflow. dsl must be a supported " +
			"semantic version.",
		suggestedFix: "Set document.dsl to a supported version, for example 1.0.0. Use get_schema to confirm the " +
			"version this server supports.",
		documentation: []string{doc("dsl/schema"), doc("dsl/intro")},
	},
	{
		id:        "unsupported-call-type",
		anyGroups: [][]string{{"unsupported call type", "unsupported call task"}},
		explanation: "A call task selects a supported call type. Zigflow supports activity, http and grpc calls. The " +
			"value after call: is not one of these.",
		suggestedFix: "Use call: http for HTTP requests, call: activity for Temporal activities or call: grpc for " +
			"gRPC. See the call task reference.",
		documentation:    []string{doc("dsl/tasks/call"), doc("concepts/common-mistakes")},
		relatedTaskTypes: []string{taskTypeCall},
		exampleNames:     []string{"activity-call"},
	},
	{
		id:        "unsupported-task-type",
		anyGroups: [][]string{{"unsupported task type"}},
		explanation: "A task's type is the single task key in its body. Zigflow supports eleven task types: call, do, " +
			"for, fork, listen, raise, run, set, switch, try and wait. Types such as http, parallel, function, delay " +
			"or emit do not exist.",
		suggestedFix: "Replace the task key with a supported type. An HTTP request is call: http, concurrency is fork " +
			"and a delay is wait.",
		documentation: []string{doc("dsl/tasks/intro"), doc("concepts/common-mistakes")},
	},
	{
		id:        "invalid-duration",
		anyGroups: [][]string{{"duration"}},
		explanation: "A duration is an object with plural integer keys: days, hours, minutes, seconds and " +
			"milliseconds. ISO 8601 strings such as PT1M are not supported, and singular keys such as minute are " +
			"rejected.",
		suggestedFix:     "Use plural keys, for example minutes: 1 and seconds: 30, instead of PT1M or minute: 1.",
		documentation:    []string{doc("dsl/tasks/wait"), doc("concepts/common-mistakes")},
		relatedTaskTypes: []string{taskTypeWait},
		exampleNames:     []string{taskTypeWait},
	},
	{
		// document is a closed object; unknown keys fail unevaluatedProperties.
		id:       "unknown-document-field",
		contains: []string{"/properties/document"},
		anyGroups: [][]string{
			{"unevaluatedproperties", "additionalproperties"},
		},
		explanation: "The document object is closed and rejects unknown fields. There is no name, description or " +
			"author. Use title for a human-readable name, summary for prose and metadata or tags for anything else.",
		suggestedFix: "Move unsupported keys: use document.title instead of document.name and document.summary " +
			"instead of document.description.",
		documentation: []string{doc("concepts/common-mistakes"), doc("dsl/schema")},
	},
	{
		// endpoint vs url. The raw schema message for this is the generic task
		// oneOf failure below; this rule fires on normalised messages that name
		// the field directly.
		id: "endpoint-not-url",
		re: regexp.MustCompile(`(?i)\b(url|endpoint)\b`),
		explanation: "An HTTP call addresses its target with endpoint, not url. There is no url property; the name " +
			"comes from the Serverless Workflow specification.",
		suggestedFix: "Rename url to endpoint inside the call's with block.",
		documentation: []string{
			doc("dsl/tasks/call"),
			doc("concepts/common-mistakes"),
		},
		relatedTaskTypes: []string{taskTypeCall},
		exampleNames:     []string{"activity-call"},
	},
	{
		// export vs output. As above, the raw schema message is generic; this
		// fires on normalised messages that name the field directly.
		id: "export-vs-output",
		re: regexp.MustCompile(`(?i)\b(export|output)\b`),
		explanation: "export and output are different channels. output.as shapes $output, the result that flows to " +
			"the next task and that the workflow returns. export.as writes to $context, which persists for later " +
			"tasks. Each export replaces $context wholesale.",
		suggestedFix: "Use output.as to shape the task result and export.as to persist values to $context. To add to " +
			"$context without losing earlier values, merge explicitly with ${ $context + { ... } }.",
		documentation: []string{doc("concepts/data-flow"), doc("concepts/common-mistakes")},
	},
	{
		// Generic task body mismatch. Zigflow schema validation collapses many
		// distinct task mistakes (non-existent task type, url instead of
		// endpoint, ISO 8601 or singular duration keys, malformed output or
		// export) into a single oneOf failure under $defs/task, so this catch-all
		// enumerates the common causes rather than guessing one.
		id:       "task-shape-oneof",
		contains: []string{"oneof"},
		anyGroups: [][]string{
			{"$defs/task", "did not validate against any"},
		},
		explanation: "The task body did not match any known task type or shape. Common causes are a task type that " +
			"does not exist (such as http, parallel, function, delay or emit), using url instead of endpoint in a " +
			"call, an ISO 8601 or singular duration key, or a malformed output or export block.",
		suggestedFix: "Check the task against the task reference. Map habits from elsewhere: http: becomes call: " +
			"http, parallel: becomes fork, delay: becomes wait, url: becomes endpoint, and durations use plural keys " +
			"such as minutes.",
		documentation: []string{doc("concepts/common-mistakes"), doc("dsl/tasks/intro")},
	},
}

// resolveExampleRefs turns curated example names into validated example
// references, preserving order. It returns an error if a name does not exist in
// the catalog so a stale reference is caught rather than silently dropped.
func resolveExampleRefs(fsys fs.FS, names []string) ([]TaskExampleRef, error) {
	if len(names) == 0 {
		return nil, nil
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("loading examples: %w", err)
	}

	byName := make(map[string]examples.Example, len(catalog))
	for _, ex := range catalog {
		byName[ex.Name] = ex
	}

	refs := make([]TaskExampleRef, 0, len(names))
	for _, n := range names {
		ex, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("explain_error references unknown example %q", n)
		}
		refs = append(refs, TaskExampleRef{Name: ex.Name, Title: ex.Title})
	}

	return refs, nil
}

func explainError(examplesFS fs.FS, errorMessage string) (ExplainErrorOutput, error) {
	trimmed := strings.TrimSpace(errorMessage)
	if trimmed == "" {
		return ExplainErrorOutput{
			Errors: []ExplainErrorError{{
				Stage:   stageInput,
				Message: "error_message is required",
			}},
		}, nil
	}

	lower := strings.ToLower(trimmed)
	for i := range explanationRules {
		rule := &explanationRules[i]
		if !rule.matches(lower, trimmed) {
			continue
		}

		refs, err := resolveExampleRefs(examplesFS, rule.exampleNames)
		if err != nil {
			return ExplainErrorOutput{}, err
		}

		return ExplainErrorOutput{
			Error:            trimmed,
			Matched:          true,
			Explanation:      rule.explanation,
			SuggestedFix:     rule.suggestedFix,
			Documentation:    rule.documentation,
			RelatedTaskTypes: rule.relatedTaskTypes,
			Examples:         refs,
		}, nil
	}

	// No curated rule matched. Degrade gracefully rather than inventing an
	// explanation, and point at the common mistakes page as a starting point.
	return ExplainErrorOutput{
		Error:   trimmed,
		Matched: false,
		Explanation: "No curated explanation is available for this error message. Validate the workflow with " +
			"validate_workflow for the precise failure, and review the common mistakes page.",
		Documentation: []string{doc("concepts/common-mistakes")},
	}, nil
}

func (m *MCP) ExplainError(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ExplainErrorInput,
) (*mcp.CallToolResult, ExplainErrorOutput, error) {
	out, err := explainError(examples.EmbeddedFS, input.ErrorMessage)
	return nil, out, err
}

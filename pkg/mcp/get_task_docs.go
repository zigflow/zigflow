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
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/schema"
	"github.com/zigflow/zigflow/docs"
	"github.com/zigflow/zigflow/examples"
)

// taskDocsBaseURL is the canonical website location for a task reference page.
// The page for task type <type> is taskDocsBaseURL + <type>.
const taskDocsBaseURL = "https://zigflow.dev/docs/dsl/tasks/"

// taskExampleTags maps a task type to the example tags that demonstrate it. It
// is a discovery hint only - the authoritative documentation comes from the
// schema and the embedded reference pages. Tasks that are ubiquitous (do, set)
// have no dedicated tag and are intentionally absent. The mapping is guarded by
// tests so a renamed or removed tag is caught rather than silently returning no
// examples.
var taskExampleTags = map[string][]string{
	"call":   {"activity", "grpc", "http"},
	"for":    {"for-loop"},
	"fork":   {"fork"},
	"listen": {"signal", "query", "update"},
	"raise":  {"error"},
	"run":    {"run"},
	"switch": {"switch"},
	"try":    {"try-catch"},
	"wait":   {"timeout"},
}

type GetTaskDocsInput struct {
	//nolint:lll // Struct tag contains schema description used by MCP tooling.
	TaskType string `json:"task_type" jsonschema:"The task type to document. One of: call, do, for, fork, listen, raise, run, set, switch, try, wait."`
}

type GetTaskDocsError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// TaskExampleRef points to a bundled, validated example that exercises the task.
type TaskExampleRef struct {
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
}

type GetTaskDocsOutput struct {
	// TaskType is the normalised task type the response describes.
	TaskType string `json:"taskType,omitempty"`
	// Description is the authoritative summary from the JSON schema definition.
	Description string `json:"description,omitempty"`
	// SubTypes lists the discriminated variants of the task, when the schema
	// exposes them (for example call -> activity, grpc, http). Empty otherwise.
	SubTypes []string `json:"subTypes,omitempty"`
	// Schema is the task's JSON schema definition (properties, required fields
	// and constraints). It is the authoritative source for task structure.
	Schema string `json:"schema,omitempty"`
	// Documentation is the full Markdown reference page for the task, served
	// verbatim from the same source the website renders.
	Documentation string `json:"documentation,omitempty"`
	// RelatedLinks are canonical documentation URLs for the task.
	RelatedLinks []string `json:"relatedLinks,omitempty"`
	// Examples are bundled, validated workflows that demonstrate the task.
	Examples []TaskExampleRef   `json:"examples,omitempty"`
	Errors   []GetTaskDocsError `json:"errors,omitempty"`
}

// supportedTaskTypes returns the task types backed by an embedded reference
// page, derived from the embedded docs filesystem so the list cannot drift from
// what can actually be served. intro.md is excluded as it is not a task.
func supportedTaskTypes(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, docs.TaskDocsDir)
	if err != nil {
		return nil, fmt.Errorf("reading task docs directory: %w", err)
	}

	var types []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}

		taskType := strings.TrimSuffix(name, ".md")
		if taskType == "intro" {
			continue
		}

		types = append(types, taskType)
	}

	sort.Strings(types)

	return types, nil
}

// firstConst recursively returns the first const string found within a schema
// subtree. It is used to read the discriminator value from a task variant.
func firstConst(s *jsonschema.Schema) (string, bool) {
	if s == nil {
		return "", false
	}

	if s.Const != nil {
		if str, ok := (*s.Const).(string); ok {
			return str, true
		}
	}

	for _, child := range s.AllOf {
		if v, ok := firstConst(child); ok {
			return v, true
		}
	}

	for _, child := range s.Properties {
		if v, ok := firstConst(child); ok {
			return v, true
		}
	}

	return "", false
}

// subTypesFromSchema extracts the discriminated variants of a task from the
// const values of its oneOf entries. It returns nil when the schema does not
// expose variants this way (for example listen, run and wait, whose variants
// are not modelled as a discriminated oneOf).
func subTypesFromSchema(def *jsonschema.Schema) []string {
	if def == nil || len(def.OneOf) == 0 {
		return nil
	}

	var subTypes []string
	for _, entry := range def.OneOf {
		if v, ok := firstConst(entry); ok {
			subTypes = append(subTypes, v)
		}
	}

	if len(subTypes) == 0 {
		return nil
	}

	sort.Strings(subTypes)

	return subTypes
}

// examplesForTask returns the bundled examples whose tags match the task.
func examplesForTask(fsys fs.FS, taskType string) ([]TaskExampleRef, error) {
	wantTags := taskExampleTags[taskType]
	if len(wantTags) == 0 {
		return nil, nil
	}

	want := make(map[string]struct{}, len(wantTags))
	for _, t := range wantTags {
		want[t] = struct{}{}
	}

	catalog, err := examples.LoadCatalog(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("loading examples: %w", err)
	}

	var refs []TaskExampleRef
	for _, ex := range catalog {
		for _, tag := range ex.Tags {
			if _, ok := want[tag]; ok {
				refs = append(refs, TaskExampleRef{Name: ex.Name, Title: ex.Title})
				break
			}
		}
	}

	return refs, nil
}

func getTaskDocs(
	version string,
	docsFS, examplesFS fs.FS,
	taskType string,
) (GetTaskDocsOutput, error) {
	taskType = strings.ToLower(strings.TrimSpace(taskType))

	supported, err := supportedTaskTypes(docsFS)
	if err != nil {
		return GetTaskDocsOutput{}, err
	}

	if taskType == "" {
		return GetTaskDocsOutput{Errors: []GetTaskDocsError{{
			Stage:   stageInput,
			Message: "task_type is required; supported: " + strings.Join(supported, ", "),
		}}}, nil
	}

	known := false
	for _, t := range supported {
		if t == taskType {
			known = true
			break
		}
	}

	if !known {
		return GetTaskDocsOutput{Errors: []GetTaskDocsError{{
			Stage:   stageInput,
			Message: fmt.Sprintf("unknown task type %q; supported: %s", taskType, strings.Join(supported, ", ")),
		}}}, nil
	}

	s, err := schema.BuildSchema(version, outputFormatJSON)
	if err != nil {
		return GetTaskDocsOutput{}, fmt.Errorf("building schema: %w", err)
	}

	def, ok := s.Defs[taskType+"Task"]
	if !ok {
		return GetTaskDocsOutput{}, fmt.Errorf("schema definition for task %q not found", taskType)
	}

	schemaJSON, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return GetTaskDocsOutput{}, fmt.Errorf("marshalling schema for task %q: %w", taskType, err)
	}

	documentation, err := fs.ReadFile(docsFS, docs.TaskDocsDir+"/"+taskType+".md")
	if err != nil {
		return GetTaskDocsOutput{}, fmt.Errorf("reading documentation for task %q: %w", taskType, err)
	}

	refs, err := examplesForTask(examplesFS, taskType)
	if err != nil {
		return GetTaskDocsOutput{}, err
	}

	return GetTaskDocsOutput{
		TaskType:      taskType,
		Description:   def.Description,
		SubTypes:      subTypesFromSchema(def),
		Schema:        string(schemaJSON),
		Documentation: string(documentation),
		RelatedLinks:  []string{taskDocsBaseURL + taskType},
		Examples:      refs,
	}, nil
}

func (m *MCP) GetTaskDocs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetTaskDocsInput,
) (*mcp.CallToolResult, GetTaskDocsOutput, error) {
	out, err := getTaskDocs(m.version, docs.TaskDocsFS, examples.EmbeddedFS, input.TaskType)
	return nil, out, err
}

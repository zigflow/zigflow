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

// Package llmsdoc renders the reference sections of docs/static/llms.txt from
// authoritative sources, so they cannot drift from the schema, the bundled
// examples, the validation error registry or the MCP tool registration.
//
// The file is a hybrid: curated prose is hand-written in the template
// (docs/llms.txt.in) and reference regions are generated. Each generated
// region is delimited by HTML comment markers so it stays readable for humans
// and useful for AI assistants:
//
//	<!-- BEGIN GENERATED: task-types -->
//	...generated content...
//	<!-- END GENERATED: task-types -->
//
// Render replaces the content between each pair of markers and leaves everything
// else untouched.
package llmsdoc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/zigflow/schema"
	"github.com/zigflow/zigflow/examples"
	"github.com/zigflow/zigflow/pkg/mcp"
	"github.com/zigflow/zigflow/pkg/utils"
)

// schemaVersion is the schema build used to derive the generated content. The
// textual facts we extract (task keys, descriptions and call sub-types) are the
// same across versions, so a fixed value keeps generation deterministic.
const schemaVersion = "development"

// region names. Each must have a matching BEGIN/END marker pair in the template.
const (
	regionTaskTypes        = "task-types"
	regionCallSubtypes     = "call-subtypes"
	regionExampleCatalogue = "example-catalogue"
	regionErrorCodes       = "error-codes"
	regionMCPTools         = "mcp-tools"
)

// Render returns the rendered llms.txt produced from the template, with every
// generated region replaced by content derived from authoritative sources.
func Render(templateSrc string) (string, error) {
	s, err := schema.BuildSchema(schemaVersion, "json")
	if err != nil {
		return "", fmt.Errorf("building schema: %w", err)
	}

	catalog, err := examples.LoadCatalog(examples.EmbeddedFS, ".")
	if err != nil {
		return "", fmt.Errorf("loading example catalogue: %w", err)
	}

	regions := map[string]string{
		regionTaskTypes:        taskTypes(s),
		regionCallSubtypes:     callSubtypes(s),
		regionExampleCatalogue: exampleCatalogue(catalog),
		regionErrorCodes:       errorCodes(),
		regionMCPTools:         mcpTools(),
	}

	return replaceRegions(templateSrc, regions)
}

// replaceRegions swaps the body of each marked region for its generated
// content. It fails if the template references an unknown region, repeats a
// region, leaves one unterminated, or omits one that should be generated, so
// the template and generator cannot fall out of step.
func replaceRegions(templateSrc string, regions map[string]string) (string, error) {
	lines := strings.Split(templateSrc, "\n")

	var out []string
	seen := make(map[string]bool, len(regions))

	for i := 0; i < len(lines); i++ {
		name, ok := beginMarker(lines[i])
		if !ok {
			out = append(out, lines[i])
			continue
		}

		content, known := regions[name]
		if !known {
			return "", fmt.Errorf("template has unknown generated region %q", name)
		}
		if seen[name] {
			return "", fmt.Errorf("template repeats generated region %q", name)
		}
		seen[name] = true

		// Skip the existing body up to the matching end marker.
		end := -1
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == endLine(name) {
				end = j
				break
			}
		}
		if end == -1 {
			return "", fmt.Errorf("generated region %q is not terminated", name)
		}

		out = append(out, lines[i])
		out = append(out, strings.Split(content, "\n")...)
		out = append(out, lines[end])
		i = end
	}

	for name := range regions {
		if !seen[name] {
			return "", fmt.Errorf("template is missing generated region %q", name)
		}
	}

	return strings.Join(out, "\n"), nil
}

func endLine(name string) string { return "<!-- END GENERATED: " + name + " -->" }

// beginMarker returns the region name if the line is a begin marker.
func beginMarker(line string) (string, bool) {
	const prefix = "<!-- BEGIN GENERATED: "
	const suffix = " -->"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
	return name, name != ""
}

// taskTypes renders the supported task types table from the schema. The keys
// are derived from $defs/task OneOf and the purpose from each task definition's
// description, so adding or renaming a task in the schema updates the table.
func taskTypes(s *jsonschema.Schema) string {
	taskDef := s.Defs["task"]

	var b strings.Builder
	fmt.Fprintf(&b, "Zigflow supports exactly %d task types:\n\n", len(taskDef.OneOf))
	b.WriteString("| Task key | Purpose |\n")
	b.WriteString("|----------|---------|\n")

	for _, ref := range taskDef.OneOf {
		name := strings.TrimPrefix(ref.Ref, "#/$defs/")
		key := strings.TrimSuffix(name, "Task")
		purpose := ""
		if def := s.Defs[name]; def != nil {
			purpose = firstSentence(def.Description)
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", key, cell(purpose))
	}

	return strings.TrimRight(b.String(), "\n")
}

// callSubtypes renders the sentence naming the supported call sub-types, derived
// from the discriminator constants of the callTask schema definition.
func callSubtypes(s *jsonschema.Schema) string {
	subtypes := discriminators(s.Defs["callTask"])

	return fmt.Sprintf(
		"The `call` value selects the sub-type. Zigflow supports exactly %d: %s.",
		len(subtypes), backtickList(subtypes),
	)
}

// exampleCatalogue renders the catalogue of validated examples exposed through
// the MCP example tools.
func exampleCatalogue(catalog []examples.Example) string {
	var b strings.Builder
	b.WriteString("| Example | Description |\n")
	b.WriteString("|---------|-------------|\n")
	for _, ex := range catalog {
		fmt.Fprintf(&b, "| `%s` | %s |\n", ex.Name, cell(ex.Description))
	}
	return strings.TrimRight(b.String(), "\n")
}

// errorCodes renders the stable validation error codes and their documentation
// URLs from the validation code registry.
func errorCodes() string {
	var b strings.Builder
	b.WriteString("| Code | Documentation |\n")
	b.WriteString("|------|---------------|\n")
	for _, code := range utils.ErrorCodes() {
		fmt.Fprintf(&b, "| `%s` | %s |\n", code, utils.DocumentationURL(code))
	}
	return strings.TrimRight(b.String(), "\n")
}

// mcpTools renders the MCP tool list from the authoritative tool registration.
func mcpTools() string {
	var lines []string
	for _, t := range mcp.ToolDefinitions() {
		lines = append(lines, fmt.Sprintf("- `%s`: %s", t.Name, t.Description))
	}
	return strings.Join(lines, "\n")
}

// discriminators returns the sorted const discriminator values of a task's
// OneOf variants (for example call -> activity, grpc, http).
func discriminators(def *jsonschema.Schema) []string {
	if def == nil {
		return nil
	}

	var values []string
	for _, entry := range def.OneOf {
		if v, ok := firstConst(entry); ok {
			values = append(values, v)
		}
	}

	sort.Strings(values)

	return values
}

// firstConst recursively returns the first string const found within a schema
// subtree, used to read the discriminator value from a task variant.
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

// firstSentence returns the first sentence of a description, keeping table cells
// concise while staying faithful to the authoritative text.
func firstSentence(desc string) string {
	desc = strings.TrimSpace(desc)
	if idx := strings.Index(desc, ". "); idx != -1 {
		return desc[:idx+1]
	}
	return desc
}

// cell makes text safe for a Markdown table cell: single line, with pipes
// escaped so they are not read as column separators.
func cell(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.ReplaceAll(s, "|", "\\|")
}

// backtickList renders a list as `a`, `b` and `c`.
func backtickList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = "`" + item + "`"
	}
	switch len(quoted) {
	case 0:
		return ""
	case 1:
		return quoted[0]
	default:
		return strings.Join(quoted[:len(quoted)-1], ", ") + " and " + quoted[len(quoted)-1]
	}
}

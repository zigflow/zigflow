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
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zigflow/zigflow/docs"
	"github.com/zigflow/zigflow/examples"
)

// maxSuggestions caps how many task types and examples are returned, keeping the
// response focused and the ranking meaningful.
const (
	maxSuggestedTaskTypes = 5
	maxSuggestedExamples  = 5
)

// patternRule maps recognisable phrasing in a description to the task types that
// implement that pattern, together with a human-readable rationale. The rules
// are the deterministic, explainable core of suggest_pattern: matching is plain
// substring containment against the normalised description. Triggers must be
// lowercase and use single spaces (the same normalisation applied to input).
//
// excludes are negative guards: if any exclude is present in the description the
// rule does not fire, even when a trigger matches. They keep ambiguous triggers
// (such as "signal" or "event") from firing when the wording clearly indicates a
// different task (such as "send a signal", which means raise).
type patternRule struct {
	triggers  []string
	excludes  []string
	taskTypes []string
	rationale string
}

// patternRules is evaluated in order, so rationale lines appear in a stable
// sequence. Task types referenced here are validated against the schema-derived
// task list by tests, so a renamed or removed task type is caught.
var patternRules = []patternRule{
	{
		triggers: []string{
			"list", "lists", "iterate", "iterating", "iteration", "each item",
			"every item", "batch", "records", "record", "collection", "loop",
			"items", "process a list", "process list", "for each",
		},
		taskTypes: []string{taskFor, taskDo, taskCall},
		rationale: "Use for to iterate over a collection, do to group the per-item steps and call when each item needs external work.",
	},
	{
		triggers: []string{
			"approval", "approve", "approver", "human", "sign off",
			"wait for response", "manual review", "await approval", "human in the loop",
		},
		taskTypes: []string{taskListen, taskWait, taskSwitch},
		rationale: "Use listen to await an approval signal, wait to bound how long to wait and switch to branch on the decision.",
	},
	{
		triggers: []string{
			"listen", "listen for", "receive signal", "receive an event",
			"wait for signal", "wait for an event", subSignal, subQuery, subUpdate,
			"event",
		},
		// Wording that clearly means producing an event, which is raise, not
		// listening for one.
		excludes: []string{
			"send signal", "send a signal", "send event", "send an event",
			"raise event", "raise an event", "raise a signal", "emit event",
			"emit an event", "emit signal", "emit a signal", "fire event",
			"fire an event",
		},
		taskTypes: []string{taskListen},
		rationale: "Use listen to receive an external event such as a signal, query or update.",
	},
	{
		triggers: []string{
			"parallel", "fan out", "fan in", "concurrent", "concurrently",
			"simultaneously", "multiple apis", "several apis", "scatter", "gather",
			"at the same time", "in parallel",
		},
		taskTypes: []string{taskFork, taskCall},
		rationale: "Use fork to run branches concurrently and call to invoke each service.",
	},
	{
		triggers: []string{
			"fallback", "retry", "retries", "handle error", "error handling",
			"recover", "recovery", "catch error", "on failure", "try again",
		},
		taskTypes: []string{taskTry, taskCall},
		rationale: "Use try to catch errors and retry, with call for the primary and fallback work.",
	},
	{
		triggers: []string{
			"delay", "sleep", "pause", "timeout", "wait until", "cooldown",
			"throttle", "after a delay",
		},
		taskTypes: []string{taskWait},
		rationale: "Use wait to pause on a durable timer, either for a duration or until a timestamp.",
	},
	{
		triggers: []string{
			"route", "routing", "branch", "branching", "condition", "conditional",
			"decision", "dispatch", "depending on", "based on",
		},
		taskTypes: []string{taskSwitch},
		rationale: "Use switch to branch based on a condition.",
	},
	{
		triggers: []string{
			"set variable", "transform", "store result", "store the result",
			"assign", "compute", "shape data", "map data", "derive", "build payload",
		},
		taskTypes: []string{taskSet},
		rationale: "Use set to assign, transform or store values in workflow state.",
	},
	{
		triggers: []string{
			"send signal", "raise event", "raise an error", "fail workflow",
			"fail the workflow", "throw", "abort", "emit event",
		},
		taskTypes: []string{taskRaise},
		rationale: "Use raise to fail the workflow with a structured error.",
	},
	{
		triggers: []string{
			"child workflow", "start workflow", "sub workflow", "subworkflow",
			"run workflow", "invoke workflow", "orchestrate workflows",
		},
		taskTypes: []string{taskRun},
		rationale: "Use run to start a child workflow.",
	},
	{
		triggers: []string{
			"api", "apis", "http", "https", "grpc", "endpoint", "external service",
			"external api", "rest", "request", "webhook", "activity",
		},
		taskTypes: []string{taskCall},
		rationale: "Use call to invoke an HTTP endpoint, gRPC service or Temporal activity.",
	},
}

// stopWords are ignored when scoring example metadata so common filler words do
// not create spurious matches.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "to": {}, "of": {}, "in": {}, "on": {}, "and": {},
	"or": {}, "for": {}, "with": {}, "my": {}, "is": {}, "are": {}, "that": {},
	"this": {}, "it": {}, "as": {}, "by": {}, "at": {}, "be": {}, "i": {}, "we": {},
}

type SuggestPatternInput struct {
	//nolint:lll // Struct tag contains schema description used by MCP tooling.
	Description string `json:"description" jsonschema:"A plain-language description of what the workflow should achieve, for example 'process a list of records' or 'wait for approval'."`
}

type SuggestPatternError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type SuggestPatternOutput struct {
	// Description echoes the input it was matched against.
	Description string `json:"description,omitempty"`
	// SuggestedTaskTypes are the most relevant task types, most relevant first.
	SuggestedTaskTypes []string `json:"suggestedTaskTypes,omitempty"`
	// Examples are validated, bundled workflows that demonstrate the pattern.
	Examples []ExampleSummary `json:"examples,omitempty"`
	// Rationale explains, in plain language, why the matches were selected.
	Rationale []string              `json:"rationale,omitempty"`
	Errors    []SuggestPatternError `json:"errors,omitempty"`
}

// normalise lowercases and strips punctuation from a description, returning the
// collapsed single-spaced string (for substring trigger matching) and the set
// of significant tokens (for example-metadata scoring).
func normalise(s string) (normalised string, tokens map[string]struct{}) {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}

	fields := strings.Fields(b.String())

	tokens = make(map[string]struct{}, len(fields))
	for _, f := range fields {
		if len(f) < 2 {
			continue
		}
		if _, skip := stopWords[f]; skip {
			continue
		}
		tokens[f] = struct{}{}
	}

	return strings.Join(fields, " "), tokens
}

// tokenSet normalises a piece of metadata into a set of significant tokens.
func tokenSet(s string) map[string]struct{} {
	_, tokens := normalise(s)
	return tokens
}

// scoreTaskTypes applies the curated pattern rules to the normalised
// description, returning a score per task type and the rationale lines for the
// rules that fired, in declaration order.
func scoreTaskTypes(normDesc string) (scores map[string]int, rationale []string) {
	scores = map[string]int{}

	for _, rule := range patternRules {
		matched := false
		for _, trigger := range rule.triggers {
			if strings.Contains(normDesc, trigger) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// Negative guard: skip the rule when the wording clearly indicates a
		// different task, even though a trigger matched.
		excluded := false
		for _, exclude := range rule.excludes {
			if strings.Contains(normDesc, exclude) {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		for _, tt := range rule.taskTypes {
			scores[tt] += 3
		}
		rationale = append(rationale, rule.rationale)
	}

	return scores, rationale
}

// topTaskTypes returns the highest-scoring task types that also exist in the
// valid set, most relevant first, capped at maxSuggestedTaskTypes. Ties break
// alphabetically for deterministic output.
func topTaskTypes(scores map[string]int, valid map[string]struct{}) []string {
	type scored struct {
		taskType string
		score    int
	}

	ranked := make([]scored, 0, len(scores))
	for tt, score := range scores {
		if score <= 0 {
			continue
		}
		if _, ok := valid[tt]; !ok {
			continue
		}
		ranked = append(ranked, scored{taskType: tt, score: score})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].taskType < ranked[j].taskType
	})

	out := make([]string, 0, maxSuggestedTaskTypes)
	for _, s := range ranked {
		out = append(out, s.taskType)
		if len(out) == maxSuggestedTaskTypes {
			break
		}
	}

	return out
}

// scoreExamples ranks the catalog against the query tokens and the suggested
// task types. Token matches against tags, name, title and description all
// contribute; examples tagged for a suggested task type are boosted so the
// curated rules pull through their canonical examples.
func scoreExamples(catalog []examples.Example, tokens map[string]struct{}, taskTypes []string) []ExampleSummary {
	type scored struct {
		ex    examples.Example
		score int
	}

	ranked := make([]scored, 0, len(catalog))
	for _, ex := range catalog {
		score := 0

		tags := make(map[string]struct{}, len(ex.Tags))
		for _, tag := range ex.Tags {
			tags[tag] = struct{}{}
		}

		nameTokens := tokenSet(strings.ReplaceAll(ex.Name, "-", " "))
		titleTokens := tokenSet(ex.Title)
		descTokens := tokenSet(ex.Description)

		for token := range tokens {
			if _, ok := tags[token]; ok {
				score += 3
			}
			if _, ok := nameTokens[token]; ok {
				score += 2
			}
			if _, ok := titleTokens[token]; ok {
				score += 2
			}
			if _, ok := descTokens[token]; ok {
				score++
			}
		}

		// Boost examples whose tags match a suggested task type, so the
		// canonical example for a matched pattern surfaces even when the
		// description shares no literal words with the example metadata.
		for _, tt := range taskTypes {
			for _, want := range taskExampleTags[tt] {
				if _, ok := tags[want]; ok {
					score += 2
					break
				}
			}
		}

		if score <= 0 {
			continue
		}

		ranked = append(ranked, scored{ex: ex, score: score})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].ex.Name < ranked[j].ex.Name
	})

	out := make([]ExampleSummary, 0, maxSuggestedExamples)
	for _, s := range ranked {
		out = append(out, ExampleSummary{
			Name:        s.ex.Name,
			Title:       s.ex.Title,
			Description: s.ex.Description,
			Tags:        s.ex.Tags,
		})
		if len(out) == maxSuggestedExamples {
			break
		}
	}

	return out
}

func suggestPattern(docsFS, examplesFS fs.FS, description string) (SuggestPatternOutput, error) {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return SuggestPatternOutput{Errors: []SuggestPatternError{{
			Stage:   stageInput,
			Message: "description is required",
		}}}, nil
	}

	normDesc, tokens := normalise(trimmed)

	validTypes, err := supportedTaskTypes(docsFS)
	if err != nil {
		return SuggestPatternOutput{}, err
	}
	valid := make(map[string]struct{}, len(validTypes))
	for _, t := range validTypes {
		valid[t] = struct{}{}
	}

	scores, ruleRationale := scoreTaskTypes(normDesc)
	taskTypes := topTaskTypes(scores, valid)

	catalog, err := examples.LoadCatalog(examplesFS, ".")
	if err != nil {
		return SuggestPatternOutput{}, fmt.Errorf("loading examples: %w", err)
	}
	matched := scoreExamples(catalog, tokens, taskTypes)

	rationale := buildRationale(ruleRationale, taskTypes, matched)

	return SuggestPatternOutput{
		Description:        trimmed,
		SuggestedTaskTypes: taskTypes,
		Examples:           matched,
		Rationale:          rationale,
	}, nil
}

// buildRationale assembles the explanation, favouring the curated rule rationale
// and degrading gracefully when nothing strong matched.
func buildRationale(ruleRationale, taskTypes []string, matched []ExampleSummary) []string {
	rationale := append([]string{}, ruleRationale...)

	if len(rationale) == 0 && len(taskTypes) > 0 {
		rationale = append(rationale,
			"Suggested task types are based on terms in your description; call get_task_docs for details on each.")
	}

	if len(matched) > 0 {
		rationale = append(rationale,
			fmt.Sprintf("The %s example demonstrates a similar pattern.", matched[0].Name))
	}

	if len(taskTypes) == 0 && len(matched) == 0 {
		rationale = append(rationale,
			"No strong pattern matched the description. Try rephrasing, or call list_examples to browse available patterns.")
	}

	return rationale
}

func (m *MCP) SuggestPattern(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SuggestPatternInput,
) (*mcp.CallToolResult, SuggestPatternOutput, error) {
	out, err := suggestPattern(docs.TaskDocsFS, examples.EmbeddedFS, input.Description)
	return nil, out, err
}

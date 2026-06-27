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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zigflowdocs "github.com/zigflow/zigflow/docs"
	zigflowexamples "github.com/zigflow/zigflow/examples"
)

func suggest(t *testing.T, description string) SuggestPatternOutput {
	t.Helper()

	out, err := suggestPattern(zigflowdocs.TaskDocsFS, zigflowexamples.EmbeddedFS, description)
	require.NoError(t, err)

	return out
}

func exampleNames(summaries []ExampleSummary) []string {
	names := make([]string, len(summaries))
	for i, ex := range summaries {
		names[i] = ex.Name
	}
	return names
}

func TestSuggestPattern_ProcessListOfRecords(t *testing.T) {
	out := suggest(t, "Process a list of records")

	assert.Empty(t, out.Errors)
	assert.Equal(t, "Process a list of records", out.Description)
	assert.Contains(t, out.SuggestedTaskTypes, "for")
	assert.Contains(t, exampleNames(out.Examples), "batch-processing", "should surface the batch example")
	assert.NotEmpty(t, out.Rationale)
}

func TestSuggestPattern_WaitForApproval(t *testing.T) {
	out := suggest(t, "Wait for approval")

	assert.Empty(t, out.Errors)
	// Either listen or wait is acceptable; both are expected here.
	assert.Subset(t, out.SuggestedTaskTypes, []string{"listen", "wait"})
	assert.Contains(t, exampleNames(out.Examples), "approval-timeout", "should surface the approval example")
}

func TestSuggestPattern_CallThreeAPIsInParallel(t *testing.T) {
	out := suggest(t, "Call three APIs in parallel")

	assert.Empty(t, out.Errors)
	assert.Contains(t, out.SuggestedTaskTypes, "fork")
	assert.Contains(t, out.SuggestedTaskTypes, "call")
	assert.Contains(t, exampleNames(out.Examples), "fan-out-fan-in", "should surface the fan-out/fan-in example")
}

func TestSuggestPattern_EmptyInput(t *testing.T) {
	out := suggest(t, "")
	require.Len(t, out.Errors, 1)
	assert.Equal(t, stageInput, out.Errors[0].Stage)
	assert.NotEmpty(t, out.Errors[0].Message)
	assert.Empty(t, out.SuggestedTaskTypes)
	assert.Empty(t, out.Examples)
}

func TestSuggestPattern_WhitespaceInput(t *testing.T) {
	out := suggest(t, "   \t  ")
	require.Len(t, out.Errors, 1)
	assert.Equal(t, stageInput, out.Errors[0].Stage)
}

func TestSuggestPattern_WeakDescriptionDoesNotHallucinate(t *testing.T) {
	out := suggest(t, "xyzzy plugh frobnicate")

	assert.Empty(t, out.Errors)
	assert.Empty(t, out.SuggestedTaskTypes, "no task types should be invented")
	assert.Empty(t, out.Examples, "no examples should be invented")
	require.NotEmpty(t, out.Rationale)
	assert.Contains(t, out.Rationale[0], "No strong pattern")
}

func TestSuggestPattern_DirectListenDescriptions(t *testing.T) {
	for _, desc := range []string{
		"listen for a signal",
		"listen for a query",
		"listen for an update",
	} {
		t.Run(desc, func(t *testing.T) {
			out := suggest(t, desc)
			assert.Empty(t, out.Errors)
			assert.Contains(t, out.SuggestedTaskTypes, taskListen)
		})
	}
}

// TestSuggestPattern_ListenExcludesRaiseWording verifies the negative guard:
// wording that means producing an event must not suggest listen.
func TestSuggestPattern_ListenExcludesRaiseWording(t *testing.T) {
	for _, desc := range []string{
		"send a signal",
		"raise an event",
	} {
		t.Run(desc, func(t *testing.T) {
			out := suggest(t, desc)
			assert.Empty(t, out.Errors)
			assert.NotContains(t, out.SuggestedTaskTypes, taskListen)
		})
	}
}

// TestSuggestPattern_OnlyValidExamples guards against invented examples: every
// returned example name must exist in the validated catalog. A broad sweep of
// descriptions exercises many code paths.
func TestSuggestPattern_OnlyValidExamples(t *testing.T) {
	catalog, err := zigflowexamples.LoadCatalog(zigflowexamples.EmbeddedFS, ".")
	require.NoError(t, err)

	known := make(map[string]struct{}, len(catalog))
	for _, ex := range catalog {
		known[ex.Name] = struct{}{}
	}

	descriptions := []string{
		"Process a list of records",
		"Wait for approval",
		"Call three APIs in parallel",
		"Retry a failing call with a fallback",
		"Pause for five minutes",
		"Route based on order type",
		"Transform and store the result",
		"Start a child workflow",
		"Fail the workflow with an error",
	}

	for _, desc := range descriptions {
		out := suggest(t, desc)
		for _, ex := range out.Examples {
			_, ok := known[ex.Name]
			assert.Truef(t, ok, "description %q returned unknown example %q", desc, ex.Name)
		}
	}
}

// TestSuggestPattern_OnlyValidTaskTypes guards against invented task types:
// every returned task type must exist in the schema-derived task list.
func TestSuggestPattern_OnlyValidTaskTypes(t *testing.T) {
	valid := make(map[string]struct{})
	for _, k := range schemaTaskTypes(t) {
		valid[k] = struct{}{}
	}

	descriptions := []string{
		"Process a list of records",
		"Wait for approval",
		"Call three APIs in parallel",
		"Retry a failing call with a fallback",
		"Pause for five minutes",
		"Route based on order type",
		"Transform and store the result",
		"Start a child workflow",
		"Fail the workflow with an error",
	}

	for _, desc := range descriptions {
		out := suggest(t, desc)
		for _, tt := range out.SuggestedTaskTypes {
			_, ok := valid[tt]
			assert.Truef(t, ok, "description %q returned unknown task type %q", desc, tt)
		}
	}
}

// TestSuggestPattern_RuleTaskTypesAreValid guards the curated rules against
// schema drift: every task type referenced by a pattern rule must exist in the
// schema-derived task list.
func TestSuggestPattern_RuleTaskTypesAreValid(t *testing.T) {
	valid := make(map[string]struct{})
	for _, k := range schemaTaskTypes(t) {
		valid[k] = struct{}{}
	}

	for _, rule := range patternRules {
		for _, tt := range rule.taskTypes {
			_, ok := valid[tt]
			assert.Truef(t, ok, "pattern rule references unknown task type %q", tt)
		}
	}
}

// TestSuggestPattern_ResultsAreCapped verifies the response stays focused.
func TestSuggestPattern_ResultsAreCapped(t *testing.T) {
	// A description that fires several rules and matches many examples.
	out := suggest(t, "iterate over a list and call multiple apis in parallel with retry and fallback")

	assert.LessOrEqual(t, len(out.SuggestedTaskTypes), maxSuggestedTaskTypes)
	assert.LessOrEqual(t, len(out.Examples), maxSuggestedExamples)
}

func TestSuggestPattern_Deterministic(t *testing.T) {
	a := suggest(t, "Call three APIs in parallel")
	b := suggest(t, "Call three APIs in parallel")
	assert.Equal(t, a, b, "identical input must produce identical output")
}

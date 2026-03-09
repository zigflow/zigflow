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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixLabeledListSection_SingleArg(t *testing.T) {
	input := "Some preamble.\n\nArguments:\n  workflow-file   Path to the workflow file\n\nMore text."
	got := fixLabeledListSection(input)
	assert.Contains(t, got, "Arguments:")
	assert.Contains(t, got, "| Value | Description |")
	assert.Contains(t, got, "| `workflow-file` | Path to the workflow file |")
	assert.NotContains(t, got, "  workflow-file   Path to the workflow file")
}

func TestFixLabeledListSection_MultipleArgs(t *testing.T) {
	input := "Arguments:\n  source-file   Input file path\n  dest-file     Output file path\n"
	got := fixLabeledListSection(input)
	assert.Contains(t, got, "| `source-file` | Input file path |")
	assert.Contains(t, got, "| `dest-file` | Output file path |")
}

func TestFixLabeledListSection_Idempotent(t *testing.T) {
	input := "Arguments:\n  workflow-file   Path to the workflow file\n"
	first := fixLabeledListSection(input)
	second := fixLabeledListSection(first)
	assert.Equal(t, first, second)
}

func TestFixLabeledListSection_NoMatchingSection(t *testing.T) {
	input := "No arguments here.\n\nJust regular text.\n"
	got := fixLabeledListSection(input)
	assert.Equal(t, input, got)
}

func TestFixLabeledListSection_PreservesTextOutside(t *testing.T) {
	input := "Before.\n\nArguments:\n  my-arg   Does something\n\nAfter."
	got := fixLabeledListSection(input)
	assert.Contains(t, got, "Before.")
	assert.Contains(t, got, "After.")
}

func TestFixLabeledListSection_NoFollowingEntries(t *testing.T) {
	// Header ending in ":" but no matching indented lines — pass through unchanged.
	input := "Arguments:\nNo indented lines here.\n"
	got := fixLabeledListSection(input)
	assert.Equal(t, input, got)
}

func TestFixLabeledListSection_BlankLineBetweenHeaderAndEntries(t *testing.T) {
	// "Currently supported:" pattern has a blank line before the entries.
	input := "Use the renderer. Currently supported:\n\n  mermaid   Mermaid flowchart\n\nMore text."
	got := fixLabeledListSection(input)
	assert.Contains(t, got, "Currently supported:")
	assert.Contains(t, got, "| Value | Description |")
	assert.Contains(t, got, "| `mermaid` | Mermaid flowchart |")
	assert.NotContains(t, got, "  mermaid   Mermaid flowchart")
}

func TestFixLabeledListSection_ProseColonNotConverted(t *testing.T) {
	// An ordinary sentence ending in ":" whose next non-blank line is not a
	// key-value entry must be left unchanged.
	input := "The runtime uses:\ntasks are shown in execution order.\n"
	got := fixLabeledListSection(input)
	assert.Equal(t, input, got)
}

func TestFixLabeledListSection_BulletListNotConverted(t *testing.T) {
	// "  - item" lines do not match the key-value pattern (single space after "-").
	input := "Validation includes:\n  - DSL syntax checks\n  - Schema validation\n"
	got := fixLabeledListSection(input)
	assert.Equal(t, input, got)
}

func TestFixFlagNames_WrapsBareFlag(t *testing.T) {
	got := fixFlagNames("Use the --output flag.")
	assert.Equal(t, "Use the `--output` flag.", got)
}

func TestFixFlagNames_HyphenatedFlag(t *testing.T) {
	got := fixFlagNames("Set --start-marker to override.")
	assert.Equal(t, "Set `--start-marker` to override.", got)
}

func TestFixFlagNames_AlphanumericFlag(t *testing.T) {
	got := fixFlagNames("Pass --some-thing2 here.")
	assert.Equal(t, "Pass `--some-thing2` here.", got)
}

func TestFixFlagNames_MultipleFlagsOnOneLine(t *testing.T) {
	got := fixFlagNames("Use --output and --workflow together.")
	assert.Equal(t, "Use `--output` and `--workflow` together.", got)
}

func TestFixFlagNames_DoesNotDoubleWrap(t *testing.T) {
	got := fixFlagNames("Use the `--output` flag.")
	assert.Equal(t, "Use the `--output` flag.", got)
}

func TestFixFlagNames_SkipsFencedCodeBlock(t *testing.T) {
	input := "--workflow in prose.\n\n```\n  --workflow string\n```\n\n--output also in prose."
	got := fixFlagNames(input)
	assert.Contains(t, got, "`--workflow` in prose.")
	assert.Contains(t, got, "  --workflow string") // unchanged inside fence
	assert.Contains(t, got, "`--output` also in prose.")
}

func TestFixFlagNames_HTMLCommentNotWrapped(t *testing.T) {
	// "-->" is not a valid flag (not followed by a lowercase letter).
	got := fixFlagNames("<!-- ZIGFLOW_GRAPH_START -->")
	assert.Equal(t, "<!-- ZIGFLOW_GRAPH_START -->", got)
}

func TestFixFlagNames_TripleDashNotWrapped(t *testing.T) {
	// "---" YAML front-matter separator must not be touched.
	got := fixFlagNames("---")
	assert.Equal(t, "---", got)
}

func TestFixFlagNames_Idempotent(t *testing.T) {
	input := "Run with --output flag and --workflow set."
	first := fixFlagNames(input)
	second := fixFlagNames(first)
	assert.Equal(t, first, second)
}

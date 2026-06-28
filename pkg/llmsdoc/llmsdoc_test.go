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

package llmsdoc

import (
	"strings"
	"testing"
)

// minimalTemplate exercises every generated region so Render can be tested
// without depending on the full committed template.
const minimalTemplate = `# Heading

curated prose

<!-- BEGIN GENERATED: task-types -->
placeholder
<!-- END GENERATED: task-types -->

<!-- BEGIN GENERATED: call-subtypes -->
placeholder
<!-- END GENERATED: call-subtypes -->

<!-- BEGIN GENERATED: example-catalogue -->
placeholder
<!-- END GENERATED: example-catalogue -->

<!-- BEGIN GENERATED: error-codes -->
placeholder
<!-- END GENERATED: error-codes -->

<!-- BEGIN GENERATED: mcp-tools -->
placeholder
<!-- END GENERATED: mcp-tools -->

end of file
`

func TestRenderReplacesEveryRegion(t *testing.T) {
	out, err := Render(minimalTemplate)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out, "placeholder") {
		t.Error("rendered output still contains placeholder text")
	}

	// Curated prose and structure must be preserved verbatim.
	for _, want := range []string{"# Heading", "curated prose", "end of file"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output dropped curated content %q", want)
		}
	}

	// Markers must remain so the file stays regenerable.
	for _, region := range []string{
		regionTaskTypes, regionCallSubtypes, regionExampleCatalogue,
		regionErrorCodes, regionMCPTools,
	} {
		if !strings.Contains(out, beginMarkerLine(region)) || !strings.Contains(out, endLine(region)) {
			t.Errorf("rendered output is missing markers for region %q", region)
		}
	}

	// Spot-check authoritative content per region.
	for _, want := range []string{
		"| Task key | Purpose |",
		"`call`",
		"sub-type",
		"`activity`",
		"| Example | Description |",
		"`hello-world`",
		"ERR_INVALID_TASK_QUEUE",
		"https://zigflow.dev/errors/invalid-task-queue",
		"- `get_schema`:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output is missing expected content %q", want)
		}
	}
}

func TestRenderRejectsUnknownRegion(t *testing.T) {
	tmpl := "<!-- BEGIN GENERATED: nonsense -->\n<!-- END GENERATED: nonsense -->\n"
	if _, err := Render(tmpl); err == nil {
		t.Fatal("expected an error for an unknown region")
	}
}

func TestRenderRejectsMissingRegion(t *testing.T) {
	// A template with no markers omits every region.
	if _, err := Render("just prose\n"); err == nil {
		t.Fatal("expected an error when generated regions are missing")
	}
}

func TestRenderRejectsUnterminatedRegion(t *testing.T) {
	tmpl := "<!-- BEGIN GENERATED: task-types -->\nno end marker\n"
	if _, err := Render(tmpl); err == nil {
		t.Fatal("expected an error for an unterminated region")
	}
}

func TestFirstSentence(t *testing.T) {
	cases := map[string]string{
		"A task used to set data.":             "A task used to set data.",
		"First sentence. Second sentence.":     "First sentence.",
		"No trailing period":                   "No trailing period",
		"  leading and trailing whitespace  .": "leading and trailing whitespace  .",
	}
	for in, want := range cases {
		if got := firstSentence(in); got != want {
			t.Errorf("firstSentence(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCellEscapesPipes(t *testing.T) {
	if got := cell("a | b\nc"); got != `a \| b c` {
		t.Errorf("cell did not normalise and escape: %q", got)
	}
}

func TestBacktickList(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"one":         "`one`",
		"one,two":     "`one` and `two`",
		"one,two,thr": "`one`, `two` and `thr`",
	}
	for in, want := range cases {
		var items []string
		if in != "" {
			items = strings.Split(in, ",")
		}
		if got := backtickList(items); got != want {
			t.Errorf("backtickList(%v) = %q, want %q", items, got, want)
		}
	}
}

// beginMarkerLine mirrors the begin marker used by the template, for assertions.
func beginMarkerLine(name string) string { return "<!-- BEGIN GENERATED: " + name + " -->" }

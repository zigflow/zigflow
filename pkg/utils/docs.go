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
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// argLineRe matches a 2-space-indented argument line of the form:
//
//	arg-name   description text
var argLineRe = regexp.MustCompile(`^ {2}(\S+)\s{2,}(.+)$`)

// flagOrCodeSpanRe matches either an inline code span or a bare CLI flag.
// The alternation ensures code spans are consumed first so already-backticked
// flags are left untouched by fixFlagNames.
var flagOrCodeSpanRe = regexp.MustCompile("`[^`]+`|--[a-z][a-z0-9-]*")

func FilePrepender(filename string) string {
	base := filepath.Base(filename)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	title = strings.ReplaceAll(title, "_", " ")

	return `---
title: "` + title + `"
---

`
}

func fixIndentedCodeBlocks(content string) string {
	lines := strings.Split(content, "\n")

	var out []string
	inFence := false
	inIndentedBlock := false

	flushIndented := func() {
		if inIndentedBlock {
			out = append(out, "```")
			inIndentedBlock = false
		}
	}

	for _, line := range lines {
		trim := strings.TrimSpace(line)

		// Track existing fenced code blocks
		if strings.HasPrefix(trim, "```") {
			flushIndented()
			inFence = !inFence
			out = append(out, line)
			continue
		}

		// Detect indented code blocks (tabs or 4 spaces), outside fenced blocks
		if !inFence && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")) {
			if !inIndentedBlock {
				out = append(out, "```bash")
				inIndentedBlock = true
			}
			// Strip leading indentation
			out = append(out, strings.TrimLeft(line, "\t "))
			continue
		}

		// End indented block when indentation stops
		if inIndentedBlock {
			flushIndented()
		}

		out = append(out, line)
	}

	flushIndented()
	return strings.Join(out, "\n")
}

// fixLabeledListSection converts cobra-style labelled key-value sections to
// markdown tables so they render correctly in MDX documentation.
//
// Any line ending with ":" that is followed (with optional blank lines) by one
// or more 2-space-indented "  key   description" entries is converted. For
// example:
//
//	Arguments:
//	  arg-name   Description text
//
// becomes:
//
//	Arguments:
//
//	| Value | Description |
//	|---|---|
//	| `arg-name` | Description text |
//
// Lines that end with ":" but are not followed by matching entries are left
// unchanged, so ordinary prose sentences ending in a colon are unaffected.
func fixLabeledListSection(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		if strings.HasSuffix(lines[i], ":") {
			// Skip optional blank lines between the header and its entries.
			j := i + 1
			for j < len(lines) && lines[j] == "" {
				j++
			}
			// Collect matching key-value entries.
			type argRow struct{ name, desc string }
			var args []argRow
			k := j
			for k < len(lines) {
				m := argLineRe.FindStringSubmatch(lines[k])
				if m == nil {
					break
				}
				args = append(args, argRow{m[1], m[2]})
				k++
			}
			if len(args) > 0 {
				// Emit the header line unchanged, a blank line, then the table.
				out = append(out, lines[i], "", "| Value | Description |", "|---|---|")
				for _, a := range args {
					out = append(out, "| `"+a.name+"` | "+a.desc+" |")
				}
				i = k
				continue
			}
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n")
}

// fixFlagNames wraps bare CLI flag references (e.g. --output, --start-marker)
// in backticks so they render as inline code in MDX documentation. Flags that
// are already inside a backtick span or inside a fenced code block are left
// untouched.
func fixFlagNames(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			out = append(out, line)
			continue
		}
		if inFence {
			out = append(out, line)
			continue
		}
		line = flagOrCodeSpanRe.ReplaceAllStringFunc(line, func(m string) string {
			if strings.HasPrefix(m, "`") {
				return m
			}
			return "`" + m + "`"
		})
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func LinkHandler(name string) string {
	base := strings.TrimSuffix(name, ".md")
	return base
}

func SanitizeForMDX(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := fixIndentedCodeBlocks(string(b))
	content = fixLabeledListSection(content)
	content = fixFlagNames(content)

	return os.WriteFile(path, []byte(content), 0o600)
}

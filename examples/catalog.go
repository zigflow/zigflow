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

package examples

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"sigs.k8s.io/yaml"
)

// Example describes a single bundled Zigflow workflow example.
type Example struct {
	// Name is the stable directory-based identifier, e.g. "hello-world".
	Name string
	// Title is the human-friendly label from document.title.
	Title string
	// Description is the short summary from document.summary.
	Description string
	// Tags are inferred retrieval hints for this example.
	Tags []string
	// Dir is the path to the example directory within the FS.
	Dir string
}

// knownTags maps example directory names to retrieval tags. Only entries where
// the tag is unambiguously grounded in what the example demonstrates are listed.
var knownTags = map[string][]string{
	"activity-call":              {"activity"},
	"authorise-change-request":   {"signal", "update"},
	"child-workflows":            {"child-workflow"},
	"cloudevents":                {"cloudevents", "debugging"},
	"competing-concurrent-tasks": {"fork", "competition"},
	"conditionally-execute":      {"switch"},
	"external-calls":             {"http", "grpc"},
	"for-loop":                   {"for-loop"},
	"heartbeat":                  {"heartbeat", "activity"},
	"multiple-workflow-files":    {"multi-file"},
	"multiple-workflows":         {"multi-workflow"},
	"priority-and-fairness":      {"priority", "fairness"},
	"query":                      {"query"},
	"raise":                      {"error"},
	"run-task":                   {"run", "script"},
	"schedule":                   {"schedule"},
	"search-attributes":          {"search-attributes"},
	"signal":                     {"signal"},
	"switch":                     {"switch"},
	"try-catch":                  {"error", "try-catch"},
	"update":                     {"update"},
}

// exampleMeta is the minimal YAML structure needed to extract example metadata.
type exampleMeta struct {
	Document struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
	} `json:"document"`
}

// LoadCatalog reads all subdirectories of root within fsys and returns a sorted
// slice of Examples. Each subdirectory must contain either workflow.yaml or
// info.yaml with a document.title and document.summary. An error is returned if
// any bundled example cannot be read or parsed.
func LoadCatalog(fsys fs.FS, root string) ([]Example, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading examples directory: %w", err)
	}

	var result []Example

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		exDir := name
		if root != "." {
			exDir = root + "/" + name
		}

		meta, err := readMeta(fsys, exDir)
		if err != nil {
			return nil, fmt.Errorf("example %q: %w", name, err)
		}

		result = append(result, Example{
			Name:        name,
			Title:       meta.Document.Title,
			Description: meta.Document.Summary,
			Tags:        knownTags[name],
			Dir:         exDir,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// readMeta tries workflow.yaml first, then info.yaml as a fallback.
func readMeta(fsys fs.FS, dir string) (exampleMeta, error) {
	for _, filename := range []string{"workflow.yaml", "info.yaml"} {
		data, err := fs.ReadFile(fsys, dir+"/"+filename)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return exampleMeta{}, fmt.Errorf("reading %s: %w", filename, err)
		}

		var meta exampleMeta
		if err := yaml.Unmarshal(data, &meta); err != nil {
			return exampleMeta{}, fmt.Errorf("parsing %s: %w", filename, err)
		}

		return meta, nil
	}

	return exampleMeta{}, fmt.Errorf("no workflow.yaml or info.yaml found")
}

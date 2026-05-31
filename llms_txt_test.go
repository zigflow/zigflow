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

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/zigflow/zigflow/pkg/schema"
)

// llmsTxtPath is resolved relative to the repository root, which is the working
// directory when `go test` runs the root package.
const llmsTxtPath = "docs/static/llms.txt"

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(b)
}

// TestLLMSTxtDocumentsEveryTaskType guards against the llms.txt task catalogue
// drifting from the schema. The task keys are derived from the authoritative
// task definition (schema $defs/task OneOf) rather than hard-coded here, so a
// task type added to or renamed in the schema fails this test until llms.txt is
// updated. This is the only drift check in scope for issue #447; broader
// generation of llms.txt is deferred to a sibling issue.
func TestLLMSTxtDocumentsEveryTaskType(t *testing.T) {
	s, err := schema.BuildSchema("development", "json")
	if err != nil {
		t.Fatalf("failed to build schema: %v", err)
	}

	taskDef, ok := s.Defs["task"]
	if !ok {
		t.Fatal("schema is missing the $defs/task definition")
	}
	if len(taskDef.OneOf) == 0 {
		t.Fatal("$defs/task has no OneOf task references")
	}

	llms := readFile(t, llmsTxtPath)

	for _, ref := range taskDef.OneOf {
		// Refs look like "#/$defs/callTask"; the DSL task key is the ref name
		// with the "$defs/" prefix and "Task" suffix removed (e.g. "call").
		name := strings.TrimPrefix(ref.Ref, "#/$defs/")
		key := strings.TrimSuffix(name, "Task")
		if key == "" || key == name {
			t.Fatalf("unexpected task ref %q in $defs/task OneOf", ref.Ref)
		}

		if !strings.Contains(llms, "`"+key+"`") {
			t.Errorf("llms.txt does not document task type %q (derived from %s)", key, ref.Ref)
		}
	}
}

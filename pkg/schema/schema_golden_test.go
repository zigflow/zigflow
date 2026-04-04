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

package schema_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/zigflow/zigflow/pkg/schema"
)

// buildCompiledSchema generates the Zigflow schema in-memory and compiles it
// for validation. The schema is never written to disk.
func buildCompiledSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	s := schema.BuildSchema("1.0.0", "json")

	schemaJSON, err := json.Marshal(s)
	require.NoError(t, err, "marshal schema to JSON")

	compiler := jsonschema.NewCompiler()
	err = compiler.AddResource("schema.json", bytes.NewReader(schemaJSON))
	require.NoError(t, err, "add schema resource to compiler")

	sch, err := compiler.Compile("schema.json")
	require.NoError(t, err, "compile schema")

	return sch
}

// validateYAML converts a YAML document to JSON in-memory and validates it
// against the compiled schema.
func validateYAML(sch *jsonschema.Schema, yamlData []byte) error {
	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		return err
	}

	var instance any
	if err := json.Unmarshal(jsonData, &instance); err != nil {
		return err
	}

	return sch.Validate(instance)
}

// fixtureFiles returns the paths of all .yaml files in the given directory.
func fixtureFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "read fixture directory %s", dir)

	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}

	return paths
}

func TestSchemaValidFixtures(t *testing.T) {
	sch := buildCompiledSchema(t)

	for _, path := range fixtureFiles(t, filepath.Join("testdata", "schema", "valid")) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)

			err = validateYAML(sch, data)
			assert.NoError(t, err, "fixture %s should be valid", filepath.Base(path))
		})
	}
}

func TestSchemaInvalidFixtures(t *testing.T) {
	sch := buildCompiledSchema(t)

	for _, path := range fixtureFiles(t, filepath.Join("testdata", "schema", "invalid")) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)

			err = validateYAML(sch, data)
			assert.Error(t, err, "fixture %s should be invalid", filepath.Base(path))
		})
	}
}

// validateWorkflow is a convenience helper for task-specific schema tests.
// Pass an inline YAML document; it returns the validation result.
// Add new cases to TestTaskSupportMatrix below to extend coverage over time.
func validateWorkflow(t *testing.T, yamlDoc string) error {
	t.Helper()
	return validateYAML(buildCompiledSchema(t), []byte(yamlDoc))
}

// TestTaskSupportMatrix contains per-task validation cases. Add a new entry
// whenever a task type is implemented or its schema changes.
func TestTaskSupportMatrix(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "set task - valid",
			yaml: `
document:
  dsl: 1.0.0
  namespace: zigflow
  name: set-matrix
  version: 0.0.1
do:
  - step:
      set:
        key: value
`,
		},
		{
			name: "wait task - ISO 8601 duration string",
			yaml: `
document:
  dsl: 1.0.0
  namespace: zigflow
  name: wait-matrix
  version: 0.0.1
do:
  - pause:
      wait: PT30S
`,
		},
		{
			name:    "set task - missing set field",
			wantErr: true,
			yaml: `
document:
  dsl: 1.0.0
  namespace: zigflow
  name: set-missing-field
  version: 0.0.1
do:
  - step: {}
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWorkflow(t, tc.yaml)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"sigs.k8s.io/yaml"
)

// CompileSchema builds the Zigflow workflow JSON Schema for the given DSL
// version and compiles it ready for validation. The schema is constructed
// in-memory and never written to disk.
func CompileSchema(version string) (*jsonschema.Schema, error) {
	s := BuildSchema(version, "json")

	schemaJSON, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", bytes.NewReader(schemaJSON)); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}

	return compiler.Compile("schema.json")
}

// ValidateFile reads a workflow file from disk and validates it against the
// compiled schema. YAML (.yaml, .yml) and JSON (.json) inputs are both
// supported; format is detected from the file extension.
func ValidateFile(sch *jsonschema.Schema, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	return validateData(sch, data, isJSONFile(path))
}

// validateData validates raw workflow document bytes against the compiled
// schema. Set jsonInput to true if the data is already JSON; false treats it
// as YAML and converts it before validation.
func validateData(sch *jsonschema.Schema, data []byte, jsonInput bool) error {
	jsonData := data

	if !jsonInput {
		var err error

		jsonData, err = yaml.YAMLToJSON(data)
		if err != nil {
			return fmt.Errorf("parse YAML: %w", err)
		}
	}

	var instance any
	if err := json.Unmarshal(jsonData, &instance); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	return sch.Validate(instance)
}

func isJSONFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".json"
}

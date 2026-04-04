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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zigflow/zigflow/pkg/schema"
)

const goldenFile = "testdata/schema/schema.golden.json"

// TestSchemaSnapshot compares the generated schema against the committed golden
// file. If the schema has changed intentionally, regenerate the golden file by
// running:
//
//	UPDATE_GOLDEN=1 go test ./pkg/schema/... -run TestSchemaSnapshot
func TestSchemaSnapshot(t *testing.T) {
	got, err := marshalSchemaJSON()
	require.NoError(t, err)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		dir := filepath.Dir(goldenFile)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(goldenFile, got, 0o600))
		t.Logf("golden file updated: %s", goldenFile)
		return
	}

	want, err := os.ReadFile(goldenFile)
	require.NoError(t, err, "golden file missing; generate it with: UPDATE_GOLDEN=1 go test ./pkg/schema/... -run TestSchemaSnapshot")

	assert.Equal(t, string(want), string(got),
		"schema snapshot changed; if intentional, regenerate with: UPDATE_GOLDEN=1 go test ./pkg/schema/... -run TestSchemaSnapshot",
	)
}

// TestSchemaStructure asserts that the key structural building blocks of the
// generated schema are present and have the expected shape. These checks are
// intentionally coarse: they protect against accidental deletions or renames
// without making the test brittle to minor wording changes.
func TestSchemaStructure(t *testing.T) {
	s := schema.BuildSchema("1.0.0", "json")

	t.Run("top-level $schema present", func(t *testing.T) {
		assert.Equal(t, schema.SchemaVersion, s.SchemaURI)
	})

	t.Run("top-level $defs present", func(t *testing.T) {
		assert.NotEmpty(t, s.Defs)
	})

	t.Run("$defs contains task", func(t *testing.T) {
		assert.Contains(t, s.Defs, "task")
	})

	t.Run("$defs contains metadata", func(t *testing.T) {
		assert.Contains(t, s.Defs, "metadata")
	})

	t.Run("top-level properties include do", func(t *testing.T) {
		assert.Contains(t, s.Properties, "do")
	})

	t.Run("top-level properties include document", func(t *testing.T) {
		assert.Contains(t, s.Properties, "document")
	})

	t.Run("task is a oneOf", func(t *testing.T) {
		task := s.Defs["task"]
		assert.NotEmpty(t, task.OneOf, "task should use oneOf for discriminated union")
	})

	t.Run("metadata is an object schema", func(t *testing.T) {
		meta := s.Defs["metadata"]
		assert.Equal(t, "object", meta.Type, "metadata should have type: object")
	})

	t.Run("do references taskList", func(t *testing.T) {
		do := s.Properties["do"]
		assert.Equal(t, "#/$defs/taskList", do.Ref, "do should reference #/$defs/taskList")
	})
}

// marshalSchemaJSON builds the schema and serialises it as pretty-printed JSON
// with a trailing newline, matching the format of the golden file.
func marshalSchemaJSON() ([]byte, error) {
	s := schema.BuildSchema("1.0.0", "json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

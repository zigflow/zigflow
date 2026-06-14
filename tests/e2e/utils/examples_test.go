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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	helloDataKey    = "data"
	helloMessageKey = "message"
	helloMessage    = "Hello from Ziggy"
)

// examplesDir resolves the repository's examples directory relative to this
// package (tests/e2e/utils).
func examplesDir(t *testing.T) string {
	t.Helper()

	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "examples"))
	require.NoError(t, err)
	return dir
}

func TestDiscoverExamples(t *testing.T) {
	cases, err := DiscoverExamples(examplesDir(t))
	require.NoError(t, err)
	require.NotEmpty(t, cases, "expected at least one example to opt into e2e testing")

	byName := make(map[string]TestCase, len(cases))
	for _, c := range cases {
		byName[c.Name] = c
	}

	hello, ok := byName["examples/hello-world"]
	require.True(t, ok, "hello-world example should be discovered via its test.yaml")

	// The discovered case must point at the workflow.yaml sat alongside the
	// test.yaml, and carry the expected output parsed from test.yaml.
	assert.Equal(t, "workflow.yaml", filepath.Base(hello.WorkflowPath))
	assert.Equal(t, map[string]any{
		helloDataKey: map[string]any{
			helloMessageKey: helloMessage,
		},
	}, hello.ExpectedOutput)
	assert.NotNil(t, hello.Test, "discovered example must have a test runner")
}

// TestExpectedOutputComparison proves the assertion used by the harness
// (assert.Equal, via assert.ObjectsAreEqual) passes on a match and fails when
// the workflow output differs from the expected output.
func TestExpectedOutputComparison(t *testing.T) {
	expected := map[string]any{
		helloDataKey: map[string]any{
			helloMessageKey: helloMessage,
		},
	}

	matching := map[string]any{
		helloDataKey: map[string]any{
			helloMessageKey: helloMessage,
		},
	}

	differing := map[string]any{
		helloDataKey: map[string]any{
			helloMessageKey: "Goodbye from Ziggy",
		},
	}

	assert.True(t, assert.ObjectsAreEqual(expected, matching), "identical output should match")
	assert.False(t, assert.ObjectsAreEqual(expected, differing), "differing output must fail the comparison")
}

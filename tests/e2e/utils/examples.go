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
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/mrsimonemms/golang-helpers/temporal"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"sigs.k8s.io/yaml"
)

// ExampleTest is the schema for an example's test.yaml file. It is the opt-in
// mechanism for example-based end-to-end testing: an example is included in the
// e2e run only if it contains a test.yaml alongside its workflow.yaml.
//
// Two assertion styles are supported and may be combined in the same file:
//
//   - Expected compares the workflow result exactly. Use it when the output is
//     fully deterministic.
//   - Assert validates the shape and type of the result without requiring exact
//     values. Use it for examples that produce variable data such as UUIDs,
//     timestamps or generated IDs. See CheckStructure for the assertion model.
//
// Compose-backed examples and Temporal-direct execution are out of scope.
type ExampleTest struct {
	// Input is passed to the workflow when it is started.
	Input map[string]any `json:"input"`
	// Expected is compared exactly against the workflow result. When nil the
	// exact-match check is skipped.
	Expected any `json:"expected"`
	// Assert is a structural/type assertion compared against the workflow
	// result. When nil the structural check is skipped.
	Assert any `json:"assert"`
}

// DiscoverExamples walks examplesDir looking for test.yaml files. Each one is
// turned into a TestCase that runs the workflow.yaml in the same directory and
// asserts the result matches the expected output.
//
// Examples without a test.yaml file are ignored. An example with a test.yaml
// but no workflow.yaml is an error, because the opt-in is incomplete.
func DiscoverExamples(examplesDir string) ([]TestCase, error) {
	cases := make([]TestCase, 0)

	// Walk via os.Root so all reads are confined to examplesDir. This avoids the
	// symlink traversal race that a bare os.ReadFile inside a WalkDir callback
	// would otherwise allow.
	root, err := os.OpenRoot(examplesDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "test.yaml" {
			return nil
		}

		dir := filepath.Dir(p)
		workflowRel := filepath.Join(dir, "workflow.yaml")
		if _, statErr := root.Stat(workflowRel); statErr != nil {
			return fmt.Errorf("example %q has a test.yaml but no workflow.yaml: %w", dir, statErr)
		}

		data, err := root.ReadFile(p)
		if err != nil {
			return err
		}

		var et ExampleTest
		if err := yaml.Unmarshal(data, &et); err != nil {
			return fmt.Errorf("parsing %q: %w", p, err)
		}

		cases = append(cases, TestCase{
			Name:           "examples/" + filepath.Base(dir),
			WorkflowPath:   filepath.Join(examplesDir, workflowRel),
			Input:          et.Input,
			ExpectedOutput: et.Expected,
			Test:           exampleRunner(et),
			Example:        true,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return cases, nil
}

// exampleRunner builds the test function for a discovered example. It executes
// the workflow to completion and then applies whichever assertions the
// test.yaml declared: an exact match against Expected, a structural match
// against Assert, or both. A test.yaml with neither still asserts that the
// workflow runs to completion without error.
func exampleRunner(et ExampleTest) func(t *testing.T, test *TestCase) {
	return func(t *testing.T, test *TestCase) {
		result := executeExampleWorkflow(t, test)

		if et.Expected != nil {
			assert.Equal(t, et.Expected, result, "workflow output did not match the expected output")
		}

		if et.Assert != nil {
			assert.NoError(t, CheckStructure(et.Assert, result), "workflow output did not match the structural assertions")
		}
	}
}

// executeExampleWorkflow runs the example's workflow on its task queue and
// returns the decoded result. The result is decoded into an untyped value so
// that both exact and structural assertions can be applied to it.
func executeExampleWorkflow(t *testing.T, test *TestCase) any {
	t.Helper()

	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&zlog.Logger),
	)
	require.NoError(t, err)
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: test.Workflow.Document.Namespace,
	}

	wCtx := context.Background()

	we, err := c.ExecuteWorkflow(wCtx, workflowOptions, test.Workflow.Document.Name, test.Input)
	require.NoError(t, err)

	var result any
	require.NoError(t, we.Get(wCtx, &result))

	return result
}

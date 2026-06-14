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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// ExampleTest is the minimal schema for an example's test.yaml file. It is the
// opt-in mechanism for example-based end-to-end testing: an example is included
// in the e2e run only if it contains a test.yaml alongside its workflow.yaml.
//
// The schema is intentionally small. Compose-backed examples, Temporal-direct
// execution, partial assertions and advanced metadata are not supported yet.
type ExampleTest struct {
	// Input is passed to the workflow when it is started.
	Input map[string]any `json:"input"`
	// Expected is compared exactly against the workflow result.
	Expected any `json:"expected"`
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
			Test:           RunToCompletion[any],
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return cases, nil
}

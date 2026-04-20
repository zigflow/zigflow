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

package zigflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/schema"
	"sigs.k8s.io/yaml"
)

// LoadFromBytes parses and normalises a workflow definition from raw YAML or
// JSON bytes. It does not perform schema validation.
//
// Call ValidateBytes before LoadFromBytes when schema enforcement is required.
func LoadFromBytes(data []byte) (*model.Workflow, error) {
	jsonBytes, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	// normalise the Zigflow data structure to Serverless Workflow data structure
	var raw map[string]any
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("error unmarshalling to zigflow raw workflow: %w", err)
	}

	if err := normaliseWorkflowDocument(raw); err != nil {
		return nil, fmt.Errorf("error normalising workflow document: %w", err)
	}

	// Convert back to JSON
	normalisedJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("error marshalling raw workflow to json: %w", err)
	}

	// Now convert to Serverless Workflow's Workflow model
	var wf *model.Workflow
	if err := json.Unmarshal(normalisedJSON, &wf); err != nil {
		return nil, fmt.Errorf("error unmarshaling json to workflow: %w", err)
	}

	if err := newWorkflowPostLoad(wf); err != nil {
		return nil, fmt.Errorf("error preparing workflow: %w", err)
	}

	c, err := semver.NewConstraint(">= 1.0.0, <2.0.0")
	if err != nil {
		return nil, fmt.Errorf("error creating semver constraint: %w", err)
	}

	v, err := semver.NewVersion(wf.Document.DSL)
	if err != nil {
		return nil, fmt.Errorf("error creating semver version: %w", err)
	}

	if !c.Check(v) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDSL, wf.Document.DSL)
	}

	return wf, nil
}

// ValidateBytes validates raw YAML or JSON bytes against the Zigflow JSON
// Schema. It returns ErrSchemaValidation (via errors.Is) if the document does
// not conform.
func ValidateBytes(data []byte) error {
	jsonBytes, err := yaml.YAMLToJSON(data)
	if err != nil {
		return fmt.Errorf("error converting yaml to json: %w", err)
	}

	var rawDoc map[string]any
	if err := json.Unmarshal(jsonBytes, &rawDoc); err != nil {
		return fmt.Errorf("error parsing workflow document: %w", err)
	}

	if err := schema.ValidateDocument(rawDoc); err != nil {
		return fmt.Errorf("%w: %s", ErrSchemaValidation, err)
	}

	return nil
}

// LoadFromFile reads a workflow definition from file, maps Zigflow-specific
// field names (workflowType, taskQueue) onto the SDK model, and returns a
// parsed *model.Workflow. It does not perform schema validation.
//
// Call ValidateFile before LoadFromFile when schema enforcement is required,
// such as in CLI validation paths or when running with --validate=true.
func LoadFromFile(file string) (*model.Workflow, error) {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("error loading file: %w", err)
	}

	return LoadFromBytes(data)
}

// ValidateFile validates the workflow file at path against the Zigflow JSON
// Schema. It returns ErrSchemaValidation (via errors.Is) if the document does
// not conform, including when legacy fields (document.name, document.namespace)
// are present or required fields (document.workflowType, document.taskQueue)
// are absent.
//
// This function is intentionally separate from LoadFromFile so that callers
// control whether schema enforcement runs. CLI validation paths (zigflow
// validate, zigflow run --validate=true) call it explicitly; tooling paths
// (graph generation, pre-commit hooks) do not.
func ValidateFile(file string) error {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return fmt.Errorf("error loading file: %w", err)
	}

	return ValidateBytes(data)
}

func normaliseDoTask(task map[string]any) error {
	raw, ok := task["do"]
	if !ok {
		return nil
	}
	return normaliseTaskList(raw)
}

func normaliseForTask(task map[string]any) error {
	raw, ok := task["do"]
	if !ok {
		return nil
	}
	// for tasks also carry nested task list in top-level "do"
	return normaliseTaskList(raw)
}

func normaliseForkTask(task map[string]any) error {
	rawFork, ok := task["fork"]
	if !ok {
		return nil
	}

	fork, ok := rawFork.(map[string]any)
	if !ok {
		return fmt.Errorf("fork must be an object")
	}

	rawBranches, ok := fork["branches"]
	if !ok {
		return nil
	}

	return normaliseTaskList(rawBranches)
}

func normaliseRunTask(task map[string]any) error {
	rawRun, ok := task["run"]
	if !ok {
		return nil
	}

	run, ok := rawRun.(map[string]any)
	if !ok {
		return fmt.Errorf("run must be an object")
	}

	rawWorkflow, ok := run["workflow"]
	if !ok {
		return nil
	}

	workflow, ok := rawWorkflow.(map[string]any)
	if !ok {
		return fmt.Errorf("run.workflow must be an object")
	}

	renameKey(workflow, "type", "name")

	return nil
}

func normaliseTask(task map[string]any) error {
	fns := []func(map[string]any) error{
		// Tasks that can contain task lists
		normaliseDoTask,
		normaliseForTask,
		normaliseForkTask,
		normaliseTryTask,

		// Tasks that need normalising
		normaliseRunTask,
	}

	for _, fn := range fns {
		if err := fn(task); err != nil {
			return err
		}
	}

	return nil
}

func normaliseTaskList(raw any) error {
	taskList, ok := raw.([]any)
	if !ok {
		return nil
	}

	for i, task := range taskList {
		namedTask, ok := task.(map[string]any)
		if !ok {
			return fmt.Errorf("task item %d must be an object", i)
		}

		for taskName, rawTask := range namedTask {
			task, ok := rawTask.(map[string]any)
			if !ok {
				return fmt.Errorf("task %q must be an object", taskName)
			}

			if err := normaliseTask(task); err != nil {
				return fmt.Errorf("normalise task %q: %w", taskName, err)
			}
		}
	}

	return nil
}

func normaliseTopLevelDocument(doc map[string]any) error {
	rawDocument, ok := doc["document"]
	if !ok {
		return nil
	}

	document, ok := rawDocument.(map[string]any)
	if !ok {
		return fmt.Errorf("document must be an object")
	}

	renameKey(document, "workflowType", "name")
	renameKey(document, "taskQueue", "namespace")

	return nil
}

func normaliseTryTask(task map[string]any) error {
	rawTry, ok := task["try"]
	if ok {
		if err := normaliseTaskList(rawTry); err != nil {
			return err
		}
	}

	rawCatch, ok := task["catch"]
	if !ok {
		return nil
	}

	catchObj, ok := rawCatch.(map[string]any)
	if !ok {
		return fmt.Errorf("catch must be an object")
	}

	rawCatchDo, ok := catchObj["do"]
	if !ok {
		return nil
	}

	return normaliseTaskList(rawCatchDo)
}

func normaliseWorkflowDocument(doc map[string]any) error {
	if err := normaliseTopLevelDocument(doc); err != nil {
		return err
	}

	rawTasks, ok := doc["do"]
	if !ok {
		return nil
	}

	return normaliseTaskList(rawTasks)
}

func renameKey(m map[string]any, oldKey, newKey string) {
	val, ok := m[oldKey]
	if !ok {
		return
	}

	if _, exists := m[newKey]; !exists {
		m[newKey] = val
	}

	delete(m, oldKey)
}

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

// zigflowDocFields holds the Zigflow-specific document fields that differ from
// the upstream Serverless Workflow SDK. The SDK uses "name" and "namespace";
// Zigflow uses "workflowType" and "taskQueue" to align with Temporal concepts.
// This struct is used only to extract those fields after JSON unmarshal so that
// they can be mapped onto the SDK's model.Document fields.
type zigflowDocFields struct {
	WorkflowType string `json:"workflowType"`
	TaskQueue    string `json:"taskQueue"`
}

type zigflowRawDoc struct {
	Document zigflowDocFields `json:"document"`
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

	var jsonBytes []byte
	if jsonBytes, err = yaml.YAMLToJSON(data); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	var wf *model.Workflow
	if err := json.Unmarshal(jsonBytes, &wf); err != nil {
		return nil, fmt.Errorf("error unmarshaling json to workflow: %w", err)
	}

	// The SDK's Document struct uses json:"name" and json:"namespace", but
	// Zigflow's schema uses "workflowType" and "taskQueue". Extract those
	// fields from the raw JSON and map them to the SDK's fields so that all
	// downstream code continues to work unchanged.
	var raw zigflowRawDoc
	if err := json.Unmarshal(jsonBytes, &raw); err == nil {
		if raw.Document.WorkflowType != "" {
			wf.Document.Name = raw.Document.WorkflowType
		}
		if raw.Document.TaskQueue != "" {
			wf.Document.Namespace = raw.Document.TaskQueue
		}
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

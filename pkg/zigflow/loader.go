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
	"sigs.k8s.io/yaml"
)

func LoadFromFile(file string) (*model.Workflow, error) {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("error loading file: %w", err)
	}

	// Load the workflow without validating - we'll do that later
	var jsonBytes []byte
	if jsonBytes, err = yaml.YAMLToJSON(data); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	var wf *model.Workflow
	if err := json.Unmarshal(jsonBytes, &wf); err != nil {
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

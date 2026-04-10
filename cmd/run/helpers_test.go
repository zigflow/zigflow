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

package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
)

// minimalWorkflowYAML returns the smallest valid workflow YAML for the given
// taskQueue and workflowType. The document version and DSL are fixed to keep
// fixtures short.
func minimalWorkflowYAML(namespace, name string) string {
	return `document:
  dsl: 1.0.0
  taskQueue: ` + namespace + `
  workflowType: ` + name + `
  version: 0.0.1
do:
  - noop:
      set:
        set: {}
`
}

// writeTempWorkflow writes a minimal workflow YAML file into dir and returns
// the absolute path. It fails the test immediately on any write error.
func writeTempWorkflow(t *testing.T, dir, namespace, name string) string {
	t.Helper()
	p := filepath.Join(dir, namespace+"."+name+".yaml")
	require.NoError(t, os.WriteFile(p, []byte(minimalWorkflowYAML(namespace, name)), 0o600))
	return p
}

func newTestValidator(t *testing.T) *utils.Validator {
	t.Helper()
	v, err := utils.NewValidator()
	require.NoError(t, err)
	return v
}

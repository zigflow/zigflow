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

package zigflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

func TestLoadWorkflowFile(t *testing.T) {
	tests := []struct {
		Name        string
		Content     string
		Error       error
		ExpectError bool
	}{
		{
			Name: "Load valid workflow file",
			Content: `document:
  dsl: 1.0.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
		{
			Name: "Invalid DSL version",
			Content: `document:
  dsl: 0.9.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
			Error:       zigflow.ErrUnsupportedDSL,
			ExpectError: true,
		},
		{
			Name:        "Invalid YAML",
			Content:     `invalid content: [`,
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "workflow_test")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, os.RemoveAll(tmpDir))
			}()

			filePath := filepath.Join(tmpDir, "zigflow.yaml")
			err = os.WriteFile(filePath, []byte(test.Content), 0o600)
			assert.NoError(t, err)

			workflow, err := zigflow.LoadFromFile(filePath)
			if test.ExpectError {
				assert.Error(t, err)
				assert.Nil(t, workflow)

				if test.Error != nil {
					assert.ErrorIs(t, err, test.Error)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workflow)
			}
		})
	}
}

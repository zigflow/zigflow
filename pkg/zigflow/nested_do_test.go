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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const nestedDoHeader = `document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: nested
  version: 0.0.1
`

func nestedDoWarningPaths(t *testing.T, body string) []string {
	t.Helper()

	doc, err := LoadFromBytes([]byte(nestedDoHeader + body))
	require.NoError(t, err)

	paths := []string{}
	for _, w := range NestedDoWarnings(doc) {
		assert.Equal(t, WarningCodeNestedDoDefinition, w.Code)
		paths = append(paths, w.Path)
	}
	return paths
}

func TestNestedDoWarnings(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "do after an executable task is flagged",
			body: `do:
  - initData:
      set:
        version: "1.0.0"
  - handleError:
      do:
        - fault:
            set:
              failed: true
`,
			want: []string{"$.do[1]"},
		},
		{
			name: "only do tasks define multiple workflows",
			body: `do:
  - first:
      do:
        - step:
            set:
              a: 1
  - second:
      do:
        - step:
            set:
              b: 2
`,
			want: []string{},
		},
		{
			name: "do before executable tasks is flagged too",
			body: `do:
  - group:
      do:
        - step:
            set:
              a: 1
  - after:
      set:
        b: 2
`,
			want: []string{"$.do[0]"},
		},
		{
			name: "switch targets are skipped, task-level then targets are not",
			body: `do:
  - pick:
      switch:
        - yes:
            when: ${ $input.ok }
            then: onYes
        - default:
            then: continue
  - jump:
      set:
        a: 1
      then: onJump
  - onYes:
      do:
        - step:
            set:
              b: 2
  - onJump:
      do:
        - step:
            set:
              c: 3
  - orphan:
      do:
        - step:
            set:
              d: 4
`,
			want: []string{"$.do[3]", "$.do[4]"},
		},
		{
			name: "nested inside a do workflow",
			body: `do:
  - main:
      do:
        - step:
            set:
              a: 1
        - inner:
            do:
              - other:
                  set:
                    b: 2
`,
			want: []string{"$.do[0].main.do[1]"},
		},
		{
			name: "inside try and catch",
			body: `do:
  - guard:
      try:
        - step:
            set:
              a: 1
        - inTry:
            do:
              - other:
                  set:
                    b: 2
      catch:
        do:
          - handle:
              set:
                c: 3
          - inCatch:
              do:
                - other:
                    set:
                      d: 4
`,
			want: []string{"$.do[0].guard.try[1]", "$.do[0].guard.catch.do[1]"},
		},
		{
			name: "inside a multi-task fork branch",
			body: `do:
  - parallel:
      fork:
        branches:
          - left:
              do:
                - step:
                    set:
                      a: 1
                - inBranch:
                    do:
                      - other:
                          set:
                            b: 2
          - right:
              set:
                c: 3
`,
			want: []string{"$.do[0].parallel.fork.branches[0].left.do[1]"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, nestedDoWarningPaths(t, tc.body))
		})
	}
}

func TestNestedDoWarningsNilDocument(t *testing.T) {
	assert.Nil(t, NestedDoWarnings(nil))
}

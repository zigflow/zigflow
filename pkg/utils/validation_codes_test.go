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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeForPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "schema instance path for task queue",
			path: "$.document.taskQueue",
			want: CodeInvalidTaskQueue,
		},
		{
			name: "struct namespace for task queue",
			path: "Workflow.Document.Namespace",
			want: CodeInvalidTaskQueue,
		},
		{
			name: "schema instance path for workflow type",
			path: "$.document.workflowType",
			want: CodeInvalidWorkflowType,
		},
		{
			name: "struct namespace for workflow type",
			path: "Workflow.Document.Name",
			want: CodeInvalidWorkflowType,
		},
		{
			name: "unrecognised path has no code",
			path: "$.document",
			want: "",
		},
		{
			name: "empty path has no code",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CodeForPath(tt.path))
		})
	}
}

func TestDocumentationURL(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "task queue code derives slug",
			code: CodeInvalidTaskQueue,
			want: "https://zigflow.dev/errors/invalid-task-queue",
		},
		{
			name: "workflow type code derives slug",
			code: CodeInvalidWorkflowType,
			want: "https://zigflow.dev/errors/invalid-workflow-type",
		},
		{
			name: "dsl version code derives slug",
			code: CodeInvalidDSLVersion,
			want: "https://zigflow.dev/errors/invalid-dsl-version",
		},
		{
			name: "empty code yields empty URL",
			code: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DocumentationURL(tt.code))
		})
	}
}

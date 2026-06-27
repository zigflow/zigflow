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
	"sort"
	"strings"
)

// Stable validation error codes. A code is a durable identifier for a class of
// validation failure. It is additive metadata: the underlying validation
// message is unchanged, but consumers can map the code onto documentation
// without parsing message text or inferring intent.
//
// Codes are intentionally tied to the failing field rather than the specific
// rule, so the same logical problem yields the same code regardless of which
// validation stage (schema or struct) detects it.
const (
	CodeInvalidTaskQueue    = "ERR_INVALID_TASK_QUEUE"
	CodeInvalidWorkflowType = "ERR_INVALID_WORKFLOW_TYPE"
	CodeInvalidDSLVersion   = "ERR_INVALID_DSL_VERSION"
	CodeInvalidVersion      = "ERR_INVALID_VERSION"
)

// errorDocumentationBaseURL is the stable landing area for validation error
// documentation. Pages may initially redirect into the existing documentation.
const errorDocumentationBaseURL = "https://zigflow.dev/errors/"

// validationCodeByPath maps a validation path onto a stable error code. Both
// the schema stage (instance paths such as "$.document.taskQueue") and the
// struct stage (struct namespaces such as "Workflow.Document.Namespace") are
// registered so the same field maps to the same code regardless of which stage
// reports it. The lookup is exact: there is no fuzzy matching and no inference
// from message text.
var validationCodeByPath = map[string]string{
	// Schema stage: best-effort instance paths produced by schema validation.
	"$.document.taskQueue":    CodeInvalidTaskQueue,
	"$.document.workflowType": CodeInvalidWorkflowType,
	"$.document.dsl":          CodeInvalidDSLVersion,
	"$.document.version":      CodeInvalidVersion,

	// Struct stage: go-playground/validator struct namespaces. taskQueue and
	// workflowType are mapped onto the SDK's Namespace and Name fields by the
	// loader, so the namespaces differ from the document field names.
	"Workflow.Document.Namespace": CodeInvalidTaskQueue,
	"Workflow.Document.Name":      CodeInvalidWorkflowType,
	"Workflow.Document.DSL":       CodeInvalidDSLVersion,
	"Workflow.Document.Version":   CodeInvalidVersion,
}

// ErrorCodes returns the unique set of stable validation error codes the
// validator can emit, sorted for deterministic output. It is derived from the
// same registry used to enrich validation errors, so it cannot drift from the
// codes that are actually produced.
func ErrorCodes() []string {
	seen := make(map[string]struct{}, len(validationCodeByPath))
	codes := make([]string, 0, len(validationCodeByPath))
	for _, code := range validationCodeByPath {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}

	sort.Strings(codes)

	return codes
}

// CodeForPath returns the stable validation error code for a validation path,
// or an empty string when the path is not recognised. An empty string means
// "no stable documentation page", and callers should omit documentation rather
// than invent it.
func CodeForPath(path string) string {
	return validationCodeByPath[path]
}

// DocumentationURL derives the documentation URL for a stable error code. The
// code is the single source of truth: the URL is derived from it rather than
// stored alongside it. An empty code yields an empty URL so callers can omit
// the documentation field for unrecognised errors.
//
// For example, "ERR_INVALID_TASK_QUEUE" becomes
// "https://zigflow.dev/errors/invalid-task-queue".
func DocumentationURL(code string) string {
	if code == "" {
		return ""
	}

	slug := strings.ToLower(strings.TrimPrefix(code, "ERR_"))
	slug = strings.ReplaceAll(slug, "_", "-")

	return errorDocumentationBaseURL + slug
}

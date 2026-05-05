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

// Test constants shared across the cmd/run test suite to satisfy the goconst
// linter, which requires string literals appearing 3+ times to be named.
const (
	testEnableVersioningTrue = "true"
	testWorkflowPathA        = "/a/workflow.yaml"
	testDirectoryGlob        = "*.yaml"
	testTemporalServerName   = "your-namespace.tmprl.cloud"
	testSourceFileA          = "a.yaml"
	testSourceFileB          = "b.yaml"
	testWorkflowType         = "wf"
)

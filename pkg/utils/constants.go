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

// State variable names Zigflow injects into every runtime (jq) expression
// evaluation from workflow State (see State.GetAsMap).
//
// These names are a public contract: workflow definitions reference them
// directly (${ $data.foo }, ${ $env.BAR }, ...), so changing a value silently
// changes how every workflow's expressions resolve. They are guarded by a test
// and must not be changed without treating it as a breaking change.
const (
	StateVarContext = "$context"
	StateVarData    = "$data"
	StateVarEnv     = "$env"
	StateVarInput   = "$input"
	StateVarOutput  = "$output"
)

// StateVarNames is the canonical, ordered set of state variable names. Derived
// structures (the determinism allow-list, membership checks) are built from
// this single list so they cannot drift from the constants above or from each
// other.
var StateVarNames = []string{
	StateVarContext,
	StateVarData,
	StateVarEnv,
	StateVarInput,
	StateVarOutput,
}

// ActivityStateKey is the $data key under which activity-runtime metadata is
// exposed to runtime expressions (see State.AddActivityInfo). Expressions that
// read $data.activity.* depend on state that only exists once an activity is
// executing, so they cannot be resolved workflow-side before scheduling.
//
// This is a separate contract from the top-level state variable names above: it
// names a key nested under $data, not a variable injected into evaluation.
const ActivityStateKey = "activity"

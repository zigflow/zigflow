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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStateVarNamesContract pins the literal values of the state variable
// names. These are a public contract referenced directly by workflow
// definitions (${ $data.foo }, ${ $env.BAR }, ...), so a change here would
// silently break every workflow that uses expressions. The hard-coded
// expectations are intentional: this test must fail if a constant value drifts,
// forcing the change to be a deliberate, reviewed decision.
func TestStateVarNamesContract(t *testing.T) {
	assert.Equal(t, "$context", StateVarContext)
	assert.Equal(t, "$data", StateVarData)
	assert.Equal(t, "$env", StateVarEnv)
	assert.Equal(t, "$input", StateVarInput)
	assert.Equal(t, "$output", StateVarOutput)

	assert.Equal(t, []string{
		"$context",
		"$data",
		"$env",
		"$input",
		"$output",
	}, StateVarNames, "StateVarNames must list exactly the state variables, in order")
}

// TestGetAsMapInjectsExactlyStateVarNames guards the wiring between the contract
// and what is actually injected: State.GetAsMap must expose exactly the
// variables named in StateVarNames, no more and no fewer. This catches a name
// that is renamed in one place but not the other, which would otherwise only
// surface as expressions resolving to null at runtime.
func TestGetAsMapInjectsExactlyStateVarNames(t *testing.T) {
	got := make([]string, 0)
	for key := range NewState().GetAsMap() {
		got = append(got, key)
	}
	sort.Strings(got)

	want := append([]string(nil), StateVarNames...)
	sort.Strings(want)

	assert.Equal(t, want, got, "GetAsMap keys must match StateVarNames exactly")
}

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

package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubExtension is a tiny Extension used by tests. The claim predicate is
// configurable per-test so each test can craft its own claim logic.
type stubExtension struct {
	taskType  string
	claimFunc func(any) bool
}

func (s stubExtension) TaskType() string     { return s.taskType }
func (s stubExtension) Claims(body any) bool { return s.claimFunc(body) }

// withFreshRegistry resets the global registry for the duration of the
// test and restores it afterwards. White-box access is intentional: the
// registry is global state populated by extension init() blocks, and
// tests need to drive it deterministically.
func withFreshRegistry(t *testing.T) {
	t.Helper()
	saved := registry
	registry = nil
	t.Cleanup(func() { registry = saved })
}

func TestRegister_AddsToRegistry(t *testing.T) {
	withFreshRegistry(t)

	ext := stubExtension{taskType: "wait", claimFunc: func(any) bool { return true }}
	Register(ext)

	require.Len(t, registry, 1, "extension must be added to the registry")
	assert.Equal(t, "wait", registry[0].TaskType())
}

func TestRegister_PanicsOnEmptyTaskType(t *testing.T) {
	withFreshRegistry(t)

	assert.PanicsWithValue(t, "extensions: TaskType must not be empty", func() {
		Register(stubExtension{taskType: "", claimFunc: func(any) bool { return true }})
	})
}

func TestRegister_PanicsOnDuplicateTaskType(t *testing.T) {
	withFreshRegistry(t)

	Register(stubExtension{taskType: "wait", claimFunc: func(any) bool { return true }})

	assert.Panics(t, func() {
		Register(stubExtension{taskType: "wait", claimFunc: func(any) bool { return true }})
	}, "registering a second extension for the same task type must panic")
}

func TestNormalise_ClaimsTaskWhenExtensionMatches(t *testing.T) {
	withFreshRegistry(t)

	Register(stubExtension{
		taskType:  "wait",
		claimFunc: func(body any) bool { _, ok := body.(map[string]any); return ok },
	})

	task := map[string]any{
		"wait": map[string]any{"until": "2026-12-31T23:59:59Z"},
		"if":   "${ true }",
	}

	Normalise(task)

	assert.NotContains(t, task, "wait", "claimed task must no longer carry the task type")
	require.Contains(t, task, ZigflowExtKeyPrefix+"wait", "claimed task must carry the Zigflow key")
	assert.Equal(t, map[string]any{"until": "2026-12-31T23:59:59Z"}, task[ZigflowExtKeyPrefix+"wait"])
	assert.Equal(t, "${ true }", task["if"], "unrelated task fields must be left alone")
}

func TestNormalise_LeavesTaskAloneWhenExtensionDoesNotClaim(t *testing.T) {
	withFreshRegistry(t)

	Register(stubExtension{
		taskType:  "wait",
		claimFunc: func(any) bool { return false },
	})

	body := map[string]any{"seconds": 5}
	task := map[string]any{"wait": body}

	Normalise(task)

	assert.Contains(t, task, "wait", "unclaimed task must keep its original task type")
	assert.NotContains(t, task, ZigflowExtKeyPrefix+"wait", "unclaimed task must not be renamed")
	assert.Equal(t, body, task["wait"], "unclaimed body must be preserved unchanged")
}

func TestNormalise_NoMatchingExtensionLeavesTaskAlone(t *testing.T) {
	withFreshRegistry(t)

	Register(stubExtension{
		taskType:  "wait",
		claimFunc: func(any) bool { return true },
	})

	task := map[string]any{"set": map[string]any{"x": "y"}}

	Normalise(task)

	assert.Equal(t, map[string]any{"set": map[string]any{"x": "y"}}, task)
}

func TestNormalise_FirstMatchWins(t *testing.T) {
	withFreshRegistry(t)

	Register(stubExtension{
		taskType:  "wait",
		claimFunc: func(any) bool { return true },
	})
	// The second registration on a different task type is fine; we want to
	// verify that the iterator only acts on extensions whose TaskType is
	// actually present in the task body.
	Register(stubExtension{
		taskType:  "set",
		claimFunc: func(any) bool { return true },
	})

	task := map[string]any{"wait": map[string]any{"seconds": 5}}
	Normalise(task)

	assert.Contains(t, task, ZigflowExtKeyPrefix+"wait")
	assert.NotContains(t, task, ZigflowExtKeyPrefix+"set", "extension whose TaskType is absent must not run")
}

func TestNormalise_EmptyRegistryIsSafe(t *testing.T) {
	withFreshRegistry(t)

	task := map[string]any{"wait": map[string]any{"seconds": 5}}
	Normalise(task)

	assert.Equal(t, map[string]any{"wait": map[string]any{"seconds": 5}}, task)
}

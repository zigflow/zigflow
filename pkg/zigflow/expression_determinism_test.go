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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

const workflowHeader = `document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: test
  version: 0.0.1
`

// TestLoadFromBytes_RejectsNonDeterministicAcrossTaskTypes proves the central
// determinism pass catches non-determinism in every expression-bearing task
// type, not just wait_ext. Each case uses the same offending symbol so the
// only variable is the surrounding task type.
func TestLoadFromBytes_RejectsNonDeterministicAcrossTaskTypes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "wait extension seconds field",
			body: `do:
  - waitForCooldown:
      wait:
        seconds: ${ $data.delay + timestamp }
`,
		},
		{
			name: "wait extension until field",
			body: `do:
  - waitUntil:
      wait:
        until: ${ timestamp_iso8601 }
`,
		},
		{
			name: "call http endpoint",
			body: `do:
  - getUser:
      call: http
      with:
        method: get
        endpoint: ${ "https://example.com/" + uuid }
`,
		},
		{
			name: "for in expression",
			body: `do:
  - loop:
      for:
        each: item
        in: ${ [uuid, $data.x] }
      do:
        - body:
            set:
              v: ${ $data.x }
`,
		},
		{
			name: "for while expression",
			body: `do:
  - loop:
      for:
        each: item
        in: ${ $data.items }
      while: ${ now < 1000 }
      do:
        - body:
            set:
              v: ${ $data.x }
`,
		},
		{
			name: "switch when expression",
			body: `do:
  - decide:
      switch:
        - case1:
            when: ${ $data.x == timestamp }
            then: continue
        - default:
            then: continue
`,
		},
		{
			name: "task base if expression",
			body: `do:
  - skipMaybe:
      if: ${ now > 1000 }
      set:
        v: ${ $data.x }
`,
		},
		{
			name: "task base output expression",
			body: `do:
  - withOutput:
      output:
        as: ${ uuid }
      set:
        v: ${ $data.x }
`,
		},
		{
			name: "task base export expression",
			body: `do:
  - withExport:
      export:
        as: ${ timestamp }
      set:
        v: ${ $data.x }
`,
		},
		{
			name: "task base metadata expression",
			body: `do:
  - withMeta:
      metadata:
        traceId: ${ uuid }
      set:
        v: ${ $data.x }
`,
		},
		{
			name: "run task workflow input",
			body: `do:
  - subflow:
      run:
        workflow:
          namespace: default
          type: child
          version: 0.0.1
          input:
            id: ${ uuid }
`,
		},
		{
			name: "try catch when expression",
			body: `do:
  - guarded:
      try:
        - inner:
            set:
              v: ${ $data.x }
      catch:
        when: ${ now > 0 }
        do:
          - handler:
              set:
                v: ${ $data.x }
`,
		},
		{
			name: "try catch.do task base output",
			body: `do:
  - guarded:
      try:
        - inner:
            set:
              v: ${ $data.x }
      catch:
        do:
          - handler:
              output:
                as: ${ uuid }
              set:
                v: ${ $data.x }
`,
		},
		{
			name: "fork branch task non-deterministic",
			body: `do:
  - parallel:
      fork:
        branches:
          - branchA:
              wait:
                seconds: ${ timestamp }
          - branchB:
              set:
                v: ${ $data.x }
`,
		},
		{
			name: "nested do task non-deterministic",
			body: `do:
  - outer:
      do:
        - inner:
            wait:
              seconds: ${ timestamp + $data.delay }
`,
		},
		{
			name: "workflow output as",
			body: `output:
  as: ${ timestamp }
do:
  - step:
      set:
        v: ${ $data.x }
`,
		},
		{
			name: "workflow input from",
			body: `input:
  from: ${ uuid }
do:
  - step:
      set:
        v: ${ $data.x }
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yaml := workflowHeader + tc.body
			_, err := zigflow.LoadFromBytes([]byte(yaml))
			require.Error(t, err, "non-deterministic expression must be rejected")
			assert.ErrorIs(t, err, zigflow.ErrNonDeterministicExpression,
				"error must be %v; got %v", zigflow.ErrNonDeterministicExpression, err)
		})
	}
}

// TestLoadFromBytes_AllowsNonDeterminismInsideSetBody proves the same symbols
// that fail outside Set tasks succeed when wrapped in a Set body.
func TestLoadFromBytes_AllowsNonDeterminismInsideSetBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "bare uuid in set",
			body: `do:
  - capture:
      set:
        id: ${ uuid }
`,
		},
		{
			name: "timestamp in set",
			body: `do:
  - capture:
      set:
        now: ${ timestamp }
`,
		},
		{
			name: "nested timestamp in set arithmetic",
			body: `do:
  - capture:
      set:
        delayedAt: ${ $data.delay + timestamp }
`,
		},
		{
			name: "uuid deep inside set object",
			body: `do:
  - capture:
      set:
        meta:
          ids:
            - ${ uuid }
            - ${ uuid }
`,
		},
		{
			name: "mixed set expressions",
			body: `do:
  - capture:
      set:
        id: ${ uuid }
        ts: ${ timestamp }
        derived: ${ $data.base + 1 }
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yaml := workflowHeader + tc.body
			wf, err := zigflow.LoadFromBytes([]byte(yaml))
			require.NoError(t, err, "set body must allow non-determinism")
			require.NotNil(t, wf)
		})
	}
}

// TestLoadFromBytes_SetTaskBaseFieldsMustBeDeterministic proves that the Set
// exemption only covers the set body — TaskBase fields on a Set task are
// still evaluated outside the side effect and must remain replay-safe.
func TestLoadFromBytes_SetTaskBaseFieldsMustBeDeterministic(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "if on set task",
			body: `do:
  - capture:
      if: ${ now > 0 }
      set:
        id: ${ uuid }
`,
		},
		{
			name: "output on set task",
			body: `do:
  - capture:
      output:
        as: ${ timestamp }
      set:
        id: ${ uuid }
`,
		},
		{
			name: "metadata on set task",
			body: `do:
  - capture:
      metadata:
        marker: ${ now }
      set:
        id: ${ uuid }
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yaml := workflowHeader + tc.body
			_, err := zigflow.LoadFromBytes([]byte(yaml))
			require.Error(t, err, "set task base fields must be deterministic")
			assert.ErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
		})
	}
}

// TestLoadFromBytes_AllowsDeterministicExpressions confirms the existing
// expression catalogue continues to load. Without this, regressions in the
// analyser's allow-list could silently break valid workflows.
func TestLoadFromBytes_AllowsDeterministicExpressions(t *testing.T) {
	yaml := workflowHeader + `do:
  - waitForCooldown:
      wait:
        seconds: ${ $data.delay + 10 }
  - loop:
      for:
        each: item
        in: ${ $data.items }
      do:
        - body:
            set:
              v: ${ $data.x }
  - getUser:
      call: http
      with:
        method: get
        endpoint: ${ "https://example.com/" + ($input.userId | tostring) }
  - decide:
      switch:
        - case1:
            when: ${ $data.count > 0 }
            then: continue
        - default:
            then: continue
`

	wf, err := zigflow.LoadFromBytes([]byte(yaml))
	require.NoError(t, err, "deterministic workflow must load cleanly")
	require.NotNil(t, wf)
}

// TestValidateBytes_RejectsNonDeterministic ensures the determinism check
// is reachable via ValidateBytes too, not only LoadFromBytes. This is the
// MCP and CLI validate entry point.
func TestValidateBytes_RejectsNonDeterministic(t *testing.T) {
	yaml := workflowHeader + `do:
  - waitForCooldown:
      wait:
        seconds: ${ timestamp }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.Error(t, err)
	assert.ErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
}

// TestValidateBytes_AcceptsValidSetWorkflow ensures ValidateBytes does not
// over-reject; a workflow using Set for non-determinism must validate clean.
func TestValidateBytes_AcceptsValidSetWorkflow(t *testing.T) {
	yaml := workflowHeader + `do:
  - capture:
      set:
        id: ${ uuid }
        ts: ${ timestamp }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.NoError(t, err)
}

// TestValidateBytes_RejectsInvalidExpressionInsideSetBody proves that Set
// bodies are exempt from the determinism rule but NOT from syntax validation:
// an unparseable jq expression inside a set body must be rejected as an invalid
// runtime expression, not silently accepted to fail later at execution time.
func TestValidateBytes_RejectsInvalidExpressionInsideSetBody(t *testing.T) {
	yaml := workflowHeader + `do:
  - capture:
      set:
        id: ${ @@@ }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.Error(t, err)
	assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression,
		"invalid jq in a set body must wrap ErrInvalidRuntimeExpression")
	assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression,
		"a parse failure must not be reported as non-determinism")
}

// TestValidateBytes_RejectsCompileInvalidExpressionInsideSetBody proves that a
// parse-valid but compile-invalid expression (an unregistered symbol, which jq
// treats as a 0-arg function call) inside a Set body is rejected. Set exempts
// determinism, not compile validation, so this must not slip through to fail at
// execution time.
func TestValidateBytes_RejectsCompileInvalidExpressionInsideSetBody(t *testing.T) {
	yaml := workflowHeader + `do:
  - capture:
      set:
        id: ${ definitely_not_registered }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.Error(t, err)
	assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression,
		"compile-invalid expression in a set body must wrap ErrInvalidRuntimeExpression")
	assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression,
		"a compile failure must not be reported as non-determinism")
}

// TestValidateBytes_RejectsInvalidExpressionOutsideSet proves the same parse
// validation applies to expressions outside a Set body (here a free-form HTTP
// query argument).
func TestValidateBytes_RejectsInvalidExpressionOutsideSet(t *testing.T) {
	yaml := workflowHeader + `do:
  - getUser:
      call: http
      with:
        method: get
        endpoint: https://example.com
        query:
          id: ${ @@@ }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.Error(t, err)
	assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression)
	assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
}

// TestValidateBytes_InvalidExpressionTakesPriorityOverNonDeterminism proves a
// parse failure is reported as an invalid runtime expression even when another
// expression is non-deterministic: parsing must succeed before determinism is
// judged.
func TestValidateBytes_InvalidExpressionTakesPriorityOverNonDeterminism(t *testing.T) {
	yaml := workflowHeader + `do:
  - waitForCooldown:
      wait:
        seconds: ${ timestamp }
  - getUser:
      call: http
      with:
        method: get
        endpoint: https://example.com
        query:
          id: ${ @@@ }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.Error(t, err)
	assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression)
	assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
}

// scheduleWorkflow builds a workflow whose document.metadata.scheduleInput
// carries the given expression. scheduleInput is interpolated at
// schedule-registration time, so it must be parse/compile-validated.
func scheduleWorkflow(expr string) string {
	return `document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: schedule
  version: 0.0.1
  metadata:
    scheduleWorkflowName: schedule
    scheduleInput:
      - envvars: ` + expr + `
schedule:
  every:
    minutes: 3
do:
  - wait:
      wait:
        seconds: 5
`
}

// TestValidateBytes_RejectsInvalidExpressionInScheduleInput proves that
// document.metadata.scheduleInput, which is interpolated at registration time,
// is parse/compile-validated rather than only failing later at `zigflow run`.
func TestValidateBytes_RejectsInvalidExpressionInScheduleInput(t *testing.T) {
	t.Run("parse invalid", func(t *testing.T) {
		err := zigflow.ValidateBytes([]byte(scheduleWorkflow("${ @@@ }")))
		require.Error(t, err)
		assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression)
		assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
	})

	t.Run("compile invalid", func(t *testing.T) {
		err := zigflow.ValidateBytes([]byte(scheduleWorkflow("${ definitely_not_registered }")))
		require.Error(t, err)
		assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression)
		assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
	})
}

// TestValidateBytes_AllowsNonDeterministicExpressionInScheduleInput proves that
// scheduleInput is exempt from the determinism rule (it runs at registration,
// outside the workflow), so a valid non-deterministic expression is accepted.
func TestValidateBytes_AllowsNonDeterministicExpressionInScheduleInput(t *testing.T) {
	err := zigflow.ValidateBytes([]byte(scheduleWorkflow("${ now }")))
	require.NoError(t, err, "scheduleInput must allow non-determinism")
}

// TestCollectExpressions_CollectsScheduleInput proves the scheduleInput
// expression is collected at the expected path and tagged as allowing
// non-determinism.
func TestCollectExpressions_CollectsScheduleInput(t *testing.T) {
	wf, err := zigflow.LoadFromBytes([]byte(scheduleWorkflow("${ $env.EXAMPLE }")))
	require.NoError(t, err)

	refs, err := zigflow.CollectExpressions(wf)
	require.NoError(t, err)

	var found bool
	for _, ref := range refs {
		if ref.Value == "${ $env.EXAMPLE }" {
			found = true
			assert.Equal(t, "document.metadata.scheduleInput[0].envvars", ref.Path)
			assert.True(t, ref.AllowsNonDeterminism,
				"scheduleInput is evaluated at registration and must be determinism-exempt")
		}
	}
	assert.True(t, found, "scheduleInput expression must be collected")
}

// TestCollectExpressions_TagsSetBodyAsAllowingNonDeterminism asserts the
// per-location tag, since the rest of the code depends on it.
func TestCollectExpressions_TagsSetBodyAsAllowingNonDeterminism(t *testing.T) {
	yaml := workflowHeader + `do:
  - capture:
      if: ${ $data.flag }
      set:
        id: ${ uuid }
`
	wf, err := zigflow.LoadFromBytes([]byte(yaml))
	require.NoError(t, err)

	refs, err := zigflow.CollectExpressions(wf)
	require.NoError(t, err)

	var foundSet, foundIf bool
	for _, ref := range refs {
		switch ref.Value {
		case "${ uuid }":
			foundSet = true
			assert.True(t, ref.AllowsNonDeterminism,
				"set body expression must allow non-determinism at %s", ref.Path)
			assert.True(t, strings.Contains(ref.Path, ".set"),
				"set body expression path must mention set, got %s", ref.Path)
		case "${ $data.flag }":
			foundIf = true
			assert.False(t, ref.AllowsNonDeterminism,
				"taskbase if expression must not allow non-determinism at %s", ref.Path)
		}
	}
	assert.True(t, foundSet, "expected to find ${ uuid } in collected refs")
	assert.True(t, foundIf, "expected to find ${ $data.flag } in collected refs")
}

// TestValidateWorkflowDeterminism_ReportsEveryIssue confirms the validator
// collects every non-deterministic reference rather than stopping at the
// first. Users fixing a broken workflow should see the full list.
func TestValidateWorkflowDeterminism_ReportsEveryIssue(t *testing.T) {
	yaml := workflowHeader + `do:
  - first:
      wait:
        seconds: ${ timestamp }
  - second:
      if: ${ uuid }
      set:
        v: ${ $data.x }
`
	// Cannot load (will fail), so analyse via a hand-built doc through
	// LoadFromBytes is impossible. Instead use the public collector via a
	// minimum-fixture workflow: build a *model.Workflow* via the loader on a
	// deterministic skeleton, then call ValidateWorkflowDeterminism is not
	// possible without exposing internals. Instead just check the error
	// message references both offending fields.
	_, err := zigflow.LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "timestamp", "error must mention timestamp")
	assert.Contains(t, msg, "uuid", "error must mention uuid")
}

// TestCollectExpressions_StableOrderForMapBackedStructures proves that
// expressions collected from a map (here a Set body with several keys) are
// returned in a stable, sorted order across runs, despite Go randomising map
// iteration. Unstable order would make CLI/JSON output and tests flaky.
func TestCollectExpressions_StableOrderForMapBackedStructures(t *testing.T) {
	yaml := workflowHeader + `do:
  - capture:
      set:
        zebra: ${ $data.z }
        alpha: ${ $data.a }
        mango: ${ $data.m }
`
	wf, err := zigflow.LoadFromBytes([]byte(yaml))
	require.NoError(t, err)

	// Keys are sorted, so the set body expressions appear in alphabetical order.
	wantPaths := []string{
		"do[0].capture.set.alpha",
		"do[0].capture.set.mango",
		"do[0].capture.set.zebra",
	}

	pathsFor := func() []string {
		refs, err := zigflow.CollectExpressions(wf)
		require.NoError(t, err)
		var paths []string
		for _, ref := range refs {
			paths = append(paths, ref.Path)
		}
		return paths
	}

	first := pathsFor()
	assert.Equal(t, wantPaths, first, "set body expressions must be in sorted order")

	// Repeat to catch map-iteration randomisation.
	for i := 0; i < 50; i++ {
		assert.Equal(t, first, pathsFor(), "collected order must be stable across runs")
	}
}

// TestCollectExpressions_CollectsActivityWithPayload proves expressions inside
// activity with payloads (for example grpc arguments) are collected for validation.
func TestCollectExpressions_CollectsActivityWithPayload(t *testing.T) {
	yaml := workflowHeader + `do:
  - callGrpc:
      call: grpc
      with:
        method: Command1
        arguments:
          input: ${ $env.GRPC_INPUT }
`
	wf, err := zigflow.LoadFromBytes([]byte(yaml))
	require.NoError(t, err)

	refs, err := zigflow.CollectExpressions(wf)
	require.NoError(t, err)

	var found bool
	for _, ref := range refs {
		if ref.Value == "${ $env.GRPC_INPUT }" {
			found = true
			assert.Contains(t, ref.Path, "arguments")
			assert.False(t, ref.AllowsNonDeterminism)
		}
	}
	assert.True(t, found, "grpc with.arguments expression must be collected")
}

// TestValidateBytes_RejectsInvalidExpressionInActivityWith proves invalid
// expressions inside activity with payloads fail validation.
func TestValidateBytes_RejectsInvalidExpressionInActivityWith(t *testing.T) {
	yaml := workflowHeader + `do:
  - getUser:
      call: http
      with:
        method: get
        endpoint: https://example.com
        headers:
          X-Token: ${ @@@ }
`
	err := zigflow.ValidateBytes([]byte(yaml))
	require.Error(t, err)
	assert.ErrorIs(t, err, zigflow.ErrInvalidRuntimeExpression)
	assert.NotErrorIs(t, err, zigflow.ErrNonDeterministicExpression)
}

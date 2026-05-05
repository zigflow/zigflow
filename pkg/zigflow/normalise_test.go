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

// package zigflow (not zigflow_test) so that unexported normalisation
// helpers are accessible directly without any public-API surface changes.
package zigflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// String constants used across normalise_test.go to satisfy goconst requirements.
const (
	testKeyDocument     = "document"
	testKeyDo           = "do"
	testKeyDSL          = "dsl"
	testKeyRun          = "run"
	testKeySet          = "set"
	testKeyType         = "type"
	testKeyWorkflow     = "workflow"
	testKeyWorkflowType = "workflowType"
	testKeyTaskQueue    = "taskQueue"
	testKeyVersion      = "version"
	testDSLVersion      = "1.0.0"
	testWFName          = "wf"
	testChildWorkflow   = "child-workflow"
	testHello           = "hello"
	testParent          = "parent"
)

// cloneMap returns a deep copy of m so that mutations in the function under
// test do not affect the original test-case literal.
func cloneMap(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(m)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// docOf extracts the "document" sub-map from a top-level workflow map.
func docOf(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	raw, ok := m[testKeyDocument]
	require.True(t, ok, "expected a 'document' key")
	doc, ok := raw.(map[string]any)
	require.True(t, ok, "expected document to be a map")
	return doc
}

// workflowOf extracts run.workflow from a single task-body map.
func workflowOf(t *testing.T, task map[string]any) map[string]any {
	t.Helper()
	raw, ok := task[testKeyRun]
	require.True(t, ok, "expected a 'run' key in task")
	run, ok := raw.(map[string]any)
	require.True(t, ok, "expected run to be a map")
	raw, ok = run[testKeyWorkflow]
	require.True(t, ok, "expected a 'workflow' key in run")
	wf, ok := raw.(map[string]any)
	require.True(t, ok, "expected run.workflow to be a map")
	return wf
}

// --- normaliseTopLevelDocument ---

func TestNormaliseTopLevelDocument_WorkflowTypeRenamedToName(t *testing.T) {
	// workflowType must become name.
	input := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: "my-workflow",
			testKeyTaskQueue:    "my-queue",
			testKeyVersion:      testDSLVersion,
		},
	})

	require.NoError(t, normaliseTopLevelDocument(input))

	doc := docOf(t, input)
	assert.Equal(t, "my-workflow", doc["name"], "workflowType must be mapped to name")
	assert.Equal(t, "my-queue", doc["namespace"], "taskQueue must be mapped to namespace")
}

func TestNormaliseTopLevelDocument_ZigflowKeysRemoved(t *testing.T) {
	// After normalisation the original Zigflow field names must not remain.
	input := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testWFName,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
	})

	require.NoError(t, normaliseTopLevelDocument(input))

	doc := docOf(t, input)
	assert.NotContains(t, doc, testKeyWorkflowType, "workflowType must be removed after normalisation")
	assert.NotContains(t, doc, testKeyTaskQueue, "taskQueue must be removed after normalisation")
}

func TestNormaliseTopLevelDocument_UnrelatedFieldsPreserved(t *testing.T) {
	// Fields unrelated to the Zigflow rename must survive unchanged.
	input := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testWFName,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
			"summary":           "A workflow summary",
		},
	})

	require.NoError(t, normaliseTopLevelDocument(input))

	doc := docOf(t, input)
	assert.Equal(t, testDSLVersion, doc[testKeyDSL])
	assert.Equal(t, testDSLVersion, doc[testKeyVersion])
	assert.Equal(t, "A workflow summary", doc["summary"])
}

func TestNormaliseTopLevelDocument_OnlyWorkflowType(t *testing.T) {
	// If only workflowType is present, name is added but namespace is not
	// introduced from thin air.
	input := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testWFName,
			testKeyVersion:      testDSLVersion,
		},
	})

	require.NoError(t, normaliseTopLevelDocument(input))

	doc := docOf(t, input)
	assert.Equal(t, testWFName, doc["name"])
	assert.NotContains(t, doc, "namespace", "namespace must not be introduced when taskQueue is absent")
}

func TestNormaliseTopLevelDocument_OnlyTaskQueue(t *testing.T) {
	// If only taskQueue is present, namespace is added but name is not
	// introduced from thin air.
	input := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:       testDSLVersion,
			testKeyTaskQueue: "q",
			testKeyVersion:   testDSLVersion,
		},
	})

	require.NoError(t, normaliseTopLevelDocument(input))

	doc := docOf(t, input)
	assert.Equal(t, "q", doc["namespace"])
	assert.NotContains(t, doc, "name", "name must not be introduced when workflowType is absent")
}

func TestNormaliseTopLevelDocument_NoZigflowFields(t *testing.T) {
	// When neither workflowType nor taskQueue are present the document must
	// not gain name or namespace keys.
	input := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:     testDSLVersion,
			testKeyVersion: testDSLVersion,
		},
	})

	require.NoError(t, normaliseTopLevelDocument(input))

	doc := docOf(t, input)
	assert.NotContains(t, doc, "name")
	assert.NotContains(t, doc, "namespace")
}

func TestNormaliseTopLevelDocument_NoDocumentKey(t *testing.T) {
	// A map without a "document" key is a no-op and must not error.
	input := cloneMap(t, map[string]any{
		testKeyDo: []any{},
	})

	require.NoError(t, normaliseTopLevelDocument(input))
}

// --- normaliseRunTask ---

func TestNormaliseRunTask_TypeRenamedToName(t *testing.T) {
	// run.workflow.type must become run.workflow.name.
	task := cloneMap(t, map[string]any{
		testKeyRun: map[string]any{
			testKeyWorkflow: map[string]any{
				testKeyType: testChildWorkflow,
			},
		},
	})

	require.NoError(t, normaliseRunTask(task))

	wf := workflowOf(t, task)
	assert.Equal(t, testChildWorkflow, wf["name"], "type must be mapped to name")
}

func TestNormaliseRunTask_TypeKeyRemoved(t *testing.T) {
	// The original "type" key must not remain after normalisation.
	task := cloneMap(t, map[string]any{
		testKeyRun: map[string]any{
			testKeyWorkflow: map[string]any{
				testKeyType: testChildWorkflow,
				"input":     map[string]any{"foo": "bar"},
			},
		},
	})

	require.NoError(t, normaliseRunTask(task))

	wf := workflowOf(t, task)
	assert.NotContains(t, wf, testKeyType, "type must be removed after normalisation")
}

func TestNormaliseRunTask_OtherWorkflowFieldsPreserved(t *testing.T) {
	// Fields alongside "type" in run.workflow must survive unchanged.
	task := cloneMap(t, map[string]any{
		testKeyRun: map[string]any{
			testKeyWorkflow: map[string]any{
				testKeyType: testChildWorkflow,
				"input":     map[string]any{"key": "value"},
			},
		},
	})

	require.NoError(t, normaliseRunTask(task))

	wf := workflowOf(t, task)
	assert.Equal(t, map[string]any{"key": "value"}, wf["input"])
}

func TestNormaliseRunTask_NoWorkflowKey(t *testing.T) {
	// A run task without a workflow key (e.g. container run) must not be
	// modified and must not error.
	task := cloneMap(t, map[string]any{
		testKeyRun: map[string]any{
			"container": map[string]any{
				"image": "alpine",
			},
		},
	})

	require.NoError(t, normaliseRunTask(task))

	// run.container must survive unchanged.
	run := task[testKeyRun].(map[string]any)
	assert.Equal(t, map[string]any{"image": "alpine"}, run["container"])
	assert.NotContains(t, run, testKeyWorkflow)
}

func TestNormaliseRunTask_NoRunKey(t *testing.T) {
	// A task without a "run" key at all (e.g. a set task) must be left
	// completely unchanged and must not error.
	task := cloneMap(t, map[string]any{
		testKeySet: map[string]any{
			"greeting": testHello,
		},
	})

	require.NoError(t, normaliseRunTask(task))

	assert.Equal(t, map[string]any{"greeting": testHello}, task[testKeySet])
}

func TestNormaliseRunTask_NormalisationIsIdempotentAfterRename(t *testing.T) {
	// If the task already uses the SDK field name (name instead of type),
	// the existing name must not be overwritten by a second rename attempt.
	task := cloneMap(t, map[string]any{
		testKeyRun: map[string]any{
			testKeyWorkflow: map[string]any{
				"name": "already-normalised",
			},
		},
	})

	require.NoError(t, normaliseRunTask(task))

	wf := workflowOf(t, task)
	assert.Equal(t, "already-normalised", wf["name"],
		"an existing name key must not be overwritten")
}

// --- normaliseWorkflowDocument: recursive traversal ---

func TestNormaliseWorkflowDocument_TopLevelDoRunTask(t *testing.T) {
	// A run.workflow task nested directly in the top-level do list must be
	// normalised.
	doc := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testParent,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
		testKeyDo: []any{
			map[string]any{
				"callChild": map[string]any{
					testKeyRun: map[string]any{
						testKeyWorkflow: map[string]any{
							testKeyType: testChildWorkflow,
						},
					},
				},
			},
		},
	})

	require.NoError(t, normaliseWorkflowDocument(doc))

	// Top-level document mapping.
	d := docOf(t, doc)
	assert.Equal(t, testParent, d["name"])
	assert.Equal(t, "q", d["namespace"])

	// Nested run.workflow normalisation.
	tasks := doc[testKeyDo].([]any)
	callChild := tasks[0].(map[string]any)["callChild"].(map[string]any)
	wf := callChild[testKeyRun].(map[string]any)[testKeyWorkflow].(map[string]any)
	assert.Equal(t, testChildWorkflow, wf["name"])
	assert.NotContains(t, wf, testKeyType)
}

func TestNormaliseWorkflowDocument_RunTaskInsideTry(t *testing.T) {
	// A run.workflow task inside a try block's task list must be normalised.
	doc := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testParent,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
		testKeyDo: []any{
			map[string]any{
				"tryBlock": map[string]any{
					"try": []any{
						map[string]any{
							"callChild": map[string]any{
								testKeyRun: map[string]any{
									testKeyWorkflow: map[string]any{
										testKeyType: "child-in-try",
									},
								},
							},
						},
					},
					"catch": map[string]any{
						"errors": map[string]any{"with": map[string]any{}},
					},
				},
			},
		},
	})

	require.NoError(t, normaliseWorkflowDocument(doc))

	tasks := doc[testKeyDo].([]any)
	tryBlock := tasks[0].(map[string]any)["tryBlock"].(map[string]any)
	tryList := tryBlock["try"].([]any)
	wf := tryList[0].(map[string]any)["callChild"].(map[string]any)[testKeyRun].(map[string]any)[testKeyWorkflow].(map[string]any)
	assert.Equal(t, "child-in-try", wf["name"])
	assert.NotContains(t, wf, testKeyType)
}

func TestNormaliseWorkflowDocument_RunTaskInsideCatchDo(t *testing.T) {
	// A run.workflow task inside a catch.do list must be normalised.
	doc := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testParent,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
		testKeyDo: []any{
			map[string]any{
				"tryBlock": map[string]any{
					"try": []any{
						map[string]any{
							"noop": map[string]any{
								testKeySet: map[string]any{"x": "1"},
							},
						},
					},
					"catch": map[string]any{
						"errors": map[string]any{"with": map[string]any{}},
						testKeyDo: []any{
							map[string]any{
								"recover": map[string]any{
									testKeyRun: map[string]any{
										testKeyWorkflow: map[string]any{
											testKeyType: "recovery-workflow",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	require.NoError(t, normaliseWorkflowDocument(doc))

	tasks := doc[testKeyDo].([]any)
	tryBlock := tasks[0].(map[string]any)["tryBlock"].(map[string]any)
	catchDo := tryBlock["catch"].(map[string]any)[testKeyDo].([]any)
	wf := catchDo[0].(map[string]any)["recover"].(map[string]any)[testKeyRun].(map[string]any)[testKeyWorkflow].(map[string]any)
	assert.Equal(t, "recovery-workflow", wf["name"])
	assert.NotContains(t, wf, testKeyType)
}

func TestNormaliseWorkflowDocument_RunTaskInsideForkBranch(t *testing.T) {
	// A run.workflow task inside a fork branch must be normalised.
	doc := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testParent,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
		testKeyDo: []any{
			map[string]any{
				"fanOut": map[string]any{
					"fork": map[string]any{
						"compete": false,
						"branches": []any{
							map[string]any{
								"branch1": map[string]any{
									testKeyRun: map[string]any{
										testKeyWorkflow: map[string]any{
											testKeyType: "child-a",
										},
									},
								},
							},
							map[string]any{
								"branch2": map[string]any{
									testKeyRun: map[string]any{
										testKeyWorkflow: map[string]any{
											testKeyType: "child-b",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	require.NoError(t, normaliseWorkflowDocument(doc))

	tasks := doc[testKeyDo].([]any)
	fanOut := tasks[0].(map[string]any)["fanOut"].(map[string]any)
	branches := fanOut["fork"].(map[string]any)["branches"].([]any)

	wf1 := branches[0].(map[string]any)["branch1"].(map[string]any)[testKeyRun].(map[string]any)[testKeyWorkflow].(map[string]any)
	assert.Equal(t, "child-a", wf1["name"])
	assert.NotContains(t, wf1, testKeyType)

	wf2 := branches[1].(map[string]any)["branch2"].(map[string]any)[testKeyRun].(map[string]any)[testKeyWorkflow].(map[string]any)
	assert.Equal(t, "child-b", wf2["name"])
	assert.NotContains(t, wf2, testKeyType)
}

// --- no-op behaviour ---

func TestNormaliseWorkflowDocument_SetTaskUntouched(t *testing.T) {
	// A set task must pass through normalisation without any modification.
	original := map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testWFName,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
		testKeyDo: []any{
			map[string]any{
				"greet": map[string]any{
					testKeySet: map[string]any{
						testHello: "world",
					},
				},
			},
		},
	}
	doc := cloneMap(t, original)

	require.NoError(t, normaliseWorkflowDocument(doc))

	tasks := doc[testKeyDo].([]any)
	greet := tasks[0].(map[string]any)["greet"].(map[string]any)
	assert.Equal(t, map[string]any{testHello: "world"}, greet[testKeySet],
		"set task body must be unchanged after normalisation")
}

func TestNormaliseWorkflowDocument_NoDo(t *testing.T) {
	// A document without a do key must have only the top-level field mapping
	// applied and must not error.
	doc := cloneMap(t, map[string]any{
		testKeyDocument: map[string]any{
			testKeyDSL:          testDSLVersion,
			testKeyWorkflowType: testWFName,
			testKeyTaskQueue:    "q",
			testKeyVersion:      testDSLVersion,
		},
	})

	require.NoError(t, normaliseWorkflowDocument(doc))

	d := docOf(t, doc)
	assert.Equal(t, testWFName, d["name"])
	assert.Equal(t, "q", d["namespace"])
	assert.NotContains(t, doc, testKeyDo)
}

// --- normaliser boundary: defaults are not the normaliser's responsibility ---
//
// namespace and version defaults ("default" / "0.0.1") on run.workflow are
// applied by RunTaskBuilder.PostLoad in pkg/zigflow/tasks/task_builder_run.go,
// not by the normaliser. The normaliser only renames type->name and does not
// inject defaults. The tests above intentionally do not assert on those fields.

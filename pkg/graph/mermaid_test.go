/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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

package graph_test

import (
	"testing"

	"github.com/mrsimonemms/zigflow/pkg/graph"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeWF is a helper that constructs a minimal Workflow for testing.
func makeWF(name string, tasks ...*model.TaskItem) *model.Workflow {
	tl := model.TaskList(make([]*model.TaskItem, len(tasks)))
	copy(tl, tasks)
	return &model.Workflow{
		Document: model.Document{
			DSL:       "1.0.0",
			Namespace: "test",
			Name:      name,
			Version:   "0.0.1",
		},
		Do: &tl,
	}
}

func taskItem(key string, task model.Task) *model.TaskItem {
	return &model.TaskItem{Key: key, Task: task}
}

func setTask(fields map[string]interface{}) *model.SetTask {
	return &model.SetTask{Set: fields}
}

func waitTaskSeconds(seconds int32) *model.WaitTask {
	return &model.WaitTask{Wait: &model.Duration{Value: model.DurationInline{Seconds: seconds}}}
}

func TestNew_UnknownFormat(t *testing.T) {
	_, err := graph.New("unknown")
	assert.Error(t, err)
}

func TestNew_MermaidFormat(t *testing.T) {
	gen, err := graph.New(graph.FormatMermaid)
	require.NoError(t, err)
	assert.NotNil(t, gen)
}

func TestMermaid_FlowchartHeader(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	wf := makeWF("test", taskItem("greet", setTask(map[string]interface{}{"msg": "hi"})))
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "flowchart TD")
}

func TestMermaid_SingleWorkflow_StartEnd(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	wf := makeWF("mywf",
		taskItem("step1", setTask(map[string]interface{}{"key": "val"})),
		taskItem("step2", waitTaskSeconds(5)),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// Flat flow: start and end nodes present
	assert.Contains(t, out, `([Start])`)
	assert.Contains(t, out, `([End])`)
	// Task nodes present
	assert.Contains(t, out, "step1")
	assert.Contains(t, out, "step2")
}

func TestMermaid_WaitLabel(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	wf := makeWF("mywf", taskItem("pause", waitTaskSeconds(10)))
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "WAIT (pause)")
}

func TestMermaid_SetLabel(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	wf := makeWF("mywf",
		taskItem("init", setTask(map[string]interface{}{"aaa": 1, "bbb": 2})),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "SET (init)")
}

func TestMermaid_HTTPLabel(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	ep := model.NewEndpoint("https://example.com/api")
	wf := makeWF("mywf",
		taskItem("fetch", &model.CallHTTP{
			Call: "http",
			With: model.HTTPArguments{Method: "get", Endpoint: ep},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "CALL_HTTP (fetch)")
}

func TestMermaid_SwitchNode(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	then := &model.FlowDirective{Value: "end"}
	wf := makeWF("mywf",
		taskItem("router", &model.SwitchTask{
			Switch: []model.SwitchItem{
				{
					"default": model.SwitchCase{Then: then},
				},
			},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// Switch rendered as diamond shape with TASK_TYPE label
	assert.Contains(t, out, `{"SWITCH (router)"}`)
}

func TestMermaid_SwitchTerminationEdge(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	endDir := &model.FlowDirective{Value: "end"}
	wf := makeWF("mywf",
		taskItem("check", &model.SwitchTask{
			Switch: []model.SwitchItem{
				{
					"done": model.SwitchCase{
						When: &model.RuntimeExpression{Value: "${ true }"},
						Then: endDir,
					},
				},
			},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// Edge label from switch to end
	assert.Contains(t, out, `${ true }`)
}

func TestMermaid_ForNode(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	inner := model.TaskList([]*model.TaskItem{
		taskItem("body", setTask(map[string]interface{}{"i": 0})),
	})
	wf := makeWF("mywf",
		taskItem("loop", &model.ForTask{
			For: model.ForTaskConfiguration{
				Each: "item",
				In:   "${ $input.list }",
			},
			Do: &inner,
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// For node rendered as subroutine shape
	assert.Contains(t, out, `[["`)
	assert.Contains(t, out, "FOR (loop)")
	assert.Contains(t, out, "loop (loop body)")
	// Loop body task present
	assert.Contains(t, out, "body")
	// Back-edge label
	assert.Contains(t, out, "next iteration")
}

func TestMermaid_ForkNode(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	b1Tasks := model.TaskList([]*model.TaskItem{
		taskItem("step", waitTaskSeconds(1)),
	})
	b2Tasks := model.TaskList([]*model.TaskItem{
		taskItem("step", waitTaskSeconds(2)),
	})
	branches := model.TaskList([]*model.TaskItem{
		taskItem("branch1", &model.DoTask{Do: &b1Tasks}),
		taskItem("branch2", &model.DoTask{Do: &b2Tasks}),
	})
	wf := makeWF("mywf",
		taskItem("fan", &model.ForkTask{
			Fork: model.ForkTaskConfiguration{
				Branches: &branches,
				Compete:  false,
			},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "FORK (fan)")
	assert.Contains(t, out, "branch1")
	assert.Contains(t, out, "branch2")
}

func TestMermaid_ForkCompete(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	branches := model.TaskList([]*model.TaskItem{
		taskItem("a", &model.DoTask{Do: func() *model.TaskList {
			tl := model.TaskList([]*model.TaskItem{taskItem("t", waitTaskSeconds(1))})
			return &tl
		}()}),
	})
	wf := makeWF("mywf",
		taskItem("race", &model.ForkTask{
			Fork: model.ForkTaskConfiguration{Branches: &branches, Compete: true},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "FORK (race) 🏁")
}

func TestMermaid_TryNode(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	tryTasks := model.TaskList([]*model.TaskItem{
		taskItem("risky", &model.CallHTTP{
			Call: "http",
			With: model.HTTPArguments{
				Method:   "get",
				Endpoint: model.NewEndpoint("https://example.com"),
			},
		}),
	})
	catchTasks := model.TaskList([]*model.TaskItem{
		taskItem("recover", setTask(map[string]interface{}{"err": "caught"})),
	})
	wf := makeWF("mywf",
		taskItem("safe", &model.TryTask{
			Try:   &tryTasks,
			Catch: &model.TryTaskCatch{Do: &catchTasks},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "TRY (safe)")
	assert.Contains(t, out, "CATCH (safe)")
	assert.Contains(t, out, `on error`)
	// Dashed edge
	assert.Contains(t, out, `-.->`)
	assert.Contains(t, out, "risky")
	assert.Contains(t, out, "recover")
}

func TestMermaid_MultipleWorkflows(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)

	wf1Tasks := model.TaskList([]*model.TaskItem{
		taskItem("ping", waitTaskSeconds(1)),
	})
	wf2Tasks := model.TaskList([]*model.TaskItem{
		taskItem("pong", waitTaskSeconds(2)),
	})
	// All top-level tasks are DoTasks → multiple independent workflows.
	wf := makeWF("root",
		taskItem("wf1", &model.DoTask{Do: &wf1Tasks}),
		taskItem("wf2", &model.DoTask{Do: &wf2Tasks}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// Both workflows shown as separate subgraphs
	assert.Contains(t, out, `subgraph wf_wf1`)
	assert.Contains(t, out, `subgraph wf_wf2`)
	assert.Contains(t, out, "ping")
	assert.Contains(t, out, "pong")
}

func TestMermaid_SubWorkflowCrossEdge(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)

	// Sub-workflow tasks (DoTask after non-DoTask)
	subTasks := model.TaskList([]*model.TaskItem{
		taskItem("action", waitTaskSeconds(1)),
	})

	switchThen := &model.FlowDirective{Value: "handler"}
	// Mixed: switch (non-DoTask) followed by handler (DoTask) = sub-workflow
	wf := makeWF("main",
		taskItem("router", &model.SwitchTask{
			Switch: []model.SwitchItem{
				{"go": model.SwitchCase{
					When: &model.RuntimeExpression{Value: "${ true }"},
					Then: switchThen,
				}},
			},
		}),
		taskItem("handler", &model.DoTask{Do: &subTasks}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// Main flow subgraph present
	assert.Contains(t, out, `subgraph wf_main`)
	// Sub-workflow subgraph present
	assert.Contains(t, out, `subgraph wf_handler`)
	// Cross-edge: switch → handler sub-workflow start
	assert.Contains(t, out, "wf_handler__start")
}

func TestMermaid_EmptyWorkflow(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	tl := model.TaskList([]*model.TaskItem{})
	wf := &model.Workflow{
		Document: model.Document{Name: "empty", DSL: "1.0.0", Namespace: "test", Version: "0.0.1"},
		Do:       &tl,
	}
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// Should at minimum produce the flowchart header
	assert.Contains(t, out, "flowchart TD")
}

func TestMermaid_ConditionalTask(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	ifExpr := &model.RuntimeExpression{Value: "${ $input.run }"}
	wf := makeWF("mywf",
		taskItem("maybe", &model.SetTask{
			TaskBase: model.TaskBase{If: ifExpr},
			Set:      map[string]interface{}{"x": 1},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	assert.Contains(t, out, "[?]")
}

func TestMermaid_LabelTruncation(t *testing.T) {
	gen, _ := graph.New(graph.FormatMermaid)
	longURL := "https://very.long.example.com/api/v1/users/profile/settings/preferences/items"
	ep := model.NewEndpoint(longURL)
	wf := makeWF("mywf",
		taskItem("fetch", &model.CallHTTP{
			Call: "http",
			With: model.HTTPArguments{Method: "get", Endpoint: ep},
		}),
	)
	out, err := gen.Generate(wf)
	require.NoError(t, err)
	// URL is never included in the label regardless of length
	assert.NotContains(t, out, longURL)
	assert.Contains(t, out, "CALL_HTTP (fetch)")
}

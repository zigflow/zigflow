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

package tasks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"sigs.k8s.io/yaml"
)

// nonDeterministicWarning is the message logged when a non-deterministic
// expression is evaluated without a side-effect wrapper.
const nonDeterministicWarning = "Non-deterministic expression used outside of a set task"

// helloWorldSetUUID is the hello-world example workflow, which generates a
// UUID inside a set task.
const helloWorldSetUUID = `
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: hello-world
  version: 0.0.1
  title: Hello World
  summary: Hello world with Zigflow
  metadata:
    display: false
do:
  - set:
      output:
        as:
          data: ${ . }
      set:
        id: ${ uuid }
        message: Hello from Ziggy
`

// captureLogs redirects the global zerolog logger to a buffer for the
// duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	prev := log.Logger
	log.Logger = zerolog.New(buf)
	t.Cleanup(func() { log.Logger = prev })

	return buf
}

func loadHelloWorldSetUUID(t *testing.T) *model.Workflow {
	t.Helper()

	jsonBytes, err := yaml.YAMLToJSON([]byte(helloWorldSetUUID))
	require.NoError(t, err)

	var doc model.Workflow
	require.NoError(t, json.Unmarshal(jsonBytes, &doc))

	return &doc
}

func TestSetTaskNonDeterministicExpressionDoesNotWarn(t *testing.T) {
	doc := loadHelloWorldSetUUID(t)
	logs := captureLogs(t)

	builder, err := NewDoTaskBuilder(nil, &model.DoTask{Do: doc.Do}, "hello-world", doc, testEvents, nil,
		DoTaskOpts{DisableRegisterWorkflow: true})
	require.NoError(t, err)

	fn, err := builder.Build()
	require.NoError(t, err)

	state := utils.NewState()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (any, error) {
		return fn(ctx, nil, state)
	}, workflow.RegisterOptions{Name: "hello-world"})

	env.ExecuteWorkflow("hello-world")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))

	data, ok := result["data"].(map[string]any)
	require.True(t, ok, "output.as should wrap the set result under data, got %v", result)
	assert.Equal(t, "Hello from Ziggy", data[testConstMessage])
	id, _ := data["id"].(string)
	assert.NoError(t, uuid.Validate(id), "id should be a UUID")
	assert.Equal(t, id, state.Data["id"])

	assert.NotContains(t, logs.String(), nonDeterministicWarning)
}

func TestHTTPFieldNonDeterministicExpressionWarns(t *testing.T) {
	logs := captureLogs(t)

	task := &model.CallHTTP{
		With: model.HTTPArguments{
			Method:   http.MethodGet,
			Endpoint: model.NewEndpoint("https://example.com"),
			Headers: model.NewObjectOrRuntimeExpr(map[string]any{
				"X-Request-ID": "${ uuid }",
			}),
		},
	}

	_, err := activities.ParseHTTPArguments(task, utils.NewState())
	require.NoError(t, err)

	assert.Contains(t, logs.String(), nonDeterministicWarning)
	assert.Contains(t, logs.String(), `"symbol":"uuid"`)
}

// TestSetTaskNonDeterministicExpressionReplaysRecordedUUID captures the
// SideEffect marker from a first execution of the hello-world set task and
// replays it through the real workflow replayer. uuid returns a new value on
// every evaluation, so an identical id proves it came from the marker.
func TestSetTaskNonDeterministicExpressionReplaysRecordedUUID(t *testing.T) {
	doc := loadHelloWorldSetUUID(t)
	task := (*doc.Do)[0].AsSetTask()
	require.NotNil(t, task)

	logs := captureLogs(t)

	marker, err := captureSetMarker(t, task.Set, utils.NewState())
	require.NoError(t, err)

	var recorded struct {
		Result map[string]any
	}
	require.NoError(t, json.Unmarshal(marker.GetPayloads()[0].GetData(), &recorded))
	recordedID, _ := recorded.Result["id"].(string)
	require.NoError(t, uuid.Validate(recordedID), "the marker should record the generated UUID")

	state := utils.NewState()
	output, taskErr, replayErr := replaySetTask(t, task.Set, state, setTaskHistory(t, false, marker))
	require.NoError(t, replayErr)
	require.NoError(t, taskErr)

	assert.Equal(t, recordedID, output.(map[string]any)["id"])
	assert.Equal(t, recordedID, state.Data["id"])
	assert.NotContains(t, logs.String(), nonDeterministicWarning)
}

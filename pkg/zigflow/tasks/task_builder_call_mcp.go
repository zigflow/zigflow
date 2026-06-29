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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/worker"
)

func endpointDecodeHook(from, to reflect.Type, data any) (any, error) {
	if to != reflect.TypeFor[model.Endpoint]() {
		return data, nil
	}

	// Re-encode the raw map back to JSON, then let Endpoint.UnmarshalJSON do its thing
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("endpointDecodeHook: marshal failed: %w", err)
	}

	var endpoint model.Endpoint
	if err := json.Unmarshal(jsonBytes, &endpoint); err != nil {
		return nil, fmt.Errorf("endpointDecodeHook: unmarshal failed: %w", err)
	}

	return endpoint, nil
}

func NewCallMCPTaskBuilder(
	temporalWorker worker.Worker,
	task *model.CallFunction,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (*CallMCPTaskBuilder, error) {
	// Convert the Function's "With" to MCPArguments
	var with models.MCPArguments
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			endpointDecodeHook,
			mapstructure.StringToTimeDurationHookFunc(), // keep any hooks you already rely on
		),
		Result: &with,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating endpoint decoder: %w", err)
	}
	if err := decoder.Decode(task.With); err != nil {
		return nil, fmt.Errorf("error converting mcp arguments: %w", err)
	}

	t := &models.CallMCP{
		TaskBase: *task.GetBase(),
		Call:     task.Call,
		With:     with,
	}

	return &CallMCPTaskBuilder{
		builder: builder[*models.CallMCP]{
			doc:            doc,
			eventEmitter:   emitter,
			name:           taskName,
			task:           t,
			taskOpts:       taskOpts,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type CallMCPTaskBuilder struct {
	builder[*models.CallMCP]
}

// Singleton whose bound method value is registered under per-task names.
// A bound value (not the unbound method expression) is required so
// Temporal invokes the activity with the receiver supplied.
var callMCPActivity = &activities.CallMCP{}

func (t *CallMCPTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return t.buildActivityFunc(callMCPActivity.CallMCPActivity, legacyCallMCPActivityName), nil
}

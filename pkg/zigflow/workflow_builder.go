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

package zigflow

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"github.com/zigflow/zigflow/pkg/zigflow/tasks"
	"go.temporal.io/sdk/worker"
)

func NewWorkflow(
	temporalWorker worker.Worker,
	doc *model.Workflow,
	envvars map[string]any,
	emitter *cloudevents.Events,
	telem *telemetry.Telemetry,
) error {
	workflowType := doc.Document.Name
	l := log.With().Str("workflowType", workflowType).Logger()

	maxHistoryLength, err := metadata.GetMaxHistoryLength(doc)
	if err != nil {
		return err
	}

	l.Debug().Msg("Creating new Do builder")
	doBuilder, err := tasks.NewDoTaskBuilder(
		temporalWorker,
		&model.DoTask{Do: doc.Do},
		workflowType,
		doc,
		emitter,
		tasks.DoTaskOpts{
			// Pass the envvars - this will be passed to the state object
			Envvars: envvars,
			// Search for a max history length for CAN override
			MaxHistoryLength: maxHistoryLength,
			// Add in telemetry
			Telemetry: telem,
		},
	)
	if err != nil {
		l.Error().Err(err).Msg("Error creating Do builder")
		return fmt.Errorf("error creating do builder: %w", err)
	}

	l.Debug().Msg("Building workflow")
	if _, err := doBuilder.Build(); err != nil {
		l.Debug().Err(err).Msg("Error building workflow")
		return fmt.Errorf("error building workflow: %w", err)
	}

	return nil
}

func newWorkflowPostLoad(doc *model.Workflow) error {
	workflowType := doc.Document.Name
	l := log.With().Str("workflowType", workflowType).Logger()

	doBuilder, err := tasks.NewDoTaskBuilder(
		nil,
		&model.DoTask{Do: doc.Do},
		workflowType,
		doc,
		&cloudevents.Events{}, // Stubbed as not used by the post loader
	)
	if err != nil {
		l.Error().Err(err).Msg("Error creating Do prep builder")
		return fmt.Errorf("error creating do prep builder: %w", err)
	}

	if err := doBuilder.PostLoad(); err != nil {
		l.Error().Err(err).Msg("Error post loading workflow")
		return fmt.Errorf("error post loading workflow: %w", err)
	}

	return nil
}

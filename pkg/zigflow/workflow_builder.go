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

package zigflow

import (
	"fmt"

	"github.com/mrsimonemms/zigflow/pkg/cloudevents"
	"github.com/mrsimonemms/zigflow/pkg/telemetry"
	"github.com/mrsimonemms/zigflow/pkg/zigflow/metadata"
	"github.com/mrsimonemms/zigflow/pkg/zigflow/tasks"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
)

func NewWorkflow(
	temporalWorker worker.Worker,
	doc *model.Workflow,
	envvars map[string]any,
	emitter *cloudevents.Events,
	telem *telemetry.Telemetry,
) error {
	workflowName := doc.Document.Name
	l := log.With().Str("workflowName", workflowName).Logger()

	maxHistoryLength, err := metadata.GetMaxHistoryLength(doc)
	if err != nil {
		return err
	}

	l.Debug().Msg("Creating new Do builder")
	doBuilder, err := tasks.NewDoTaskBuilder(
		temporalWorker,
		&model.DoTask{Do: doc.Do},
		workflowName,
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

	for _, a := range tasks.ActivitiesList() {
		l.Debug().Msg("Registering activity")
		temporalWorker.RegisterActivity(a)
	}

	return nil
}

func newWorkflowPostLoad(doc *model.Workflow) error {
	workflowName := doc.Document.Name
	l := log.With().Str("workflowName", workflowName).Logger()

	doBuilder, err := tasks.NewDoTaskBuilder(
		nil,
		&model.DoTask{Do: doc.Do},
		workflowName,
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

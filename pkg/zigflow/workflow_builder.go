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
	"github.com/zigflow/zigflow/pkg/utils"
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
	taskOpts *tasks.TaskOpts,
) error {
	workflowType := doc.Document.Name
	l := log.With().Str("workflowType", workflowType).Logger()

	maxHistoryLength, err := metadata.GetMaxHistoryLength(doc)
	if err != nil {
		return err
	}

	if doc.Timeout != nil && doc.Timeout.Timeout != nil && doc.Timeout.Timeout.After != nil {
		log.Warn().
			Str("docs", "https://zigflow.dev/docs/dsl/intro#timeout").
			Dur("timeout", utils.ToDuration(doc.Timeout.Timeout.After)).
			Msg("The top-level timeout field is deprecated; use document.metadata.activityOptions.startToCloseTimeout instead")
	}

	l.Debug().Msg("Creating new Do builder")
	doBuilder, err := tasks.NewDoTaskBuilder(
		temporalWorker,
		&model.DoTask{Do: doc.Do},
		workflowType,
		doc,
		emitter,
		taskOpts,
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

	l.Debug().Msg("Post-loading workflow")
	if err := doBuilder.PostLoad(); err != nil {
		l.Error().Err(err).Msg("Error post-loading workflow")
		return fmt.Errorf("error post-loading workflow: %w", err)
	}

	l.Debug().Msg("Validating workflow")
	if err := doBuilder.Validate(); err != nil {
		l.Error().Err(err).Msg("Error validating workflow")
		return fmt.Errorf("error validating workflow: %w", err)
	}

	l.Debug().Msg("Validating workflow expression determinism")
	if err := ValidateWorkflowDeterminism(doc); err != nil {
		l.Error().Err(err).Msg("Error validating workflow expression determinism")
		return fmt.Errorf("error validating workflow expression determinism: %w", err)
	}

	l.Debug().Msg("Building workflow")
	if _, err := doBuilder.Build(); err != nil {
		l.Debug().Err(err).Msg("Error building workflow")
		return fmt.Errorf("error building workflow: %w", err)
	}

	return nil
}

// newWorkflowPrepare runs the load-time PostLoad and Validate steps on
// a parsed workflow with a single DoTaskBuilder.
func newWorkflowPrepare(doc *model.Workflow) error {
	workflowType := doc.Document.Name
	l := log.With().Str("workflowType", workflowType).Logger()

	doBuilder, err := tasks.NewDoTaskBuilder(
		nil,
		&model.DoTask{Do: doc.Do},
		workflowType,
		doc,
		&cloudevents.Events{}, // stubbed: not used for prepare, only when the task needs to run
		&tasks.TaskOpts{},     // stubbed: not used for prepare, only when the task needs to run
	)
	if err != nil {
		l.Error().Err(err).Msg("Error creating prep builder")
		return fmt.Errorf("error creating do prep builder: %w", err)
	}

	if err := doBuilder.PostLoad(); err != nil {
		l.Error().Err(err).Msg("Error post loading workflow")
		return fmt.Errorf("error post loading workflow: %w", err)
	}

	if err := doBuilder.Validate(); err != nil {
		l.Error().Err(err).Msg("Error validating workflow")
		return fmt.Errorf("error validating workflow: %w", err)
	}

	if err := ValidateWorkflowDeterminism(doc); err != nil {
		// Logged at debug, not error: the failure is returned to the caller,
		// which is responsible for surfacing it (the CLI renders a concise,
		// human-readable message). Logging at error here duplicated that
		// diagnostic at the default log level.
		l.Debug().Err(err).Msg("Workflow expression determinism validation failed")
		return fmt.Errorf("error validating workflow expression determinism: %w", err)
	}

	return nil
}

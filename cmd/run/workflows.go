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

package run

import (
	"path/filepath"

	"github.com/matthewmueller/glob"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow"
)

// workflowRegistration holds a loaded and validated workflow definition ready
// to be registered on a Temporal worker. TaskQueue is derived from
// document.taskQueue. WorkflowType is derived from document.workflowType,
// which is the Temporal type identifier used during worker registration.
type workflowRegistration struct {
	SourceFile   string
	Definition   *model.Workflow
	Events       *cloudevents.Events
	TaskQueue    string
	WorkflowType string
}

func runValidation(validator *utils.Validator, workflowDefinition any) error {
	log.Debug().Msg("Running validation")
	res, err := validator.ValidateStruct(workflowDefinition)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error creating validation stack",
		}
	}
	if res != nil {
		return gh.FatalError{
			Msg: "Validation failed",
			WithParams: func(l *zerolog.Event) *zerolog.Event {
				f := []struct {
					Key     string
					Message string
				}{}
				for _, r := range res {
					f = append(f, struct {
						Key     string
						Message string
					}{
						Key:     r.Key,
						Message: r.Message,
					})
				}
				return l.Interface("validationErrors", f)
			},
		}
	}
	log.Debug().Msg("Validation passed")
	return nil
}

// discoverWorkflowFiles collects workflow file paths from --file flags and from
// --dir/--glob directory scanning. Both sources may be used together. Each path
// is normalised to an absolute path before deduplication, so that relative and
// absolute references to the same file are treated as one. Returns an error if
// no files are found.
func discoverWorkflowFiles(opts *runOptions) ([]string, error) {
	seen := make(map[string]struct{})
	var files []string

	addFile := func(f string) error {
		abs, err := filepath.Abs(f)
		if err != nil {
			return gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", f)
				},
				Msg: "Error resolving workflow file path",
			}
		}
		if _, ok := seen[abs]; !ok {
			seen[abs] = struct{}{}
			files = append(files, abs)
		}
		return nil
	}

	for _, f := range opts.Files {
		if err := addFile(f); err != nil {
			return nil, err
		}
	}

	if opts.DirectoryPath != "" {
		globbed, err := glob.Glob(opts.DirectoryPath, opts.DirectoryGlob)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				Msg:   "Error compiling glob",
			}
		}
		for _, f := range globbed {
			if err := addFile(f); err != nil {
				return nil, err
			}
		}
	}

	if len(files) == 0 {
		return nil, gh.FatalError{
			Msg: "No workflow files found",
		}
	}

	return files, nil
}

// loadWorkflows parses, optionally validates, and loads the CloudEvents handler
// for each file. Returns one workflowRegistration per file.
func loadWorkflows(
	files []string,
	cloudEventsConfig string,
	validator *utils.Validator,
	validate bool,
) ([]*workflowRegistration, error) {
	registrations := make([]*workflowRegistration, 0, len(files))

	for _, file := range files {
		if validate {
			if err := zigflow.ValidateFile(file); err != nil {
				return nil, gh.FatalError{
					Cause: err,
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Str("file", file)
					},
					Msg: "Schema validation failed",
				}
			}
		}

		def, err := zigflow.LoadFromFile(file)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Unable to load workflow file",
			}
		}

		// Defensive check: workflowType and taskQueue are used as Temporal
		// registration keys and worker-grouping keys respectively. An empty
		// value would silently produce a broken worker or a duplicate-key
		// collision, so reject such definitions here regardless of schema
		// validation.
		if def.Document.Name == "" {
			return nil, gh.FatalError{
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Workflow document.workflowType must not be empty",
			}
		}
		if def.Document.Namespace == "" {
			return nil, gh.FatalError{
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Workflow document.taskQueue must not be empty",
			}
		}

		if validate {
			if err := runValidation(validator, def); err != nil {
				return nil, err
			}
		}

		log.Debug().
			Str("file", file).
			Str("cloudEventsConfig", cloudEventsConfig).
			Msg("Registering CloudEvents handler")

		events, err := cloudevents.Load(cloudEventsConfig, validator, def)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("file", file)
				},
				Msg: "Error creating CloudEvents handler",
			}
		}

		registrations = append(registrations, &workflowRegistration{
			SourceFile:   file,
			Definition:   def,
			Events:       events,
			TaskQueue:    def.Document.Namespace,
			WorkflowType: def.Document.Name,
		})
	}

	return registrations, nil
}

// validateWorkflowConflicts detects registrations that would conflict on the
// same Temporal worker. Temporal uses document.workflowType as the workflow
// type identifier (via RegisterWorkflowWithOptions), so two workflows with the
// same workflowType on the same taskQueue cannot coexist on a single worker.
func validateWorkflowConflicts(registrations []*workflowRegistration) error {
	// seen maps task queue -> workflow name -> source file
	seen := make(map[string]map[string]string)

	for _, reg := range registrations {
		if _, ok := seen[reg.TaskQueue]; !ok {
			seen[reg.TaskQueue] = make(map[string]string)
		}
		if existing, ok := seen[reg.TaskQueue][reg.WorkflowType]; ok {
			return gh.FatalError{
				Msg: "Duplicate workflow name on the same task queue",
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.
						Str("workflowType", reg.WorkflowType).
						Str("taskQueue", reg.TaskQueue).
						Str("file", reg.SourceFile).
						Str("conflictsWith", existing)
				},
			}
		}
		seen[reg.TaskQueue][reg.WorkflowType] = reg.SourceFile
	}

	return nil
}

// prepareRegistrations discovers, loads, and validates all workflow files.
// It encapsulates the pipeline from path resolution through conflict detection
// so that runRunCmd stays within a manageable cyclomatic complexity budget.
func prepareRegistrations(opts *runOptions) ([]*workflowRegistration, error) {
	files, err := discoverWorkflowFiles(opts)
	if err != nil {
		return nil, err
	}

	log.Debug().Int("count", len(files)).Msg("Discovered workflow files")

	validator, err := utils.NewValidator()
	if err != nil {
		return nil, gh.FatalError{Cause: err, Msg: "Error creating validator"}
	}

	registrations, err := loadWorkflows(files, opts.CloudEventsConfig, validator, opts.Validate)
	if err != nil {
		return nil, err
	}

	if err := validateWorkflowConflicts(registrations); err != nil {
		return nil, err
	}

	return registrations, nil
}

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
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

// Search for nested Do tasks
func hasNestedDoTask(tasks *model.TaskList) bool {
	for _, task := range *tasks {
		if do := task.AsDoTask(); do != nil {
			return true
		}
	}
	return false
}

func loadCatalogs(doc *model.Workflow) error {
	if doc.Use == nil || len(doc.Use.Catalogs) == 0 {
		log.Debug().Msg("No catalogs to load")
		return nil
	}

	client := &http.Client{Timeout: time.Second * 5}

	workflowNestedDo := hasNestedDoTask(doc.Do)

	var taskList model.TaskList
	if workflowNestedDo {
		taskList = append(taskList, *doc.Do...)
	} else {
		taskList = append(taskList, &model.TaskItem{
			Key: doc.Document.Name,
			Task: &model.DoTask{
				Do: doc.Do,
			},
		})
	}

	for name, catalog := range doc.Use.Catalogs {
		resp, err := client.Get(catalog.Endpoint.String())
		if err != nil {
			return fmt.Errorf("error downloading catalog: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("catalog download has error status: %s", resp.Status)
		}

		resBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("catalog download read body failed: %w", err)
		}

		// As these will be registered as workflows, we need to make sure that
		// the given name isn't already registered
		for _, t := range *doc.Do {
			if t.Key == name {
				return fmt.Errorf("task %s already registered", t.Key)
			}
		}

		// Write to a tmp file so we can load it as a normal workflow
		f, err := os.CreateTemp("", name)
		if err != nil {
			return fmt.Errorf("error creating temp file: %w", err)
		}
		defer func() { _ = os.Remove(f.Name()) }()

		if _, err := f.Write(resBody); err != nil {
			return fmt.Errorf("error writing to temp file: %w", err)
		}

		catalogWf, err := LoadFromFile(f.Name())
		if err != nil {
			return fmt.Errorf("error loading catalog as workflow: %w", err)
		}

		// @todo(sje): we probably want to validate this file too
		fmt.Printf("%+v\n", len(*catalogWf.Do))

		hasNestedDo := hasNestedDoTask(catalogWf.Do)

		if hasNestedDo {
			// Multiple Dos - register all as child workflow
			for _, s := range *catalogWf.Do {
				taskList = append(taskList, s)
			}
		} else {
			// Single Do - register as child workflow
			taskList = append(taskList, &model.TaskItem{
				Key: name,
				Task: &model.DoTask{
					Do: catalogWf.Do,
				},
			})
		}
	}

	// Delete all the old catalogs
	doc.Use.Catalogs = nil

	// Replace Do task
	doc.Do = &taskList

	return nil
}

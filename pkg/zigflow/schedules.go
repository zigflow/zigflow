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
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/client"
)

func UpdateSchedules(ctx context.Context, temporalClient client.Client, workflow *model.Workflow, envvars map[string]any) error {
	info, err := metadata.GetScheduleInfo(workflow, envvars)
	if err != nil {
		return fmt.Errorf("error getting schedule metadata: %w", err)
	}

	schedule := workflow.Schedule
	scheduleClient := temporalClient.ScheduleClient()

	log.Info().Str("scheduleID", info.ID).Msg("Deleting old schedules")
	if err := deleteOldSchedules(ctx, scheduleClient, info.ID); err != nil {
		return err
	}

	if schedule == nil {
		log.Debug().Msg("No schedules set")
		return nil
	} else if info.WorkflowName == "" {
		log.Error().Msg("Workflow name not set")
		return fmt.Errorf("workflow name not set for schedule")
	}

	// Build Temporal schedules
	scheduleSpec, err := buildTemporalScheduleSpec(*schedule)
	if err != nil {
		return fmt.Errorf("error converting schedule to temporal: %w", err)
	}

	// Convert the Serverless Workflow schedule to a Temporal schedule
	opts := client.ScheduleOptions{
		ID:   info.ID,
		Spec: *scheduleSpec,
		Action: &client.ScheduleWorkflowAction{
			Workflow:  info.WorkflowName,
			TaskQueue: workflow.Document.Namespace, // mapped from document.taskQueue
			Args:      info.Input,
		},
	}

	log.Info().Any("schedule", info).Msg("Creating schedule")
	if _, err := scheduleClient.Create(ctx, opts); err != nil {
		return fmt.Errorf("error creating schedule: %w", err)
	}

	return nil
}

// Converts the Serverless Workflow schedule to Temporal schedule spec
func buildTemporalScheduleSpec(schedule model.Schedule) (*client.ScheduleSpec, error) {
	calendars := make([]client.ScheduleCalendarSpec, 0)
	cronExpression := make([]string, 0)
	intervals := make([]client.ScheduleIntervalSpec, 0)

	if schedule.Cron != "" {
		cronExpression = append(cronExpression, schedule.Cron)
	}
	if schedule.Every != nil {
		if duration := schedule.Every; duration != nil {
			intervals = append(intervals, client.ScheduleIntervalSpec{
				Every: utils.ToDuration(duration),
			})
		}
	}
	if schedule.After != nil {
		return nil, fmt.Errorf("schedule.after not supported")
	}

	return &client.ScheduleSpec{
		Calendars:       calendars,
		CronExpressions: cronExpression,
		Intervals:       intervals,
	}, nil
}

func deleteOldSchedules(ctx context.Context, scheduleClient client.ScheduleClient, scheduleID string) error {
	// Always delete matching schedules
	schedules, err := scheduleClient.List(ctx, client.ScheduleListOptions{})
	if err != nil {
		return fmt.Errorf("error listing temporal schedules: %w", err)
	}

	for schedules.HasNext() {
		s, err := schedules.Next()
		if err != nil {
			return fmt.Errorf("unable to get schedule: %w", err)
		}

		// Find and destroy the schedule
		if s.ID == scheduleID {
			handler := scheduleClient.GetHandle(ctx, s.ID)

			if err := handler.Delete(ctx); err != nil {
				return fmt.Errorf("error deleting workflow schedule: %w", err)
			}
		}
	}

	return nil
}

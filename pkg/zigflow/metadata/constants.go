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

package metadata

import (
	"time"

	"go.temporal.io/sdk/temporal"
)

const MetadataActivityOptions string = "activityOptions"

const MetadataHeartbeat string = "heartbeat"

const MetadataSearchAttribute string = "searchAttributes"

const (
	MetadataScheduleID           string = "scheduleId"
	MetadataScheduleWorkflowName string = "scheduleWorkflowName"
	MetadataScheduleInput        string = "scheduleInput"
)

const MaxHistoryLengthAttribute string = "canMaxHistoryLength"

const defaultWorkflowTimeout = time.Minute * 5

var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    time.Minute,
	MaximumAttempts:    5,
}

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

const (
	customCallFunctionActivity = "activity"
	customCallMCPActivity      = "mcp"
)

const (
	constKeyInput             = "input"
	constKeyState             = "state"
	constDefaultItemVar       = "item"
	constDefaultNamespace     = "default"
	constScriptLanguagePython = "python"
)

// Path segments threaded through Try's try/catch bodies so a leaf-named
// task appearing in both does not collide.
const (
	tryBodyPathSegment   = "try"
	catchBodyPathSegment = "catch"
)

// activityNamingVersionChangeID is the Temporal workflow versioning marker
// that gates the switch from the legacy fixed per-transport activity type
// names to the per-task aliases introduced for metrics and observability.
//
// Executions started before this change have no version marker in their
// history, so workflow.GetVersion returns workflow.DefaultVersion and they
// keep scheduling the legacy name they originally recorded; new executions
// record the marker and schedule the per-task alias. This keeps open
// histories deterministic across the upgrade.
//
// This string is a durable contract written into workflow histories. It
// must never change once released.
const activityNamingVersionChangeID = "zigflow.per-task-activity-aliases"

// activityNamingVersion is the version recorded for
// activityNamingVersionChangeID on new executions.
const activityNamingVersion = 1

// Legacy fixed activity type names. These are the names Temporal derived
// from the activity method names before per-task aliases existed, and they
// remain registered on every worker via ActivitiesList for back-compat.
// Open workflow histories scheduled activities under these names, so a
// replaying execution must continue to schedule them. They must stay
// aligned with the corresponding activities.* method names and must never
// change.
const (
	legacyCallHTTPActivityName      = "CallHTTPActivity"
	legacyCallGRPCActivityName      = "CallGRPCActivity"
	legacyCallMCPActivityName       = "CallMCPActivity"
	legacyCallContainerActivityName = "CallContainerActivity"
	legacyCallScriptActivityName    = "CallScriptActivity"
	legacyCallShellActivityName     = "CallShellActivity"
)

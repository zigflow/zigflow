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

package mcp

// Task type identifiers, shared across the MCP tools so the strings are defined
// once. They mirror the schema $defs task keys (see supportedTaskTypes, which is
// the runtime source of truth) and are guarded against drift by tests.
const (
	taskCall   = "call"
	taskDo     = "do"
	taskFor    = "for"
	taskFork   = "fork"
	taskListen = "listen"
	taskRaise  = "raise"
	taskRun    = "run"
	taskSet    = "set"
	taskSwitch = "switch"
	taskTry    = "try"
	taskWait   = "wait"
)

// Call task sub-types.
const (
	subActivity = "activity"
	subGRPC     = "grpc"
	subHTTP     = "http"
)

// Listen task event sub-types.
const (
	subSignal = "signal"
	subQuery  = "query"
	subUpdate = "update"
)

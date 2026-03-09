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

package graph

import "github.com/serverlessworkflow/sdk-go/v3/model"

// Shape describes the visual shape of a task's entry node.
type Shape int

const (
	// ShapeRect is a standard process node: ["label"].
	ShapeRect Shape = iota
	// ShapeDiamond is a decision node: {"label"}.
	ShapeDiamond
	// ShapeSubroutine is a predefined-process (loop) node: [["label"]].
	ShapeSubroutine
)

// NodeInfo describes the display properties of a task's entry node.
// It is exported so that future renderers within this package can consume it
// directly without re-implementing task-type detection.
type NodeInfo struct {
	TypeName string // upper-snake-case label, e.g. "CALL_HTTP"
	Shape    Shape  // zero value is ShapeRect
}

// nodeInfoFrom returns the NodeInfo for a task item.
// It returns (NodeInfo{}, false) for DoTask and TryTask, which produce no
// single named entry node and are rendered structurally by each renderer.
func nodeInfoFrom(item *model.TaskItem) (NodeInfo, bool) {
	switch {
	case item.AsCallHTTPTask() != nil:
		return NodeInfo{TypeName: "CALL_HTTP"}, true
	case item.AsCallGRPCTask() != nil:
		return NodeInfo{TypeName: "CALL_GRPC"}, true
	case item.AsCallFunctionTask() != nil:
		return NodeInfo{TypeName: "CALL_ACTIVITY"}, true
	case item.AsWaitTask() != nil:
		return NodeInfo{TypeName: "WAIT"}, true
	case item.AsSetTask() != nil:
		return NodeInfo{TypeName: "SET"}, true
	case item.AsRaiseTask() != nil:
		return NodeInfo{TypeName: "RAISE"}, true
	case item.AsRunTask() != nil:
		return NodeInfo{TypeName: "RUN"}, true
	case item.AsListenTask() != nil:
		return NodeInfo{TypeName: "LISTEN"}, true
	case item.AsSwitchTask() != nil:
		return NodeInfo{TypeName: "SWITCH", Shape: ShapeDiamond}, true
	case item.AsForTask() != nil:
		return NodeInfo{TypeName: "FOR", Shape: ShapeSubroutine}, true
	case item.AsForkTask() != nil:
		return NodeInfo{TypeName: "FORK"}, true
	default:
		return NodeInfo{}, false
	}
}

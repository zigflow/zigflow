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
// Zigflow Visual Editor — Pure IR Mutation Helpers
//
// All functions are pure: they return new values and never mutate in place.
// No UI dependencies. Safe to use in tests, CLI, or exporters.
import type {
  CallActivityConfig,
  CallGRPCConfig,
  CallHTTPConfig,
  DocumentMetadata,
  FlowGraph,
  ForkBranch,
  ForkNode,
  GraphPath,
  ListenConfig,
  LoopNode,
  NamedWorkflow,
  Node,
  NodeType,
  RaiseConfig,
  RunContainerConfig,
  RunScriptConfig,
  RunShellConfig,
  RunWorkflowConfig,
  SetConfig,
  SwitchBranch,
  SwitchNode,
  TaskConfig,
  TaskNode,
  TryNode,
  WaitConfig,
  WorkflowFile,
} from './model';
import { TASK_REGISTRY } from './registry';

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

function newId(): string {
  return crypto.randomUUID();
}

export function emptyFlowGraph(): FlowGraph {
  return { nodes: {}, order: [] };
}

// ---------------------------------------------------------------------------
// WorkflowFile mutations
// ---------------------------------------------------------------------------

export function createWorkflowFile(document: DocumentMetadata): WorkflowFile {
  const id = newId();
  const workflow: NamedWorkflow = {
    id,
    name: document.name,
    root: emptyFlowGraph(),
  };
  return {
    document,
    workflows: { [id]: workflow },
    order: [id],
  };
}

export function addWorkflow(file: WorkflowFile, name: string): WorkflowFile {
  const id = newId();
  const workflow: NamedWorkflow = { id, name, root: emptyFlowGraph() };
  return {
    ...file,
    workflows: { ...file.workflows, [id]: workflow },
    order: [...file.order, id],
  };
}

export function removeWorkflow(file: WorkflowFile, id: string): WorkflowFile {
  if (file.order.length <= 1) {
    throw new Error('Cannot remove the last workflow from a file');
  }
  const remaining = Object.fromEntries(
    Object.entries(file.workflows).filter(([k]) => k !== id),
  ) as WorkflowFile['workflows'];
  return {
    ...file,
    workflows: remaining,
    order: file.order.filter((wId) => wId !== id),
  };
}

export function updateWorkflowName(
  file: WorkflowFile,
  id: string,
  name: string,
): WorkflowFile {
  const workflow = file.workflows[id];
  if (!workflow) throw new Error(`Workflow ${id} not found`);
  return {
    ...file,
    workflows: { ...file.workflows, [id]: { ...workflow, name } },
  };
}

export function setWorkflowRoot(
  file: WorkflowFile,
  id: string,
  root: FlowGraph,
): WorkflowFile {
  const workflow = file.workflows[id];
  if (!workflow) throw new Error(`Workflow ${id} not found`);
  return {
    ...file,
    workflows: { ...file.workflows, [id]: { ...workflow, root } },
  };
}

// ---------------------------------------------------------------------------
// FlowGraph mutations
// ---------------------------------------------------------------------------

export function addNode(
  graph: FlowGraph,
  node: Node,
  atIndex?: number,
): FlowGraph {
  const order =
    atIndex !== undefined
      ? [
          ...graph.order.slice(0, atIndex),
          node.id,
          ...graph.order.slice(atIndex),
        ]
      : [...graph.order, node.id];
  return { nodes: { ...graph.nodes, [node.id]: node }, order };
}

export function removeNode(graph: FlowGraph, nodeId: string): FlowGraph {
  const remaining = Object.fromEntries(
    Object.entries(graph.nodes).filter(([k]) => k !== nodeId),
  ) as FlowGraph['nodes'];
  return { nodes: remaining, order: graph.order.filter((id) => id !== nodeId) };
}

export function moveNode(
  graph: FlowGraph,
  nodeId: string,
  toIndex: number,
): FlowGraph {
  if (!graph.nodes[nodeId]) throw new Error(`Node ${nodeId} not found`);
  const filtered = graph.order.filter((id) => id !== nodeId);
  const clamped = Math.max(0, Math.min(toIndex, filtered.length));
  filtered.splice(clamped, 0, nodeId);
  return { ...graph, order: filtered };
}

export function replaceNode(graph: FlowGraph, node: Node): FlowGraph {
  if (!graph.nodes[node.id]) throw new Error(`Node ${node.id} not found`);
  return { ...graph, nodes: { ...graph.nodes, [node.id]: node } };
}

// ---------------------------------------------------------------------------
// Node factories
// ---------------------------------------------------------------------------

export function createTaskNode(name: string, config: TaskConfig): TaskNode {
  return {
    id: newId(),
    type: 'task',
    name,
    config,
    metadata: { __zigflow_id: newId() },
  };
}

export function createSetNode(
  name: string,
  assignments: Record<string, string>,
): TaskNode {
  const config: SetConfig = { kind: 'set', assignments };
  return createTaskNode(name, config);
}

export function createCallHTTPNode(
  name: string,
  partial: Omit<CallHTTPConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'call-http', ...partial });
}

export function createCallGRPCNode(
  name: string,
  partial: Omit<CallGRPCConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'call-grpc', ...partial });
}

export function createCallActivityNode(
  name: string,
  partial: Omit<CallActivityConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'call-activity', ...partial });
}

export function createRunContainerNode(
  name: string,
  partial: Omit<RunContainerConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'run-container', ...partial });
}

export function createRunScriptNode(
  name: string,
  partial: Omit<RunScriptConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'run-script', ...partial });
}

export function createRunShellNode(
  name: string,
  partial: Omit<RunShellConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'run-shell', ...partial });
}

export function createRunWorkflowNode(
  name: string,
  partial: Omit<RunWorkflowConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'run-workflow', ...partial });
}

export function createWaitNode(
  name: string,
  partial: Omit<WaitConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'wait', ...partial });
}

export function createRaiseNode(
  name: string,
  partial: Omit<RaiseConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'raise', ...partial });
}

export function createListenNode(
  name: string,
  partial: Omit<ListenConfig, 'kind'>,
): TaskNode {
  return createTaskNode(name, { kind: 'listen', ...partial });
}

// ---------------------------------------------------------------------------
// SwitchNode
// ---------------------------------------------------------------------------

export function createSwitchNode(name: string): SwitchNode {
  return {
    id: newId(),
    type: 'switch',
    name,
    branches: [],
    metadata: { __zigflow_id: newId() },
  };
}

export function addSwitchBranch(
  node: SwitchNode,
  label: string,
  condition?: string,
): SwitchNode {
  const branch: SwitchBranch = {
    id: newId(),
    label,
    condition,
    graph: emptyFlowGraph(),
    metadata: { __zigflow_id: newId() },
  };
  return { ...node, branches: [...node.branches, branch] };
}

export function removeSwitchBranch(
  node: SwitchNode,
  branchId: string,
): SwitchNode {
  return {
    ...node,
    branches: node.branches.filter((b) => b.id !== branchId),
  };
}

export function updateSwitchBranchGraph(
  node: SwitchNode,
  branchId: string,
  graph: FlowGraph,
): SwitchNode {
  return {
    ...node,
    branches: node.branches.map((b) =>
      b.id === branchId ? { ...b, graph } : b,
    ),
  };
}

export function renameSwitchBranch(
  node: SwitchNode,
  branchId: string,
  label: string,
): SwitchNode {
  return {
    ...node,
    branches: node.branches.map((b) =>
      b.id === branchId ? { ...b, label } : b,
    ),
  };
}

// ---------------------------------------------------------------------------
// ForkNode
// ---------------------------------------------------------------------------

export function createForkNode(name: string): ForkNode {
  return {
    id: newId(),
    type: 'fork',
    name,
    compete: false,
    branches: [],
    metadata: { __zigflow_id: newId() },
  };
}

export function addForkBranch(node: ForkNode, label: string): ForkNode {
  const branch: ForkBranch = {
    id: newId(),
    label,
    graph: emptyFlowGraph(),
    metadata: { __zigflow_id: newId() },
  };
  return { ...node, branches: [...node.branches, branch] };
}

export function removeForkBranch(node: ForkNode, branchId: string): ForkNode {
  return {
    ...node,
    branches: node.branches.filter((b) => b.id !== branchId),
  };
}

export function updateForkBranchGraph(
  node: ForkNode,
  branchId: string,
  graph: FlowGraph,
): ForkNode {
  return {
    ...node,
    branches: node.branches.map((b) =>
      b.id === branchId ? { ...b, graph } : b,
    ),
  };
}

export function renameForkBranch(
  node: ForkNode,
  branchId: string,
  label: string,
): ForkNode {
  return {
    ...node,
    branches: node.branches.map((b) =>
      b.id === branchId ? { ...b, label } : b,
    ),
  };
}

// ---------------------------------------------------------------------------
// TryNode
// ---------------------------------------------------------------------------

export function createTryNode(name: string): TryNode {
  return {
    id: newId(),
    type: 'try',
    name,
    tryGraph: emptyFlowGraph(),
    metadata: { __zigflow_id: newId() },
  };
}

export function updateTrySection(
  node: TryNode,
  section: 'tryGraph' | 'catchGraph',
  graph: FlowGraph,
): TryNode {
  return { ...node, [section]: graph };
}

// ---------------------------------------------------------------------------
// LoopNode
// ---------------------------------------------------------------------------

export function createLoopNode(name: string, inExpr: string): LoopNode {
  return {
    id: newId(),
    type: 'loop',
    name,
    in: inExpr,
    bodyGraph: emptyFlowGraph(),
    metadata: { __zigflow_id: newId() },
  };
}

export function updateLoopBodyGraph(
  node: LoopNode,
  graph: FlowGraph,
): LoopNode {
  return { ...node, bodyGraph: graph };
}

// ---------------------------------------------------------------------------
// High-level: insert a new node by type into a workflow root graph
// ---------------------------------------------------------------------------

export function insertNode(
  file: WorkflowFile,
  workflowId: string,
  nodeType: NodeType,
  atIndex?: number,
): WorkflowFile {
  const def = TASK_REGISTRY.find((d) => d.type === nodeType);
  if (!def) throw new Error(`Unknown node type: ${nodeType}`);
  const node = def.create();
  const workflow = file.workflows[workflowId];
  if (!workflow) throw new Error(`Workflow ${workflowId} not found`);
  const newRoot = addNode(workflow.root, node, atIndex);
  return setWorkflowRoot(file, workflowId, newRoot);
}

// ---------------------------------------------------------------------------
// Path-based navigation helpers
// ---------------------------------------------------------------------------

// Resolve the FlowGraph at the given GraphPath. Throws on invalid paths.
//
// Segment consumption rules per node type:
//   loop   → 1 segment  (nodeId → bodyGraph)
//   switch → 2 segments (nodeId + branchId → branch.graph)
//   fork   → 2 segments (nodeId + branchId → branch.graph)
//   try    → 2 segments (nodeId + 'tryGraph'|'catchGraph' → section)
export function getGraphAtPath(file: WorkflowFile, path: GraphPath): FlowGraph {
  const wf = file.workflows[path.workflowId];
  if (!wf) throw new Error(`Workflow ${path.workflowId} not found`);

  let graph = wf.root;
  let i = 0;

  while (i < path.segments.length) {
    const nodeId = path.segments[i];
    if (!nodeId) throw new Error(`Empty segment at index ${i}`);

    const node = graph.nodes[nodeId];
    if (!node) throw new Error(`Node ${nodeId} not found at segment ${i}`);

    if (node.type === 'task') {
      throw new Error(`Node ${nodeId} is a task and has no sub-graph`);
    }

    if (node.type === 'loop') {
      graph = node.bodyGraph;
      i += 1;
      continue;
    }

    // switch, fork, try — consume one additional segment for the sub-graph id
    i += 1;
    if (i >= path.segments.length) {
      throw new Error(
        `Expected sub-graph identifier after node ${nodeId} (type: ${node.type})`,
      );
    }
    const subId = path.segments[i]!;

    if (node.type === 'switch') {
      const branch = node.branches.find((b) => b.id === subId);
      if (!branch) {
        throw new Error(`Branch ${subId} not found in switch node ${nodeId}`);
      }
      graph = branch.graph;
    } else if (node.type === 'fork') {
      const branch = node.branches.find((b) => b.id === subId);
      if (!branch) {
        throw new Error(`Branch ${subId} not found in fork node ${nodeId}`);
      }
      graph = branch.graph;
    } else if (node.type === 'try') {
      if (subId === 'catchGraph') {
        graph = node.catchGraph ?? node.tryGraph;
      } else if (subId === 'tryGraph') {
        graph = node.tryGraph;
      } else {
        throw new Error(
          `Invalid try section "${subId}" for node ${nodeId}. Expected 'tryGraph' or 'catchGraph'`,
        );
      }
    }

    i += 1;
  }

  return graph;
}

// Apply a pure transform to the FlowGraph at the given path, rebuilding the
// object tree immutably from the target graph up to the WorkflowFile root.
//
// No object is mutated in place. Only the nodes on the path are rebuilt;
// all siblings are structurally shared.
export function updateGraphAtPath(
  file: WorkflowFile,
  path: GraphPath,
  transform: (graph: FlowGraph) => FlowGraph,
): WorkflowFile {
  const wf = file.workflows[path.workflowId];
  if (!wf) throw new Error(`Workflow ${path.workflowId} not found`);

  const newRoot = applyTransformAt(wf.root, path.segments, 0, transform);
  return setWorkflowRoot(file, path.workflowId, newRoot);
}

// Internal: recursively descend segments and apply transform at the leaf.
function applyTransformAt(
  graph: FlowGraph,
  segments: string[],
  i: number,
  transform: (g: FlowGraph) => FlowGraph,
): FlowGraph {
  // Base case — we have consumed all segments; apply the transform here.
  if (i >= segments.length) {
    return transform(graph);
  }

  const nodeId = segments[i]!;
  const node = graph.nodes[nodeId];
  if (!node) throw new Error(`Node ${nodeId} not found at segment ${i}`);

  if (node.type === 'loop') {
    const newBody = applyTransformAt(
      node.bodyGraph,
      segments,
      i + 1,
      transform,
    );
    return replaceNode(graph, { ...node, bodyGraph: newBody });
  }

  // switch, fork, try — next segment identifies which sub-graph to descend into
  const subIndex = i + 1;
  if (subIndex >= segments.length) {
    throw new Error(
      `Expected sub-graph identifier after node ${nodeId} (type: ${node.type})`,
    );
  }
  const subId = segments[subIndex]!;

  if (node.type === 'switch') {
    const newBranches = node.branches.map((b) =>
      b.id === subId
        ? {
            ...b,
            graph: applyTransformAt(b.graph, segments, subIndex + 1, transform),
          }
        : b,
    );
    return replaceNode(graph, { ...node, branches: newBranches });
  }

  if (node.type === 'fork') {
    const newBranches = node.branches.map((b) =>
      b.id === subId
        ? {
            ...b,
            graph: applyTransformAt(b.graph, segments, subIndex + 1, transform),
          }
        : b,
    );
    return replaceNode(graph, { ...node, branches: newBranches });
  }

  if (node.type === 'try') {
    if (subId === 'catchGraph') {
      const newCatch = applyTransformAt(
        node.catchGraph ?? emptyFlowGraph(),
        segments,
        subIndex + 1,
        transform,
      );
      return replaceNode(graph, { ...node, catchGraph: newCatch });
    } else {
      const newTry = applyTransformAt(
        node.tryGraph,
        segments,
        subIndex + 1,
        transform,
      );
      return replaceNode(graph, { ...node, tryGraph: newTry });
    }
  }

  throw new Error(
    `Node ${nodeId} (type: ${node.type}) cannot be navigated into`,
  );
}

// Insert a new node by type into the FlowGraph at the given path.
// Appends to the end of the graph's order unless index is provided.
export function insertNodeAtPath(
  file: WorkflowFile,
  path: GraphPath,
  nodeType: NodeType,
  index?: number,
): WorkflowFile {
  const def = TASK_REGISTRY.find((d) => d.type === nodeType);
  if (!def) throw new Error(`Unknown node type: ${nodeType}`);
  const node = def.create();
  return updateGraphAtPath(file, path, (g) => addNode(g, node, index));
}

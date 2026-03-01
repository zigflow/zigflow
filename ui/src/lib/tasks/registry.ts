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
// Zigflow Visual Editor — Task Registry
//
// Pure static data: no UI dependencies, no mutations.
// Each TaskDefinition carries a `create()` factory that returns a fresh Node.
// Factories use crypto.randomUUID() directly to avoid importing from actions.ts.
import type {
  FlowGraph,
  ForkNode,
  LoopNode,
  Node,
  NodeType,
  SwitchNode,
  TaskNode,
  TryNode,
} from './model';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type TaskCategory = 'task' | 'control';

export type TaskDefinition = {
  type: NodeType;
  label: string;
  category: TaskCategory;
  description?: string;
  create: () => Node;
};

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function id(): string {
  return crypto.randomUUID();
}

function emptyGraph(): FlowGraph {
  return { nodes: {}, order: [] };
}

function taskNode(name: string, type: NodeType): TaskNode {
  return {
    id: id(),
    type: 'task',
    name,
    config: defaultConfig(type),
    metadata: { __zigflow_id: id() },
  } as TaskNode;
}

function defaultConfig(type: NodeType): TaskNode['config'] {
  switch (type) {
    case 'set':
      return { kind: 'set', assignments: {} };
    case 'call-http':
      return { kind: 'call-http', method: 'get', endpoint: '' };
    case 'call-grpc':
      return {
        kind: 'call-grpc',
        protoEndpoint: '',
        serviceName: '',
        serviceHost: '',
        servicePort: 443,
        method: '',
      };
    case 'call-activity':
      return { kind: 'call-activity', name: '' };
    case 'run-container':
      return { kind: 'run-container', image: '' };
    case 'run-script':
      return { kind: 'run-script', language: 'js', code: '' };
    case 'run-shell':
      return { kind: 'run-shell', command: '' };
    case 'run-workflow':
      return {
        kind: 'run-workflow',
        name: '',
        namespace: 'default',
        version: '0.0.1',
      };
    case 'wait':
      return { kind: 'wait', duration: { seconds: 30 } };
    case 'raise':
      return { kind: 'raise', errorType: '', errorStatus: 500 };
    case 'listen':
      return { kind: 'listen', mode: 'one', events: [] };
    default:
      // unreachable for task kinds; control types handled below
      return { kind: 'set', assignments: {} };
  }
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

export const TASK_REGISTRY: readonly TaskDefinition[] = [
  // ── Task nodes ────────────────────────────────────────────────────────────
  {
    type: 'set',
    label: 'Set',
    category: 'task',
    description: 'Assign values to workflow variables',
    create: () => taskNode('set-variables', 'set'),
  },
  {
    type: 'call-http',
    label: 'Call HTTP',
    category: 'task',
    description: 'Make an HTTP request',
    create: () => taskNode('http-call', 'call-http'),
  },
  {
    type: 'call-grpc',
    label: 'Call gRPC',
    category: 'task',
    description: 'Invoke a gRPC service method',
    create: () => taskNode('grpc-call', 'call-grpc'),
  },
  {
    type: 'call-activity',
    label: 'Call Activity',
    category: 'task',
    description: 'Execute a Temporal activity',
    create: () => taskNode('run-activity', 'call-activity'),
  },
  {
    type: 'run-container',
    label: 'Run Container',
    category: 'task',
    description: 'Run a container image',
    create: () => taskNode('run-container', 'run-container'),
  },
  {
    type: 'run-script',
    label: 'Run Script',
    category: 'task',
    description: 'Execute an inline script',
    create: () => taskNode('run-script', 'run-script'),
  },
  {
    type: 'run-shell',
    label: 'Run Shell',
    category: 'task',
    description: 'Execute a shell command',
    create: () => taskNode('run-shell', 'run-shell'),
  },
  {
    type: 'run-workflow',
    label: 'Run Workflow',
    category: 'task',
    description: 'Invoke another Zigflow workflow',
    create: () => taskNode('run-workflow', 'run-workflow'),
  },
  {
    type: 'wait',
    label: 'Wait',
    category: 'task',
    description: 'Pause execution for a duration',
    create: () => taskNode('wait', 'wait'),
  },
  {
    type: 'raise',
    label: 'Raise',
    category: 'task',
    description: 'Raise an error',
    create: () => taskNode('raise-error', 'raise'),
  },
  {
    type: 'listen',
    label: 'Listen',
    category: 'task',
    description: 'Wait for one or more events',
    create: () => taskNode('listen-event', 'listen'),
  },

  // ── Control flow nodes ────────────────────────────────────────────────────
  {
    type: 'switch',
    label: 'Switch',
    category: 'control',
    description: 'Branch on a condition',
    create: (): SwitchNode => ({
      id: id(),
      type: 'switch',
      name: 'switch',
      branches: [],
      metadata: { __zigflow_id: id() },
    }),
  },
  {
    type: 'fork',
    label: 'Fork',
    category: 'control',
    description: 'Run branches in parallel',
    create: (): ForkNode => ({
      id: id(),
      type: 'fork',
      name: 'fork',
      compete: false,
      branches: [],
      metadata: { __zigflow_id: id() },
    }),
  },
  {
    type: 'try',
    label: 'Try / Catch',
    category: 'control',
    description: 'Execute with error handling',
    create: (): TryNode => ({
      id: id(),
      type: 'try',
      name: 'try-catch',
      tryGraph: emptyGraph(),
      metadata: { __zigflow_id: id() },
    }),
  },
  {
    type: 'loop',
    label: 'Loop',
    category: 'control',
    description: 'Iterate over a collection',
    create: (): LoopNode => ({
      id: id(),
      type: 'loop',
      name: 'loop',
      in: '$.',
      bodyGraph: emptyGraph(),
      metadata: { __zigflow_id: id() },
    }),
  },
];

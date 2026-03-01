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

// Zigflow Visual Editor — Canonical IR Types
//
// This file is the authoritative source of truth for the workflow intermediate
// representation (IR). It must remain free of UI dependencies and serialisable
// to JSON at all times.

// ---------------------------------------------------------------------------
// Document & file
// ---------------------------------------------------------------------------

export type DocumentMetadata = {
  dsl: string;
  namespace: string;
  name: string;
  version: string;
  title?: string;
  summary?: string;
  metadata?: Record<string, unknown>;
};

export type WorkflowFile = {
  document: DocumentMetadata;
  workflows: Record<string, NamedWorkflow>;
  order: string[]; // stable ordering of named workflows
};

export type NamedWorkflow = {
  id: string;
  name: string;
  root: FlowGraph;
};

// ---------------------------------------------------------------------------
// FlowGraph
// ---------------------------------------------------------------------------

export type FlowGraph = {
  nodes: Record<string, Node>;
  order: string[]; // execution sequence — edges are derived from this
};

// ---------------------------------------------------------------------------
// Node union
// ---------------------------------------------------------------------------

export type Node = TaskNode | SwitchNode | ForkNode | TryNode | LoopNode;

// ---------------------------------------------------------------------------
// TaskNode
// Represents leaf Zigflow tasks: set, call, run, wait, raise, listen
// ---------------------------------------------------------------------------

export type TaskNode = {
  id: string;
  type: 'task';
  name: string;
  config: TaskConfig;
  if?: string;
  metadata?: Record<string, unknown>;
  export?: string;
  output?: string;
};

// ---------------------------------------------------------------------------
// SwitchNode
// Branches are modelled as embedded FlowGraphs; the exporter hoists them.
// ---------------------------------------------------------------------------

export type SwitchNode = {
  id: string;
  type: 'switch';
  name: string;
  branches: SwitchBranch[];
  if?: string;
  metadata?: Record<string, unknown>;
};

export type SwitchBranch = {
  id: string;
  label: string;
  condition?: string; // undefined = default branch
  graph: FlowGraph;
  metadata?: Record<string, unknown>;
};

// ---------------------------------------------------------------------------
// ForkNode
// ---------------------------------------------------------------------------

export type ForkNode = {
  id: string;
  type: 'fork';
  name: string;
  compete: boolean;
  branches: ForkBranch[];
  if?: string;
  metadata?: Record<string, unknown>;
};

export type ForkBranch = {
  id: string;
  label: string;
  graph: FlowGraph;
  metadata?: Record<string, unknown>;
};

// ---------------------------------------------------------------------------
// TryNode
// ---------------------------------------------------------------------------

export type TryNode = {
  id: string;
  type: 'try';
  name: string;
  tryGraph: FlowGraph;
  catchGraph?: FlowGraph;
  if?: string;
  metadata?: Record<string, unknown>;
};

// ---------------------------------------------------------------------------
// LoopNode  (Zigflow `for` task)
// ---------------------------------------------------------------------------

export type LoopNode = {
  id: string;
  type: 'loop';
  name: string;
  each?: string; // variable name for the current item
  in: string; // expression: collection, array, or count
  at?: string; // variable name for the current index
  while?: string; // optional break condition
  bodyGraph: FlowGraph;
  if?: string;
  metadata?: Record<string, unknown>;
};

// ---------------------------------------------------------------------------
// Task configs
// Each config carries a `kind` discriminant for exhaustive switching.
// ---------------------------------------------------------------------------

export type SetConfig = {
  kind: 'set';
  assignments: Record<string, string>;
};

export type CallHTTPConfig = {
  kind: 'call-http';
  method: 'get' | 'post' | 'put' | 'patch' | 'delete';
  endpoint: string;
  headers?: Record<string, string>;
  body?: string;
};

export type CallGRPCConfig = {
  kind: 'call-grpc';
  protoEndpoint: string;
  serviceName: string;
  serviceHost: string;
  servicePort: number;
  method: string;
  arguments?: Record<string, string>;
};

export type CallActivityConfig = {
  kind: 'call-activity';
  name: string;
  arguments?: string[];
  taskQueue?: string;
};

export type RunContainerConfig = {
  kind: 'run-container';
  image: string;
  arguments?: string[];
  environment?: Record<string, string>;
};

export type RunScriptConfig = {
  kind: 'run-script';
  language: string;
  code: string;
  arguments?: string[];
  environment?: Record<string, string>;
};

export type RunShellConfig = {
  kind: 'run-shell';
  command: string;
  arguments?: string[];
  environment?: Record<string, string>;
};

export type RunWorkflowConfig = {
  kind: 'run-workflow';
  name: string;
  namespace: string;
  version: string;
};

export type WaitConfig = {
  kind: 'wait';
  duration: DurationSpec;
};

export type DurationSpec = {
  seconds?: number;
  minutes?: number;
  hours?: number;
  days?: number;
};

export type RaiseConfig = {
  kind: 'raise';
  errorType: string;
  errorStatus: number;
  errorDetail?: string;
};

export type ListenEvent = {
  id: string;
  type: 'signal' | 'query' | 'update';
  acceptIf?: string;
  data?: Record<string, string>;
  datacontenttype?: string;
};

export type ListenConfig = {
  kind: 'listen';
  mode: 'one' | 'all';
  events: ListenEvent[];
};

export type TaskConfig =
  | SetConfig
  | CallHTTPConfig
  | CallGRPCConfig
  | CallActivityConfig
  | RunContainerConfig
  | RunScriptConfig
  | RunShellConfig
  | RunWorkflowConfig
  | WaitConfig
  | RaiseConfig
  | ListenConfig;

export type NodeType = TaskConfig['kind'] | 'switch' | 'fork' | 'try' | 'loop';

// ---------------------------------------------------------------------------
// GraphPath — stable, ID-based navigation into nested FlowGraphs
//
// Encoding rules:
//   segments = []                    → root graph of the named workflow
//   segments = [nodeId]              → bodyGraph of a LoopNode
//   segments = [nodeId, branchId]    → branch graph of SwitchNode or ForkNode
//   segments = [nodeId, 'tryGraph']  → tryGraph of a TryNode
//   segments = [nodeId, 'catchGraph']→ catchGraph of a TryNode
//   Segments recurse for deeper nesting.
//
// Never store FlowGraph references directly. Always derive from WorkflowFile
// plus GraphPath so state remains stable across immutable updates.
// ---------------------------------------------------------------------------

export type GraphPath = {
  workflowId: string;
  segments: string[];
};

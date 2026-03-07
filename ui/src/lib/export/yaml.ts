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
// Zigflow Visual Editor — IR → Zigflow DSL YAML Exporter
//
// Pure function: accepts a validated WorkflowFile, returns a YAML string.
// Never reads UI state. Never mutates the IR.
//
// Switch branch handling:
//   Each SwitchBranch carries an embedded FlowGraph. The exporter "hoists"
//   those graphs as named `do` tasks at the top level of the document, then
//   generates `then: <hoisted-name>` references in the switch entries.
import type {
  CallActivityConfig,
  CallGRPCConfig,
  CallHTTPConfig,
  FlowGraph,
  ForkNode,
  ListenConfig,
  LoopNode,
  Node,
  RaiseConfig,
  RunContainerConfig,
  RunScriptConfig,
  RunShellConfig,
  RunWorkflowConfig,
  SetConfig,
  SwitchNode,
  TaskConfig,
  TaskNode,
  TryNode,
  WaitConfig,
  WorkflowFile,
} from '$lib/tasks/model';
import { validateWorkflowFile } from '$lib/tasks/validation';
import yaml from 'js-yaml';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ExportResult =
  | { ok: true; yaml: string }
  | { ok: false; errors: string[] };

// Internal: hoisted named `do` tasks produced by switch branch extraction.
type HoistedTask = {
  name: string;
  tasks: unknown[];
};

type ExportContext = {
  hoisted: HoistedTask[];
};

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

export function exportToYaml(file: WorkflowFile): ExportResult {
  const result = validateWorkflowFile(file);
  if (!result.valid) {
    return {
      ok: false,
      errors: result.errors.map((e) => `[${e.path.join('.')}] ${e.message}`),
    };
  }

  const ctx: ExportContext = { hoisted: [] };
  const topLevelTasks = exportWorkflows(file, ctx);

  // Append hoisted switch-branch workflows after all primary tasks.
  for (const h of ctx.hoisted) {
    topLevelTasks.push({ [h.name]: { do: h.tasks } });
  }

  const doc: Record<string, unknown> = {
    document: exportDocument(file),
    do: topLevelTasks,
  };

  return {
    ok: true,
    yaml: yaml.dump(doc, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false,
    }),
  };
}

// ---------------------------------------------------------------------------
// Document metadata
// ---------------------------------------------------------------------------

function exportDocument(file: WorkflowFile): Record<string, unknown> {
  const d = file.document;
  const out: Record<string, unknown> = {
    dsl: d.dsl,
    namespace: d.namespace,
    name: d.name,
    version: d.version,
  };
  if (d.title !== undefined) out.title = d.title;
  if (d.summary !== undefined) out.summary = d.summary;
  if (d.metadata !== undefined) out.metadata = d.metadata;
  return out;
}

// ---------------------------------------------------------------------------
// Workflow sequencing
// ---------------------------------------------------------------------------

function exportWorkflows(file: WorkflowFile, ctx: ExportContext): unknown[] {
  // Single workflow: emit its tasks directly at the top level.
  if (file.order.length === 1) {
    const wf = file.workflows[file.order[0]];
    if (!wf) return [];
    return exportFlowGraphTasks(wf.root, ctx);
  }

  // Multiple workflows: each becomes a named `do` task.
  return file.order.map((id) => {
    const wf = file.workflows[id];
    if (!wf) return {};
    const tasks = exportFlowGraphTasks(wf.root, ctx);
    return { [wf.name]: { do: tasks } };
  });
}

function exportFlowGraphTasks(graph: FlowGraph, ctx: ExportContext): unknown[] {
  return graph.order.map((id) => {
    const node = graph.nodes[id];
    if (!node) return {};
    return exportNode(node, ctx);
  });
}

// ---------------------------------------------------------------------------
// Node dispatch
// ---------------------------------------------------------------------------

function exportNode(node: Node, ctx: ExportContext): unknown {
  switch (node.type) {
    case 'task':
      return exportTaskNode(node);
    case 'switch':
      return exportSwitchNode(node, ctx);
    case 'fork':
      return exportForkNode(node, ctx);
    case 'try':
      return exportTryNode(node, ctx);
    case 'loop':
      return exportLoopNode(node, ctx);
  }
}

// ---------------------------------------------------------------------------
// TaskNode
// ---------------------------------------------------------------------------

function exportTaskNode(node: TaskNode): unknown {
  const def: Record<string, unknown> = exportTaskConfig(node.config);
  if (node.if !== undefined) def.if = node.if;
  if (node.metadata !== undefined) def.metadata = node.metadata;
  if (node.export !== undefined) def.export = { as: node.export };
  if (node.output !== undefined) def.output = { as: node.output };
  return { [node.name]: def };
}

function exportTaskConfig(config: TaskConfig): Record<string, unknown> {
  switch (config.kind) {
    case 'set':
      return exportSetConfig(config);
    case 'call-http':
      return exportCallHTTPConfig(config);
    case 'call-grpc':
      return exportCallGRPCConfig(config);
    case 'call-activity':
      return exportCallActivityConfig(config);
    case 'run-container':
      return exportRunContainerConfig(config);
    case 'run-script':
      return exportRunScriptConfig(config);
    case 'run-shell':
      return exportRunShellConfig(config);
    case 'run-workflow':
      return exportRunWorkflowConfig(config);
    case 'wait':
      return exportWaitConfig(config);
    case 'raise':
      return exportRaiseConfig(config);
    case 'listen':
      return exportListenConfig(config);
  }
}

function exportSetConfig(c: SetConfig): Record<string, unknown> {
  return { set: c.assignments };
}

function exportCallHTTPConfig(c: CallHTTPConfig): Record<string, unknown> {
  const w: Record<string, unknown> = { method: c.method, endpoint: c.endpoint };
  if (c.headers !== undefined) w.headers = c.headers;
  if (c.body !== undefined) w.body = c.body;
  return { call: 'http', with: w };
}

function exportCallGRPCConfig(c: CallGRPCConfig): Record<string, unknown> {
  const w: Record<string, unknown> = {
    proto: { endpoint: c.protoEndpoint },
    service: { name: c.serviceName, host: c.serviceHost, port: c.servicePort },
    method: c.method,
  };
  if (c.arguments !== undefined) w.arguments = c.arguments;
  return { call: 'grpc', with: w };
}

function exportCallActivityConfig(
  c: CallActivityConfig,
): Record<string, unknown> {
  const w: Record<string, unknown> = { name: c.name };
  if (c.arguments !== undefined) w.arguments = c.arguments;
  if (c.taskQueue !== undefined) w.taskQueue = c.taskQueue;
  return { call: 'activity', with: w };
}

function exportRunContainerConfig(
  c: RunContainerConfig,
): Record<string, unknown> {
  const container: Record<string, unknown> = { image: c.image };
  if (c.arguments !== undefined) container.arguments = c.arguments;
  if (c.environment !== undefined) container.environment = c.environment;
  return { run: { container } };
}

function exportRunScriptConfig(c: RunScriptConfig): Record<string, unknown> {
  const script: Record<string, unknown> = {
    language: c.language,
    code: c.code,
  };
  if (c.arguments !== undefined) script.arguments = c.arguments;
  if (c.environment !== undefined) script.environment = c.environment;
  return { run: { script } };
}

function exportRunShellConfig(c: RunShellConfig): Record<string, unknown> {
  const shell: Record<string, unknown> = { command: c.command };
  if (c.arguments !== undefined) shell.arguments = c.arguments;
  if (c.environment !== undefined) shell.environment = c.environment;
  return { run: { shell } };
}

function exportRunWorkflowConfig(
  c: RunWorkflowConfig,
): Record<string, unknown> {
  return {
    run: {
      workflow: { name: c.name, namespace: c.namespace, version: c.version },
    },
  };
}

function exportWaitConfig(c: WaitConfig): Record<string, unknown> {
  return { wait: c.duration };
}

function exportRaiseConfig(c: RaiseConfig): Record<string, unknown> {
  const error: Record<string, unknown> = {
    type: c.errorType,
    status: c.errorStatus,
  };
  if (c.errorDetail !== undefined) error.detail = c.errorDetail;
  return { raise: { error } };
}

function exportListenConfig(c: ListenConfig): Record<string, unknown> {
  const events = c.events.map((e) => {
    const w: Record<string, unknown> = { id: e.id, type: e.type };
    if (e.acceptIf !== undefined) w.acceptIf = e.acceptIf;
    if (e.data !== undefined) w.data = e.data;
    if (e.datacontenttype !== undefined) w.datacontenttype = e.datacontenttype;
    return { with: w };
  });

  if (c.mode === 'one') {
    return { listen: { to: { one: events[0] } } };
  }
  return { listen: { to: { all: events } } };
}

// ---------------------------------------------------------------------------
// SwitchNode — branch hoisting
//
// Each branch's FlowGraph is exported as a named `do` task appended to the
// top-level sequence. The switch entry references it via `then:`.
// ---------------------------------------------------------------------------

function exportSwitchNode(node: SwitchNode, ctx: ExportContext): unknown {
  const branches = node.branches.map((branch) => {
    const hoistedName = `${node.name}-${branch.label}`;
    const tasks = exportFlowGraphTasks(branch.graph, ctx);
    ctx.hoisted.push({ name: hoistedName, tasks });

    const entry: Record<string, unknown> = { then: hoistedName };
    if (branch.condition !== undefined) entry.when = branch.condition;
    if (branch.metadata !== undefined) entry.metadata = branch.metadata;
    return { [branch.label]: entry };
  });

  const def: Record<string, unknown> = { switch: branches };
  if (node.if !== undefined) def.if = node.if;
  if (node.metadata !== undefined) def.metadata = node.metadata;
  return { [node.name]: def };
}

// ---------------------------------------------------------------------------
// ForkNode — branches are inline do sequences
// ---------------------------------------------------------------------------

function exportForkNode(node: ForkNode, ctx: ExportContext): unknown {
  const branches = node.branches.map((branch) => {
    const tasks = exportFlowGraphTasks(branch.graph, ctx);
    const branchDef: Record<string, unknown> = { do: tasks };
    if (branch.metadata !== undefined) branchDef.metadata = branch.metadata;
    return { [branch.label]: branchDef };
  });

  const def: Record<string, unknown> = {
    fork: { compete: node.compete, branches },
  };
  if (node.if !== undefined) def.if = node.if;
  if (node.metadata !== undefined) def.metadata = node.metadata;
  return { [node.name]: def };
}

// ---------------------------------------------------------------------------
// TryNode
// ---------------------------------------------------------------------------

function exportTryNode(node: TryNode, ctx: ExportContext): unknown {
  const tryTasks = exportFlowGraphTasks(node.tryGraph, ctx);
  const def: Record<string, unknown> = { try: tryTasks };

  if (node.catchGraph) {
    const catchTasks = exportFlowGraphTasks(node.catchGraph, ctx);
    def.catch = { do: catchTasks };
  }

  if (node.if !== undefined) def.if = node.if;
  if (node.metadata !== undefined) def.metadata = node.metadata;
  return { [node.name]: def };
}

// ---------------------------------------------------------------------------
// LoopNode  (Zigflow `for` task)
// ---------------------------------------------------------------------------

function exportLoopNode(node: LoopNode, ctx: ExportContext): unknown {
  const bodyTasks = exportFlowGraphTasks(node.bodyGraph, ctx);

  const forDef: Record<string, unknown> = { in: node.in };
  if (node.each !== undefined) forDef.each = node.each;
  if (node.at !== undefined) forDef.at = node.at;
  if (node.while !== undefined) forDef.while = node.while;

  const def: Record<string, unknown> = { for: forDef, do: bodyTasks };
  if (node.if !== undefined) def.if = node.if;
  if (node.metadata !== undefined) def.metadata = node.metadata;
  return { [node.name]: def };
}

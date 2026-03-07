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
// Zigflow Visual Editor — Zigflow DSL YAML → WorkflowFile IR Parser
//
// Parses a Zigflow DSL YAML document into the WorkflowFile intermediate
// representation. Ensures metadata.__zigflow_id is present on every node
// and branch; generates UUIDs for any that are missing.
//
// Switch branch graphs are stored as hoisted top-level `do` entries
// (referenced via `then:`) by the exporter. The parser resolves these
// references to reconstruct the nested FlowGraph structure.
import yaml from 'js-yaml';

import type {
  CallActivityConfig,
  CallGRPCConfig,
  CallHTTPConfig,
  DocumentMetadata,
  FlowGraph,
  ForkBranch,
  ForkNode,
  ListenConfig,
  ListenEvent,
  LoopNode,
  NamedWorkflow,
  Node,
  RaiseConfig,
  RunContainerConfig,
  RunScriptConfig,
  RunShellConfig,
  RunWorkflowConfig,
  SetConfig,
  SwitchBranch,
  SwitchNode,
  TaskNode,
  TryNode,
  WaitConfig,
  WorkflowFile,
} from './model';

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export type ParseResult = {
  workflowFile: WorkflowFile;
  // True when any __zigflow_id was generated and the file should be written back.
  modified: boolean;
};

// ---------------------------------------------------------------------------
// Internal types
// ---------------------------------------------------------------------------

type RawEntry = Record<string, unknown>;
type ParseCtx = { modified: boolean };

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

export function parseWorkflowFile(
  content: string,
  fileName: string,
): ParseResult {
  const raw = yaml.load(content) as RawEntry;
  if (!raw || typeof raw !== 'object') {
    throw new Error('Invalid YAML: expected a mapping at the document root');
  }

  const rawDoc = raw['document'] as RawEntry | undefined;
  if (!rawDoc) {
    throw new Error('Missing required "document" section');
  }

  const document: DocumentMetadata = {
    dsl: String(rawDoc['dsl'] ?? '1.0.0'),
    namespace: String(rawDoc['namespace'] ?? 'default'),
    name: String(rawDoc['name'] ?? fileName.replace(/\.ya?ml$/, '')),
    version: String(rawDoc['version'] ?? '0.0.1'),
  };
  if (rawDoc['title'] !== undefined) document.title = String(rawDoc['title']);
  if (rawDoc['summary'] !== undefined)
    document.summary = String(rawDoc['summary']);

  const doEntries = (raw['do'] as RawEntry[]) ?? [];

  const ctx: ParseCtx = { modified: false };

  // Build a flat map of all named entries for hoisted branch resolution.
  const entryMap = new Map<string, RawEntry>();
  for (const entry of doEntries) {
    const [name, def] = splitEntry(entry);
    entryMap.set(name, def);
  }

  // Identify names that are hoisted sub-graphs (referenced by switch `then:`).
  const hoistedNames = collectHoistedNames(doEntries);

  // Parse the root workflow graph from non-hoisted entries.
  const root = parseGraph(doEntries, entryMap, hoistedNames, ctx);

  const workflowId = crypto.randomUUID();
  const workflow: NamedWorkflow = {
    id: workflowId,
    name: document.name,
    root,
  };

  const workflowFile: WorkflowFile = {
    document,
    workflows: { [workflowId]: workflow },
    order: [workflowId],
  };

  return { workflowFile, modified: ctx.modified };
}

// ---------------------------------------------------------------------------
// Entry splitting
// ---------------------------------------------------------------------------

function splitEntry(entry: RawEntry): [string, RawEntry] {
  const keys = Object.keys(entry);
  if (keys.length !== 1) {
    throw new Error(
      `Expected exactly one key per do-entry, got: ${keys.join(', ')}`,
    );
  }
  const name = keys[0]!;
  return [name, (entry[name] ?? {}) as RawEntry];
}

// ---------------------------------------------------------------------------
// Hoisted name collection
//
// Switch branches reference their sub-graphs via `then: hoistedName`. We
// collect all such names so the graph parser can skip them when building the
// main sequence.
// ---------------------------------------------------------------------------

function collectHoistedNames(entries: RawEntry[]): Set<string> {
  const names = new Set<string>();
  for (const entry of entries) {
    const [, def] = splitEntry(entry);
    collectHoistedFromDef(def, names);
  }
  return names;
}

function collectHoistedFromDef(def: RawEntry, names: Set<string>): void {
  if ('switch' in def && Array.isArray(def['switch'])) {
    for (const branchEntry of def['switch'] as RawEntry[]) {
      const [, branchDef] = splitEntry(branchEntry);
      if (
        'then' in branchDef &&
        typeof (branchDef as RawEntry)['then'] === 'string'
      ) {
        names.add((branchDef as RawEntry)['then'] as string);
      }
    }
  }

  if ('fork' in def) {
    const fork = (def['fork'] ?? {}) as Record<string, unknown>;
    const branches = (fork['branches'] as RawEntry[]) ?? [];
    for (const branchEntry of branches) {
      const [, branchDef] = splitEntry(branchEntry);
      if ('do' in branchDef && Array.isArray((branchDef as RawEntry)['do'])) {
        const subEntries = (branchDef as RawEntry)['do'] as RawEntry[];
        for (const sub of subEntries) {
          const [, subDef] = splitEntry(sub);
          collectHoistedFromDef(subDef, names);
        }
      }
    }
  }

  if ('try' in def && Array.isArray(def['try'])) {
    for (const sub of def['try'] as RawEntry[]) {
      const [, subDef] = splitEntry(sub);
      collectHoistedFromDef(subDef, names);
    }
    if ('catch' in def) {
      const catchDef = (def['catch'] ?? {}) as Record<string, unknown>;
      const catchEntries = (catchDef['do'] as RawEntry[]) ?? [];
      for (const sub of catchEntries) {
        const [, subDef] = splitEntry(sub);
        collectHoistedFromDef(subDef, names);
      }
    }
  }

  if ('for' in def && 'do' in def && Array.isArray(def['do'])) {
    for (const sub of def['do'] as RawEntry[]) {
      const [, subDef] = splitEntry(sub);
      collectHoistedFromDef(subDef, names);
    }
  }
}

// ---------------------------------------------------------------------------
// Graph parsing
// ---------------------------------------------------------------------------

function parseGraph(
  entries: RawEntry[],
  entryMap: Map<string, RawEntry>,
  hoistedNames: Set<string>,
  ctx: ParseCtx,
): FlowGraph {
  const nodes: Record<string, Node> = {};
  const order: string[] = [];

  for (const entry of entries) {
    const [name, def] = splitEntry(entry);
    if (hoistedNames.has(name)) continue;
    const node = parseNode(name, def, entryMap, hoistedNames, ctx);
    nodes[node.id] = node;
    order.push(node.id);
  }

  return { nodes, order };
}

// ---------------------------------------------------------------------------
// ID resolution
// ---------------------------------------------------------------------------

function resolveId(
  def: RawEntry,
  ctx: ParseCtx,
): { id: string; metadata: Record<string, unknown> } {
  const metadata = (def['metadata'] ?? {}) as Record<string, unknown>;
  const existing = metadata['__zigflow_id'];
  if (typeof existing === 'string' && existing.length > 0) {
    return { id: existing, metadata };
  }
  const id = crypto.randomUUID();
  ctx.modified = true;
  return { id, metadata: { ...metadata, __zigflow_id: id } };
}

// ---------------------------------------------------------------------------
// Node dispatch
// ---------------------------------------------------------------------------

function parseNode(
  name: string,
  def: RawEntry,
  entryMap: Map<string, RawEntry>,
  hoistedNames: Set<string>,
  ctx: ParseCtx,
): Node {
  if ('switch' in def)
    return parseSwitchNode(name, def, entryMap, hoistedNames, ctx);
  if ('fork' in def)
    return parseForkNode(name, def, entryMap, hoistedNames, ctx);
  if ('try' in def) return parseTryNode(name, def, entryMap, hoistedNames, ctx);
  if ('for' in def)
    return parseLoopNode(name, def, entryMap, hoistedNames, ctx);
  return parseTaskNode(name, def, ctx);
}

// ---------------------------------------------------------------------------
// TaskNode
// ---------------------------------------------------------------------------

function parseTaskNode(name: string, def: RawEntry, ctx: ParseCtx): TaskNode {
  const { id, metadata } = resolveId(def, ctx);
  const config = parseTaskConfig(def);
  const node: TaskNode = { id, type: 'task', name, config, metadata };
  if (typeof def['if'] === 'string') node.if = def['if'];
  if (typeof def['export'] === 'string') node.export = def['export'];
  if (typeof def['output'] === 'string') node.output = def['output'];
  return node;
}

function parseTaskConfig(
  def: RawEntry,
):
  | ReturnType<typeof buildSetConfig>
  | ReturnType<typeof buildWaitConfig>
  | CallHTTPConfig
  | CallGRPCConfig
  | CallActivityConfig
  | RunContainerConfig
  | RunScriptConfig
  | RunShellConfig
  | RunWorkflowConfig
  | RaiseConfig
  | ListenConfig
  | SetConfig
  | WaitConfig {
  if ('set' in def) return buildSetConfig(def);
  if ('wait' in def) return buildWaitConfig(def);
  if ('call' in def) return buildCallConfig(def);
  if ('run' in def) return buildRunConfig(def);
  if ('raise' in def) return buildRaiseConfig(def);
  if ('listen' in def) return buildListenConfig(def);
  // Unknown task type — fall back to empty set to avoid hard failure.
  return { kind: 'set', assignments: {} };
}

function buildSetConfig(def: RawEntry): SetConfig {
  return {
    kind: 'set',
    assignments: (def['set'] ?? {}) as Record<string, string>,
  };
}

function buildWaitConfig(def: RawEntry): WaitConfig {
  return {
    kind: 'wait',
    duration: (def['wait'] ?? {}) as WaitConfig['duration'],
  };
}

function buildCallConfig(
  def: RawEntry,
): CallHTTPConfig | CallGRPCConfig | CallActivityConfig {
  const callType = String(def['call'] ?? '');
  const w = (def['with'] ?? {}) as Record<string, unknown>;

  if (callType === 'http') {
    const cfg: CallHTTPConfig = {
      kind: 'call-http',
      method: String(w['method'] ?? 'get') as CallHTTPConfig['method'],
      endpoint: String(w['endpoint'] ?? ''),
    };
    if (w['headers'] !== undefined)
      cfg.headers = w['headers'] as Record<string, string>;
    if (w['body'] !== undefined) cfg.body = String(w['body']);
    return cfg;
  }

  if (callType === 'grpc') {
    const proto = (w['proto'] ?? {}) as Record<string, unknown>;
    const service = (w['service'] ?? {}) as Record<string, unknown>;
    const cfg: CallGRPCConfig = {
      kind: 'call-grpc',
      protoEndpoint: String(proto['endpoint'] ?? ''),
      serviceName: String(service['name'] ?? ''),
      serviceHost: String(service['host'] ?? ''),
      servicePort: Number(service['port'] ?? 0),
      method: String(w['method'] ?? ''),
    };
    if (w['arguments'] !== undefined)
      cfg.arguments = w['arguments'] as Record<string, string>;
    return cfg;
  }

  if (callType === 'activity') {
    const cfg: CallActivityConfig = {
      kind: 'call-activity',
      name: String(w['name'] ?? ''),
    };
    if (w['arguments'] !== undefined)
      cfg.arguments = w['arguments'] as string[];
    if (w['taskQueue'] !== undefined) cfg.taskQueue = String(w['taskQueue']);
    return cfg;
  }

  throw new Error(`Unknown call type: ${callType}`);
}

function buildRunConfig(
  def: RawEntry,
): RunContainerConfig | RunScriptConfig | RunShellConfig | RunWorkflowConfig {
  const run = (def['run'] ?? {}) as Record<string, unknown>;

  if ('container' in run) {
    const container = run['container'] as Record<string, unknown>;
    const cfg: RunContainerConfig = {
      kind: 'run-container',
      image: String(container['image'] ?? ''),
    };
    if (container['arguments'] !== undefined)
      cfg.arguments = container['arguments'] as string[];
    if (container['environment'] !== undefined)
      cfg.environment = container['environment'] as Record<string, string>;
    return cfg;
  }

  if ('script' in run) {
    const script = run['script'] as Record<string, unknown>;
    const cfg: RunScriptConfig = {
      kind: 'run-script',
      language: String(script['language'] ?? ''),
      code: String(script['code'] ?? ''),
    };
    if (script['arguments'] !== undefined)
      cfg.arguments = script['arguments'] as string[];
    if (script['environment'] !== undefined)
      cfg.environment = script['environment'] as Record<string, string>;
    return cfg;
  }

  if ('shell' in run) {
    const shell = run['shell'] as Record<string, unknown>;
    const cfg: RunShellConfig = {
      kind: 'run-shell',
      command: String(shell['command'] ?? ''),
    };
    if (shell['arguments'] !== undefined)
      cfg.arguments = shell['arguments'] as string[];
    if (shell['environment'] !== undefined)
      cfg.environment = shell['environment'] as Record<string, string>;
    return cfg;
  }

  if ('workflow' in run) {
    const wf = run['workflow'] as Record<string, unknown>;
    return {
      kind: 'run-workflow',
      name: String(wf['name'] ?? ''),
      namespace: String(wf['namespace'] ?? ''),
      version: String(wf['version'] ?? ''),
    };
  }

  throw new Error('Unknown run type in do-entry');
}

function buildRaiseConfig(def: RawEntry): RaiseConfig {
  const raise = (def['raise'] ?? {}) as Record<string, unknown>;
  const error = (raise['error'] ?? {}) as Record<string, unknown>;
  const cfg: RaiseConfig = {
    kind: 'raise',
    errorType: String(error['type'] ?? ''),
    errorStatus: Number(error['status'] ?? 0),
  };
  if (error['detail'] !== undefined) cfg.errorDetail = String(error['detail']);
  return cfg;
}

function buildListenConfig(def: RawEntry): ListenConfig {
  const listen = (def['listen'] ?? {}) as Record<string, unknown>;
  const to = (listen['to'] ?? {}) as Record<string, unknown>;

  if ('one' in to) {
    const event = parseListenEvent(to['one']);
    return {
      kind: 'listen',
      mode: 'one',
      events: event ? [event] : [],
    };
  }

  if ('all' in to) {
    const rawEvents = (to['all'] as unknown[]) ?? [];
    return {
      kind: 'listen',
      mode: 'all',
      events: rawEvents.map(parseListenEvent).filter(Boolean) as ListenEvent[],
    };
  }

  return { kind: 'listen', mode: 'one', events: [] };
}

function parseListenEvent(raw: unknown): ListenEvent | null {
  if (!raw || typeof raw !== 'object') return null;
  const w = ((raw as Record<string, unknown>)['with'] ?? {}) as Record<
    string,
    unknown
  >;
  const event: ListenEvent = {
    id: String(w['id'] ?? ''),
    type: String(w['type'] ?? 'signal') as ListenEvent['type'],
  };
  if (w['acceptIf'] !== undefined) event.acceptIf = String(w['acceptIf']);
  if (w['data'] !== undefined) event.data = w['data'] as Record<string, string>;
  if (w['datacontenttype'] !== undefined)
    event.datacontenttype = String(w['datacontenttype']);
  return event;
}

// ---------------------------------------------------------------------------
// SwitchNode
// ---------------------------------------------------------------------------

function parseSwitchNode(
  name: string,
  def: RawEntry,
  entryMap: Map<string, RawEntry>,
  hoistedNames: Set<string>,
  ctx: ParseCtx,
): SwitchNode {
  const { id, metadata } = resolveId(def, ctx);
  const rawBranches = (def['switch'] as RawEntry[]) ?? [];

  const branches: SwitchBranch[] = rawBranches.map((branchEntry) => {
    const [label, branchDef] = splitEntry(branchEntry);
    const { id: branchId, metadata: branchMeta } = resolveId(branchDef, ctx);

    let graph: FlowGraph = { nodes: {}, order: [] };
    const thenName = (branchDef as RawEntry)['then'];
    if (typeof thenName === 'string') {
      const hoistedDef = entryMap.get(thenName);
      if (hoistedDef && 'do' in hoistedDef) {
        const subEntries = (hoistedDef['do'] as RawEntry[]) ?? [];
        const subHoisted = collectHoistedNames(subEntries);
        graph = parseGraph(subEntries, entryMap, subHoisted, ctx);
      }
    }

    const branch: SwitchBranch = {
      id: branchId,
      label,
      graph,
      metadata: branchMeta,
    };
    if (typeof (branchDef as RawEntry)['when'] === 'string') {
      branch.condition = (branchDef as RawEntry)['when'] as string;
    }
    return branch;
  });

  const node: SwitchNode = { id, type: 'switch', name, branches, metadata };
  if (typeof def['if'] === 'string') node.if = def['if'];
  return node;
}

// ---------------------------------------------------------------------------
// ForkNode
// ---------------------------------------------------------------------------

function parseForkNode(
  name: string,
  def: RawEntry,
  entryMap: Map<string, RawEntry>,
  hoistedNames: Set<string>,
  ctx: ParseCtx,
): ForkNode {
  const { id, metadata } = resolveId(def, ctx);
  const fork = (def['fork'] ?? {}) as Record<string, unknown>;
  const compete = Boolean(fork['compete'] ?? false);
  const rawBranches = (fork['branches'] as RawEntry[]) ?? [];

  const branches: ForkBranch[] = rawBranches.map((branchEntry) => {
    const [label, branchDef] = splitEntry(branchEntry);
    const { id: branchId, metadata: branchMeta } = resolveId(branchDef, ctx);
    const subEntries = ((branchDef as RawEntry)['do'] as RawEntry[]) ?? [];
    const subHoisted = collectHoistedNames(subEntries);
    const graph = parseGraph(subEntries, entryMap, subHoisted, ctx);
    return { id: branchId, label, graph, metadata: branchMeta };
  });

  const node: ForkNode = {
    id,
    type: 'fork',
    name,
    compete,
    branches,
    metadata,
  };
  if (typeof def['if'] === 'string') node.if = def['if'];
  return node;
}

// ---------------------------------------------------------------------------
// TryNode
// ---------------------------------------------------------------------------

function parseTryNode(
  name: string,
  def: RawEntry,
  entryMap: Map<string, RawEntry>,
  hoistedNames: Set<string>,
  ctx: ParseCtx,
): TryNode {
  const { id, metadata } = resolveId(def, ctx);
  const tryEntries = (def['try'] as RawEntry[]) ?? [];
  const tryHoisted = collectHoistedNames(tryEntries);
  const tryGraph = parseGraph(tryEntries, entryMap, tryHoisted, ctx);

  const node: TryNode = { id, type: 'try', name, tryGraph, metadata };

  if ('catch' in def) {
    const catchDef = (def['catch'] ?? {}) as Record<string, unknown>;
    const catchEntries = (catchDef['do'] as RawEntry[]) ?? [];
    const catchHoisted = collectHoistedNames(catchEntries);
    node.catchGraph = parseGraph(catchEntries, entryMap, catchHoisted, ctx);
  }

  if (typeof def['if'] === 'string') node.if = def['if'];
  return node;
}

// ---------------------------------------------------------------------------
// LoopNode
// ---------------------------------------------------------------------------

function parseLoopNode(
  name: string,
  def: RawEntry,
  entryMap: Map<string, RawEntry>,
  hoistedNames: Set<string>,
  ctx: ParseCtx,
): LoopNode {
  const { id, metadata } = resolveId(def, ctx);
  const forDef = (def['for'] ?? {}) as Record<string, unknown>;
  const bodyEntries = (def['do'] as RawEntry[]) ?? [];
  const bodyHoisted = collectHoistedNames(bodyEntries);
  const bodyGraph = parseGraph(bodyEntries, entryMap, bodyHoisted, ctx);

  const node: LoopNode = {
    id,
    type: 'loop',
    name,
    in: String(forDef['in'] ?? ''),
    bodyGraph,
    metadata,
  };
  if (typeof forDef['each'] === 'string') node.each = forDef['each'];
  if (typeof forDef['at'] === 'string') node.at = forDef['at'];
  if (typeof forDef['while'] === 'string') node.while = forDef['while'];
  if (typeof def['if'] === 'string') node.if = def['if'];
  return node;
}

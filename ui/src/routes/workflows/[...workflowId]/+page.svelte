<!--
  ~ Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
  ~
  ~ Licensed under the Apache License, Version 2.0 (the "License");
  ~ you may not use this file except in compliance with the License.
  ~ You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
  -->

<script lang="ts">
  import { browser } from '$app/environment';
  import { exportToYaml } from '$lib/export/yaml';
  import {
    addForkBranch,
    addNode,
    addSwitchBranch,
    addWorkflow,
    createForkNode,
    createLoopNode,
    createSetNode,
    createSwitchNode,
    createTryNode,
    createWaitNode,
    createWorkflowFile,
    emptyFlowGraph,
    getGraphAtPath,
    insertNodeAtPath,
    moveNode,
    removeForkBranch,
    removeNode,
    removeSwitchBranch,
    renameForkBranch,
    renameSwitchBranch,
    replaceNode,
    updateGraphAtPath,
    updateTrySection,
  } from '$lib/tasks/actions';
  import type {
    FlowGraph,
    GraphPath,
    Node,
    NodeType,
    WorkflowFile,
  } from '$lib/tasks/model';
  import Breadcrumb from '$lib/ui/Breadcrumb.svelte';
  import Canvas from '$lib/ui/Canvas.svelte';
  import ContextIndicator from '$lib/ui/ContextIndicator.svelte';
  import Inspector from '$lib/ui/Inspector.svelte';
  import Sidebar from '$lib/ui/Sidebar.svelte';
  import { onMount } from 'svelte';

  // ---------------------------------------------------------------------------
  // Workflow state (IR)
  // ---------------------------------------------------------------------------

  // Demo file: a sequential workflow with varied node types.
  function buildDemoFile(): WorkflowFile {
    const file = createWorkflowFile({
      dsl: '1.0.0',
      namespace: 'demo',
      name: 'my-workflow',
      version: '0.0.1',
      title: 'Demo Workflow',
    });

    const rootId = file.order[0]!;

    // Build a FlowGraph with representative nodes
    let root: FlowGraph = { nodes: {}, order: [] };

    const greet = createSetNode('greet', { message: 'Hello, World!' });
    root = addNode(root, greet);

    const pause = createWaitNode('pause', { duration: { seconds: 5 } });
    root = addNode(root, pause);

    let switchNode = createSwitchNode('route');
    switchNode = addSwitchBranch(
      switchNode,
      'fast-path',
      '${ $input.fast == true }',
    );
    switchNode = addSwitchBranch(switchNode, 'default');
    root = addNode(root, switchNode);

    let fork = createForkNode('parallel-work');
    fork = addForkBranch(fork, 'branch-a');
    fork = addForkBranch(fork, 'branch-b');
    root = addNode(root, fork);

    const tryNode = createTryNode('safe-call');
    root = addNode(root, tryNode);

    const loop = createLoopNode('process-items', '${ $input.items }');
    root = addNode(root, loop);

    return {
      ...file,
      workflows: {
        ...file.workflows,
        [rootId]: { ...file.workflows[rootId]!, root },
      },
    };
  }

  // Build the initial file outside reactive context so downstream $state
  // initialisers can reference it without triggering Svelte's warning about
  // capturing reactive values in non-reactive initialisers.
  const _initialFile = buildDemoFile();
  const _initialWorkflowId = _initialFile.order[0]!;

  let workflowFile = $state<WorkflowFile>(_initialFile);

  // ---------------------------------------------------------------------------
  // URL hash ↔ GraphPath sync
  //
  // Hash format: #<workflowId>/<segment1>/<segment2>/...
  // Example:     #abc-123 / #abc-123/nodeId / #abc-123/nodeId/branchId
  //
  // parseHashToGraphPath runs synchronously during initialisation so the
  // correct subgraph is shown from the first render without flickering.
  // ---------------------------------------------------------------------------

  // ---------------------------------------------------------------------------
  // Hash ID helpers — use metadata.__zigflow_id for stable URL segments.
  //
  // Nodes and branches carry a __zigflow_id in their metadata that is
  // preserved in YAML. The URL hash uses these IDs rather than the internal
  // node.id / branch.id, so navigation links survive YAML round-trips once
  // import is implemented.
  //
  // Both helpers fall back to the internal ID when __zigflow_id is absent,
  // ensuring existing sessions keep working without a forced migration.
  // ---------------------------------------------------------------------------

  function getNodeHashId(node: Node): string {
    return (node.metadata?.__zigflow_id as string | undefined) ?? node.id;
  }

  function getBranchHashId(branch: {
    id: string;
    metadata?: Record<string, unknown>;
  }): string {
    return (branch.metadata?.__zigflow_id as string | undefined) ?? branch.id;
  }

  // Convert internal GraphPath segments (node.id / branch.id) to URL-safe
  // hash segments (__zigflow_id). Traverses the WorkflowFile to resolve each
  // segment; returns [] if the path cannot be fully resolved.
  function graphPathToHashSegments(
    file: WorkflowFile,
    path: GraphPath,
  ): string[] {
    const wf = file.workflows[path.workflowId];
    if (!wf || path.segments.length === 0) return [];

    const hashSegs: string[] = [];
    let graph: FlowGraph = wf.root;
    let i = 0;

    while (i < path.segments.length) {
      const nodeId = path.segments[i]!;
      const node = graph.nodes[nodeId];
      if (!node) break;

      hashSegs.push(getNodeHashId(node));

      if (node.type === 'loop') {
        graph = node.bodyGraph;
        i += 1;
        continue;
      }

      // switch, fork, try — next segment identifies the sub-graph
      i += 1;
      if (i >= path.segments.length) break;
      const subId = path.segments[i]!;

      if (node.type === 'switch') {
        const branch = node.branches.find((b) => b.id === subId);
        if (!branch) break;
        hashSegs.push(getBranchHashId(branch));
        graph = branch.graph;
      } else if (node.type === 'fork') {
        const branch = node.branches.find((b) => b.id === subId);
        if (!branch) break;
        hashSegs.push(getBranchHashId(branch));
        graph = branch.graph;
      } else if (node.type === 'try') {
        // 'tryGraph' and 'catchGraph' are stable string constants, not IDs.
        hashSegs.push(subId);
        graph =
          subId === 'catchGraph'
            ? (node.catchGraph ?? node.tryGraph)
            : node.tryGraph;
      }

      i += 1;
    }

    return hashSegs;
  }

  // Find a node in graph whose __zigflow_id (or node.id fallback) matches hashId.
  function findNodeByHashId(
    graph: FlowGraph,
    hashId: string,
  ): Node | undefined {
    for (const nodeId of graph.order) {
      const node = graph.nodes[nodeId];
      if (!node) continue;
      if (getNodeHashId(node) === hashId) return node;
    }
    return undefined;
  }

  // Convert URL hash segments (__zigflow_id) back to internal GraphPath
  // segments (node.id / branch.id). Returns null if any segment cannot be
  // resolved.
  function hashSegmentsToGraphPath(
    file: WorkflowFile,
    workflowId: string,
    hashSegs: string[],
  ): GraphPath | null {
    const wf = file.workflows[workflowId];
    if (!wf) return null;
    if (hashSegs.length === 0) return { workflowId, segments: [] };

    const segments: string[] = [];
    let graph: FlowGraph = wf.root;
    let i = 0;

    while (i < hashSegs.length) {
      const hashId = hashSegs[i]!;
      const node = findNodeByHashId(graph, hashId);
      if (!node) return null;

      segments.push(node.id);

      if (node.type === 'loop') {
        graph = node.bodyGraph;
        i += 1;
        continue;
      }

      // switch, fork, try
      i += 1;
      if (i >= hashSegs.length) break;
      const subHashId = hashSegs[i]!;

      if (node.type === 'switch') {
        const branch = node.branches.find(
          (b) => getBranchHashId(b) === subHashId,
        );
        if (!branch) return null;
        segments.push(branch.id);
        graph = branch.graph;
      } else if (node.type === 'fork') {
        const branch = node.branches.find(
          (b) => getBranchHashId(b) === subHashId,
        );
        if (!branch) return null;
        segments.push(branch.id);
        graph = branch.graph;
      } else if (node.type === 'try') {
        if (subHashId !== 'tryGraph' && subHashId !== 'catchGraph') return null;
        segments.push(subHashId);
        graph =
          subHashId === 'catchGraph'
            ? (node.catchGraph ?? node.tryGraph)
            : node.tryGraph;
      }

      i += 1;
    }

    return { workflowId, segments };
  }

  function parseHashToGraphPath(
    file: WorkflowFile,
    defaultId: string,
  ): GraphPath {
    if (!browser) return { workflowId: defaultId, segments: [] };
    const raw = window.location.hash.slice(1);
    if (!raw) return { workflowId: defaultId, segments: [] };
    const parts = raw.split('/').filter(Boolean);
    const [workflowId, ...hashSegs] = parts;
    if (!workflowId || !file.workflows[workflowId]) {
      return { workflowId: defaultId, segments: [] };
    }
    if (hashSegs.length === 0) return { workflowId, segments: [] };
    const path = hashSegmentsToGraphPath(file, workflowId, hashSegs);
    return path ?? { workflowId, segments: [] };
  }

  // ---------------------------------------------------------------------------
  // Navigation state — GraphPath (Steps 4 & 6)
  //
  // graphPath.workflowId identifies the active workflow.
  // graphPath.segments encodes the path into nested sub-graphs.
  //
  // Derive the active FlowGraph from workflowFile + graphPath; never store
  // FlowGraph references directly so state stays stable across immutable updates.
  // ---------------------------------------------------------------------------

  const _initialPath = parseHashToGraphPath(_initialFile, _initialWorkflowId);

  let selectedWorkflowId = $state<string>(_initialPath.workflowId);
  let graphPath = $state<GraphPath>(_initialPath);
  let selectedNodeId = $state<string | null>(null);

  // Sync graphPath changes to the URL hash using __zigflow_id segments.
  $effect(() => {
    if (!browser) return;
    const hashSegs = graphPathToHashSegments(workflowFile, graphPath);
    const parts = [graphPath.workflowId, ...hashSegs].filter(Boolean);
    const newHash = '#' + parts.join('/');
    if (window.location.hash !== newHash) {
      window.location.hash = newHash;
    }
  });

  // Listen for browser back / forward navigation changing the hash externally.
  onMount(() => {
    function onHashChange() {
      const raw = window.location.hash.slice(1);
      const parts = (raw || '').split('/').filter(Boolean);
      const [workflowId, ...hashSegs] = parts;
      if (!workflowId) return;
      // Skip if hash already matches current state (hash sync wrote it).
      const currentHashSegs = graphPathToHashSegments(workflowFile, graphPath);
      if (
        workflowId === graphPath.workflowId &&
        hashSegs.join('/') === currentHashSegs.join('/')
      )
        return;
      if (!workflowFile.workflows[workflowId]) return;
      const path = hashSegmentsToGraphPath(workflowFile, workflowId, hashSegs);
      if (!path) return;
      graphPath = path;
      selectedWorkflowId = workflowId;
      selectedNodeId = null;
    }
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  });

  // ---------------------------------------------------------------------------
  // Derive the current FlowGraph (Step 5)
  // ---------------------------------------------------------------------------

  const currentGraph = $derived.by((): FlowGraph | null => {
    try {
      return getGraphAtPath(workflowFile, graphPath);
    } catch {
      return null;
    }
  });

  // ---------------------------------------------------------------------------
  // Breadcrumb labels + segment boundaries (for navigating back by crumb index)
  //
  // boundaries[i] = number of segments in graphPath.segments that correspond
  // to crumb i. Used by handleNavigate to trim segments precisely.
  // ---------------------------------------------------------------------------

  type CrumbInfo = { crumbs: string[]; boundaries: number[] };

  function buildCrumbInfo(file: WorkflowFile, path: GraphPath): CrumbInfo {
    const crumbs: string[] = [];
    const boundaries: number[] = [];

    const wf = file.workflows[path.workflowId];
    if (!wf) return { crumbs: [], boundaries: [] };

    crumbs.push(wf.name);
    boundaries.push(0); // root = 0 segments consumed

    let graph: FlowGraph = wf.root;
    let i = 0;

    while (i < path.segments.length) {
      const nodeId = path.segments[i];
      if (!nodeId) break;
      const node = graph.nodes[nodeId];
      if (!node) break;

      if (node.type === 'loop') {
        crumbs.push(`${node.name} › body`);
        boundaries.push(i + 1);
        graph = node.bodyGraph;
        i += 1;
        continue;
      }

      // switch, fork, try — consume two segments
      const nextI = i + 1;
      if (nextI >= path.segments.length) break;
      const subId = path.segments[nextI];
      if (!subId) break;

      if (node.type === 'switch') {
        const branch = node.branches.find((b) => b.id === subId);
        crumbs.push(`${node.name} › ${branch?.label ?? subId}`);
        boundaries.push(nextI + 1);
        if (branch) graph = branch.graph;
        i = nextI + 1;
      } else if (node.type === 'fork') {
        const branch = node.branches.find((b) => b.id === subId);
        crumbs.push(`${node.name} › ${branch?.label ?? subId}`);
        boundaries.push(nextI + 1);
        if (branch) graph = branch.graph;
        i = nextI + 1;
      } else if (node.type === 'try') {
        const label = subId === 'catchGraph' ? 'catch' : 'try';
        crumbs.push(`${node.name} › ${label}`);
        boundaries.push(nextI + 1);
        graph =
          subId === 'catchGraph'
            ? (node.catchGraph ?? node.tryGraph)
            : node.tryGraph;
        i = nextI + 1;
      } else {
        break;
      }
    }

    return { crumbs, boundaries };
  }

  const crumbInfo = $derived(buildCrumbInfo(workflowFile, graphPath));
  const breadcrumbs = $derived(crumbInfo.crumbs);

  // ---------------------------------------------------------------------------
  // Context indicator label — derives a human-readable sentence that describes
  // exactly where the user is currently editing.
  // ---------------------------------------------------------------------------

  function buildContextLabel(file: WorkflowFile, path: GraphPath): string {
    const wf = file.workflows[path.workflowId];
    if (!wf) return '';

    if (path.segments.length === 0) {
      return `Editing Workflow: ${wf.name}`;
    }

    let graph: FlowGraph = wf.root;
    let i = 0;
    let label = '';

    while (i < path.segments.length) {
      const nodeId = path.segments[i];
      if (!nodeId) break;
      const node = graph.nodes[nodeId];
      if (!node) break;

      if (node.type === 'loop') {
        label = `Editing Loop Body: ${node.name}`;
        graph = node.bodyGraph;
        i += 1;
        continue;
      }

      const nextI = i + 1;
      if (nextI >= path.segments.length) break;
      const subId = path.segments[nextI];
      if (!subId) break;

      if (node.type === 'switch') {
        const branch = node.branches.find((b) => b.id === subId);
        label = `Editing Switch Branch: ${branch?.label ?? subId}`;
        if (branch) graph = branch.graph;
        i = nextI + 1;
      } else if (node.type === 'fork') {
        const branch = node.branches.find((b) => b.id === subId);
        label = `Editing Fork Branch: ${branch?.label ?? subId}`;
        if (branch) graph = branch.graph;
        i = nextI + 1;
      } else if (node.type === 'try') {
        const section = subId === 'catchGraph' ? 'catch' : 'try';
        label = `Editing Try Section: ${section} (${node.name})`;
        graph =
          subId === 'catchGraph'
            ? (node.catchGraph ?? node.tryGraph)
            : node.tryGraph;
        i = nextI + 1;
      } else {
        break;
      }
    }

    return label;
  }

  const contextLabel = $derived(buildContextLabel(workflowFile, graphPath));

  // ---------------------------------------------------------------------------
  // Selected node (resolved from current graph)
  // ---------------------------------------------------------------------------

  const selectedNode = $derived<Node | null>(
    selectedNodeId && currentGraph
      ? (currentGraph.nodes[selectedNodeId] ?? null)
      : null,
  );

  // ---------------------------------------------------------------------------
  // Navigation helpers (Step 6)
  //
  // All three functions update graphPath only — no WorkflowFile mutation.
  // ---------------------------------------------------------------------------

  // Navigate into a LoopNode's bodyGraph (single sub-graph).
  function navigateInto(nodeId: string) {
    graphPath = { ...graphPath, segments: [...graphPath.segments, nodeId] };
    selectedNodeId = null;
  }

  // Navigate into a branch or named section of a SwitchNode, ForkNode, or TryNode.
  // For TryNode use branchId = 'tryGraph' or 'catchGraph'.
  function navigateIntoBranch(nodeId: string, branchId: string) {
    graphPath = {
      ...graphPath,
      segments: [...graphPath.segments, nodeId, branchId],
    };
    selectedNodeId = null;
  }

  // Navigate up one level in the crumb trail.
  function navigateBack() {
    const { boundaries } = crumbInfo;
    if (boundaries.length <= 1) return;
    const targetBoundary = boundaries[boundaries.length - 2] ?? 0;
    graphPath = {
      ...graphPath,
      segments: graphPath.segments.slice(0, targetBoundary),
    };
    selectedNodeId = null;
  }

  // ---------------------------------------------------------------------------
  // Event handlers
  // ---------------------------------------------------------------------------

  function handleWorkflowSelect(id: string) {
    selectedWorkflowId = id;
    graphPath = { workflowId: id, segments: [] };
    selectedNodeId = null;
  }

  function handleAddWorkflow() {
    workflowFile = addWorkflow(workflowFile, 'new-workflow');
    const newId = workflowFile.order[workflowFile.order.length - 1]!;
    handleWorkflowSelect(newId);
  }

  function handleNodeSelect(nodeId: string | null) {
    selectedNodeId = nodeId;
  }

  // Navigate to a crumb by its index. The Breadcrumb component passes the
  // array index; we map it to a segment count using boundaries.
  function handleNavigate(index: number) {
    const boundary = crumbInfo.boundaries[index] ?? 0;
    graphPath = {
      ...graphPath,
      segments: graphPath.segments.slice(0, boundary),
    };
    selectedNodeId = null;
  }

  // Insert a new node into the currently active graph (Step 5 wiring).
  function handleInsert(nodeType: NodeType) {
    workflowFile = insertNodeAtPath(workflowFile, graphPath, nodeType);
  }

  // Apply a pure FlowGraph transform at the current graphPath, threading the
  // result back up to produce a new WorkflowFile.
  function updateCurrentGraph(transform: (g: FlowGraph) => FlowGraph): void {
    try {
      workflowFile = updateGraphAtPath(workflowFile, graphPath, transform);
    } catch (err) {
      console.error('updateCurrentGraph failed:', err);
    }
  }

  function handleDelete() {
    const id = selectedNodeId;
    if (!id) return;
    selectedNodeId = null;
    updateCurrentGraph((g) => removeNode(g, id));
  }

  const selectedNodeIndex = $derived(
    selectedNodeId && currentGraph
      ? currentGraph.order.indexOf(selectedNodeId)
      : -1,
  );
  const canMoveUp = $derived(selectedNodeIndex > 0);
  const canMoveDown = $derived(
    selectedNodeIndex >= 0 &&
      currentGraph !== null &&
      selectedNodeIndex < currentGraph.order.length - 1,
  );

  function handleMoveUp() {
    const id = selectedNodeId;
    if (!id || !canMoveUp) return;
    updateCurrentGraph((g) => moveNode(g, id, selectedNodeIndex - 1));
  }

  function handleMoveDown() {
    const id = selectedNodeId;
    if (!id || !canMoveDown) return;
    updateCurrentGraph((g) => moveNode(g, id, selectedNodeIndex + 1));
  }

  // ---------------------------------------------------------------------------
  // Navigation callbacks (forwarded from Inspector)
  // ---------------------------------------------------------------------------

  function handleEnterNode(nodeId: string) {
    navigateInto(nodeId);
  }

  function handleEnterBranch(nodeId: string, branchId: string) {
    navigateIntoBranch(nodeId, branchId);
  }

  // ---------------------------------------------------------------------------
  // Branch management callbacks (forwarded from Inspector)
  // ---------------------------------------------------------------------------

  function resolveNode(nodeId: string): Node | null {
    return currentGraph?.nodes[nodeId] ?? null;
  }

  function handleAddBranch(nodeId: string) {
    const node = resolveNode(nodeId);
    if (!node) return;
    if (node.type === 'switch') {
      updateCurrentGraph((g) =>
        replaceNode(g, addSwitchBranch(node, 'new-branch')),
      );
    } else if (node.type === 'fork') {
      updateCurrentGraph((g) =>
        replaceNode(g, addForkBranch(node, 'new-branch')),
      );
    }
  }

  function handleRemoveBranch(nodeId: string, branchId: string) {
    const node = resolveNode(nodeId);
    if (!node) return;
    if (node.type === 'switch') {
      updateCurrentGraph((g) =>
        replaceNode(g, removeSwitchBranch(node, branchId)),
      );
    } else if (node.type === 'fork') {
      updateCurrentGraph((g) =>
        replaceNode(g, removeForkBranch(node, branchId)),
      );
    }
  }

  function handleRenameBranch(nodeId: string, branchId: string, label: string) {
    const node = resolveNode(nodeId);
    if (!node) return;
    if (node.type === 'switch') {
      updateCurrentGraph((g) =>
        replaceNode(g, renameSwitchBranch(node, branchId, label)),
      );
    } else if (node.type === 'fork') {
      updateCurrentGraph((g) =>
        replaceNode(g, renameForkBranch(node, branchId, label)),
      );
    }
  }

  // Add a catchGraph to a TryNode and navigate into it immediately.
  function handleAddCatch(nodeId: string) {
    const node = resolveNode(nodeId);
    if (!node || node.type !== 'try' || node.catchGraph !== undefined) return;
    updateCurrentGraph((g) =>
      replaceNode(g, updateTrySection(node, 'catchGraph', emptyFlowGraph())),
    );
    navigateIntoBranch(nodeId, 'catchGraph');
  }

  // Expose navigation helpers so child components can invoke them in future.
  // Currently unused in the template; defined here so the API is established.
  export { navigateInto, navigateIntoBranch, navigateBack };

  // ---------------------------------------------------------------------------
  // YAML export
  // ---------------------------------------------------------------------------

  let exportOutput = $state<string>('');
  let exportError = $state<string>('');
  let showExport = $state(false);

  function handleExport() {
    const result = exportToYaml(workflowFile);
    if (result.ok) {
      exportOutput = result.yaml;
      exportError = '';
    } else {
      exportOutput = '';
      exportError = result.errors.join('\n');
    }
    showExport = true;
  }
</script>

<div class="editor-root">
  <!-- Sidebar: document + workflow list -->
  <Sidebar
    file={workflowFile}
    {selectedWorkflowId}
    onworkflowselect={handleWorkflowSelect}
    onaddworkflow={handleAddWorkflow}
  />

  <!-- Main area: breadcrumb + context indicator + canvas + inspector -->
  <div class="editor-main">
    <div class="editor-topbar">
      <Breadcrumb crumbs={breadcrumbs} onnavigate={handleNavigate} />
      <button class="export-btn" onclick={handleExport} type="button">
        Export YAML
      </button>
    </div>

    <ContextIndicator label={contextLabel} />

    <div class="editor-canvas-area">
      {#if currentGraph !== null}
        <Canvas
          graph={currentGraph}
          {selectedNodeId}
          onnodeselect={handleNodeSelect}
          oninsert={handleInsert}
          onenternode={handleEnterNode}
          onenterbranch={handleEnterBranch}
        />
      {:else}
        <div class="canvas-placeholder">No graph to display.</div>
      {/if}

      <Inspector
        node={selectedNode}
        {canMoveUp}
        {canMoveDown}
        onmoveup={handleMoveUp}
        onmovedown={handleMoveDown}
        ondelete={handleDelete}
        onenternode={handleEnterNode}
        onenterbranch={handleEnterBranch}
        onaddbranch={handleAddBranch}
        onremovebranch={handleRemoveBranch}
        onrenamebranch={handleRenameBranch}
        onaddcatch={handleAddCatch}
      />
    </div>
  </div>
</div>

<!-- YAML export overlay -->
{#if showExport}
  <div
    class="export-overlay"
    role="dialog"
    aria-modal="true"
    aria-label="YAML export"
  >
    <div class="export-dialog">
      <div class="export-dialog-header">
        <h2>Exported YAML</h2>
        <button
          class="export-close-btn"
          onclick={() => {
            showExport = false;
          }}
          type="button"
          aria-label="Close">✕</button
        >
      </div>
      {#if exportError}
        <pre class="export-error">{exportError}</pre>
      {:else}
        <pre class="export-code">{exportOutput}</pre>
      {/if}
    </div>
  </div>
{/if}

<style>
  .editor-root {
    height: 100vh;
    display: flex;
    overflow: hidden;
    font-family: system-ui, sans-serif;
  }

  .editor-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .editor-topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid #eee;
    background: #fff;
    padding-right: 1rem;
  }

  .export-btn {
    padding: 0.3rem 0.75rem;
    background: #1a56cc;
    color: #fff;
    border: none;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
  }

  .export-btn:hover {
    background: #1344a8;
  }

  .editor-canvas-area {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .canvas-placeholder {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #888;
    font-size: 0.9rem;
  }

  /* Export overlay */
  .export-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .export-dialog {
    background: #fff;
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    width: min(720px, 90vw);
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .export-dialog-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid #eee;
  }

  .export-dialog-header h2 {
    margin: 0;
    font-size: 1rem;
  }

  .export-close-btn {
    background: none;
    border: none;
    font-size: 1rem;
    cursor: pointer;
    color: #666;
    padding: 0.25rem;
    line-height: 1;
  }

  .export-code,
  .export-error {
    flex: 1;
    margin: 0;
    padding: 1rem 1.25rem;
    overflow-y: auto;
    font-size: 0.8rem;
    white-space: pre-wrap;
    font-family: 'Courier New', monospace;
  }

  .export-error {
    color: #c0392b;
    background: #fff5f5;
  }
</style>

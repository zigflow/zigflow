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
  import type { FlowGraph, Node, NodeType } from '$lib/tasks/model';
  import { Background, Controls, SvelteFlow } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import { untrack } from 'svelte';

  import FlowNode from './FlowNode.svelte';

  // ---------------------------------------------------------------------------
  // Custom node types — defined as a stable constant so SvelteFlow does not
  // remount nodes when the Canvas component re-renders.
  // ---------------------------------------------------------------------------

  const nodeTypes = { flow: FlowNode };

  // ---------------------------------------------------------------------------
  // Props
  // ---------------------------------------------------------------------------

  interface Props {
    graph: FlowGraph;
    selectedNodeId?: string | null;
    onnodeselect?: (nodeId: string | null) => void;
    oninsert?: (nodeType: NodeType) => void;
    // Navigation into structural node subgraphs.
    // onenternode: single sub-graph (loop body).
    // onenterbranch: named sub-graph (switch/fork branch, try section).
    onenternode?: (nodeId: string) => void;
    onenterbranch?: (nodeId: string, branchId: string) => void;
  }

  let {
    graph,
    selectedNodeId = null,
    onnodeselect,
    oninsert,
    onenternode,
    onenterbranch,
  }: Props = $props();

  // ---------------------------------------------------------------------------
  // Layout constants
  // ---------------------------------------------------------------------------

  const NODE_WIDTH = 240;
  const NODE_HEIGHT_TASK = 60;
  const ROW_HEADER = 28; // header row in structural nodes
  const ROW_HEIGHT = 22; // each nav row in structural nodes
  const ROW_PADDING = 8; // bottom padding in structural nodes
  const VERTICAL_GAP = 40;

  // ---------------------------------------------------------------------------
  // Per-node height — structural nodes grow to fit their nav rows.
  // ---------------------------------------------------------------------------

  function nodeHeight(node: Node): number {
    if (node.type === 'switch' || node.type === 'fork') {
      return Math.max(
        NODE_HEIGHT_TASK,
        ROW_HEADER + node.branches.length * ROW_HEIGHT + ROW_PADDING,
      );
    }
    if (node.type === 'try') {
      const rows = node.catchGraph !== undefined ? 2 : 1;
      return Math.max(
        NODE_HEIGHT_TASK,
        ROW_HEADER + rows * ROW_HEIGHT + ROW_PADDING,
      );
    }
    // loop: header(28) + body row(22) + padding(8) = 58 ≈ 60
    return NODE_HEIGHT_TASK;
  }

  // ---------------------------------------------------------------------------
  // Derive SvelteFlow nodes and edges from the FlowGraph IR.
  // Y positions are accumulated from variable node heights.
  // ---------------------------------------------------------------------------

  function nodeTypeLabel(type: string): string {
    const labels: Record<string, string> = {
      task: 'Task',
      switch: 'Switch',
      fork: 'Fork',
      try: 'Try/Catch',
      loop: 'Loop',
    };
    return labels[type] ?? type;
  }

  // NavRow shape matches FlowNode.NavRow
  type NavRow = { id: string; label: string; onclick: () => void };

  type SFNodeData = {
    label: string;
    nodeType: string;
    typeLabel: string;
    navRows?: NavRow[];
  };

  type SFNode = {
    id: string;
    type: 'flow';
    data: SFNodeData;
    position: { x: number; y: number };
    style: string;
    selected: boolean;
  };

  type SFEdge = {
    id: string;
    source: string;
    target: string;
  };

  // Build the navRows array for a structural node.
  // Callbacks are curried with the node ID so FlowNode stays ID-unaware.
  function buildNavRows(node: Node): NavRow[] {
    if (node.type === 'switch' || node.type === 'fork') {
      return node.branches.map((b) => ({
        id: b.id,
        label: b.label,
        onclick: () => onenterbranch?.(node.id, b.id),
      }));
    }
    if (node.type === 'try') {
      const rows: NavRow[] = [
        {
          id: 'tryGraph',
          label: 'try body',
          onclick: () => onenterbranch?.(node.id, 'tryGraph'),
        },
      ];
      if (node.catchGraph !== undefined) {
        rows.push({
          id: 'catchGraph',
          label: 'catch block',
          onclick: () => onenterbranch?.(node.id, 'catchGraph'),
        });
      }
      return rows;
    }
    if (node.type === 'loop') {
      return [
        {
          id: 'body',
          label: 'body',
          onclick: () => onenternode?.(node.id),
        },
      ];
    }
    return [];
  }

  function buildNodeData(node: Node): SFNodeData {
    const base: SFNodeData = {
      label: node.name,
      nodeType: node.type,
      typeLabel: nodeTypeLabel(node.type),
    };
    if (node.type !== 'task') {
      base.navRows = buildNavRows(node);
    }
    return base;
  }

  function deriveNodes(g: FlowGraph): SFNode[] {
    let y = 0;
    return g.order.map((id) => {
      const node = g.nodes[id]!;
      const h = nodeHeight(node);
      const sfNode: SFNode = {
        id: node.id,
        type: 'flow' as const,
        data: buildNodeData(node),
        position: { x: 0, y },
        style: `width: ${NODE_WIDTH}px; height: ${h}px;`,
        selected: id === selectedNodeId,
      };
      y += h + VERTICAL_GAP;
      return sfNode;
    });
  }

  function deriveEdges(g: FlowGraph): SFEdge[] {
    return g.order.slice(0, -1).map((id, index) => ({
      id: `seq-${id}-${g.order[index + 1]}`,
      source: id,
      target: g.order[index + 1]!,
    }));
  }

  // Use untrack() so the initial $state value reads the prop without creating
  // a reactive dependency. The $effect below handles all subsequent updates.
  let nodes = $state(untrack(() => deriveNodes(graph)));
  let edges = $state(untrack(() => deriveEdges(graph)));

  // Resync when graph, selection, or navigation callbacks change.
  $effect(() => {
    nodes = deriveNodes(graph);
    edges = deriveEdges(graph);
  });

  // ---------------------------------------------------------------------------
  // Selection forwarding
  // ---------------------------------------------------------------------------

  type SelectionParams = { nodes: { id: string }[]; edges: { id: string }[] };

  function handleSelectionChange(params: SelectionParams) {
    const selected = params.nodes ?? [];
    onnodeselect?.(selected.length > 0 ? (selected[0]?.id ?? null) : null);
  }

  // ---------------------------------------------------------------------------
  // Drag-and-drop: accept palette items dropped onto the canvas
  // ---------------------------------------------------------------------------

  function handleDragOver(event: DragEvent) {
    if (event.dataTransfer?.types.includes('application/node-type')) {
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy';
    }
  }

  function handleDrop(event: DragEvent) {
    event.preventDefault();
    const nodeType = event.dataTransfer?.getData('application/node-type') as
      | NodeType
      | undefined;
    if (nodeType) oninsert?.(nodeType);
  }
</script>

<div
  class="canvas-root"
  role="region"
  aria-label="Workflow canvas"
  ondragover={handleDragOver}
  ondrop={handleDrop}
>
  <SvelteFlow
    bind:nodes
    bind:edges
    {nodeTypes}
    fitView
    onselectionchange={handleSelectionChange}
  >
    <Background />
    <Controls />
  </SvelteFlow>
</div>

<style>
  .canvas-root {
    width: 100%;
    height: 100%;
    background: #f8f8f8;
  }
</style>

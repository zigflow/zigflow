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
  import {
    SvelteFlow,
    Controls,
    Background,
    MiniMap,
    useSvelteFlow,
  } from '@xyflow/svelte';
  import { onMount } from 'svelte';
  import type { Node, Edge } from '$lib/types/flow';

  let {
    nodes = $bindable(),
    edges = $bindable(),
    nodeId = $bindable(),
  }: {
    nodes: Node[];
    edges: Edge[];
    nodeId: number;
  } = $props();

  const { screenToFlowPosition, getNodes } = useSvelteFlow();

  // Watch for node selection changes and sync to parent state
  let lastSelectedIds = '';

  $effect(() => {
    const currentNodes = getNodes();
    const selectedIds = currentNodes
      .filter((n) => n.selected)
      .map((n) => n.id)
      .sort()
      .join(',');

    // Only update if selection actually changed
    if (selectedIds !== lastSelectedIds) {
      lastSelectedIds = selectedIds;
      nodes = currentNodes as Node[];
    }
  });

  function onDragOver(event: DragEvent) {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move';
    }
  }

  function onDrop(event: DragEvent) {
    event.preventDefault();

    if (!event.dataTransfer) return;

    const type = event.dataTransfer.getData('application/svelteflow');
    if (!type) return;

    const position = screenToFlowPosition({
      x: event.clientX,
      y: event.clientY,
    });

    const newNode = {
      id: String(nodeId),
      type: type === 'default' ? undefined : type,
      data: {
        label:
          type === 'input'
            ? 'Start'
            : type === 'output'
              ? 'End'
              : `Task ${nodeId}`,
      },
      position,
    };

    nodes = [...nodes, newNode];
    nodeId++;
  }

  // Prevent browser back navigation when deleting nodes with backspace
  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'Backspace') {
      // Check if any node is selected using SvelteFlow's getNodes
      const currentNodes = getNodes();
      const hasSelectedNode = currentNodes.some((node) => node.selected);
      if (hasSelectedNode) {
        // Prevent browser back navigation
        event.preventDefault();
      }
    }
  }

  onMount(() => {
    // Use capture phase to intercept the event earlier
    document.addEventListener('keydown', handleKeyDown, true);

    return () => {
      document.removeEventListener('keydown', handleKeyDown, true);
    };
  });
</script>

<div
  class="flow-canvas"
  role="application"
  aria-label="Workflow canvas"
  ondragover={onDragOver}
  ondrop={onDrop}
>
  <SvelteFlow {nodes} {edges} fitView>
    <Controls />
    <Background />
    <MiniMap />
  </SvelteFlow>
</div>

<style lang="scss">
  .flow-canvas {
    flex: 1;
    position: relative;
  }
</style>

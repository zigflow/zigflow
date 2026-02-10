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
  import type { Edge, Node } from '$lib/types/flow';
  import FlowCanvas from '$lib/ui/FlowCanvas.svelte';
  import NodeSettings from '$lib/ui/NodeSettings.svelte';
  import Sidebar from '$lib/ui/Sidebar.svelte';
  import { SvelteFlowProvider } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';

  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const workflowId = $derived(data.workflowId);

  let nodeId = $state(5); // Counter for generating unique node IDs

  // Initial nodes - example workflow using Zigflow tasks
  let nodes = $state<Node[]>([
    {
      id: '1',
      type: 'call-http',
      data: { label: 'Fetch Data' },
      position: { x: 250, y: 0 },
    },
    {
      id: '2',
      type: 'set',
      data: { label: 'Transform Data' },
      position: { x: 100, y: 120 },
    },
    {
      id: '3',
      type: 'fork',
      data: { label: 'Process in Parallel' },
      position: { x: 400, y: 120 },
    },
    {
      id: '4',
      type: 'wait',
      data: { label: 'Wait 5s' },
      position: { x: 250, y: 240 },
    },
  ]);

  // Initial edges
  let edges = $state<Edge[]>([
    { id: 'e1-2', source: '1', target: '2' },
    { id: 'e1-3', source: '1', target: '3' },
    { id: 'e2-4', source: '2', target: '4' },
    { id: 'e3-4', source: '3', target: '4' },
  ]);

  // Track selected node
  const selectedNode = $derived(nodes.find((node) => node.selected));

  // Handle closing the settings panel by deselecting all nodes
  function handleCloseSettings() {
    nodes = nodes.map((node) => ({ ...node, selected: false }));
  }
</script>

<div class="workflow-editor">
  <header>
    <h1>Workflow Editor</h1>
    <p>Workflow ID: <code>{workflowId}</code></p>
  </header>

  <SvelteFlowProvider>
    <div class="editor-layout">
      <Sidebar />
      <FlowCanvas bind:nodes bind:edges bind:nodeId />
      {#if selectedNode}
        <NodeSettings node={selectedNode} onClose={handleCloseSettings} />
      {/if}
    </div>
  </SvelteFlowProvider>
</div>

<style lang="scss">
  @use '../../../styles/tokens' as *;

  .workflow-editor {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }

  header {
    padding: $spacing-lg;
    border-bottom: 1px solid $color-border;
    background-color: $color-bg;

    h1 {
      margin: 0 0 $spacing-xs 0;
    }

    p {
      margin: 0;
    }
  }

  .editor-layout {
    flex: 1;
    display: flex;
    overflow: hidden;
  }
</style>

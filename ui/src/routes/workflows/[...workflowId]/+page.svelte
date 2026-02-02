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
  import { SvelteFlow, Controls, Background, MiniMap } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const workflowId = $derived(data.workflowId);

  // Initial nodes
  let nodes = $state([
    {
      id: '1',
      type: 'input',
      data: { label: 'Start' },
      position: { x: 250, y: 0 },
    },
    {
      id: '2',
      data: { label: 'Task 1' },
      position: { x: 100, y: 100 },
    },
    {
      id: '3',
      data: { label: 'Task 2' },
      position: { x: 400, y: 100 },
    },
    {
      id: '4',
      type: 'output',
      data: { label: 'End' },
      position: { x: 250, y: 200 },
    },
  ]);

  // Initial edges
  let edges = $state([
    { id: 'e1-2', source: '1', target: '2' },
    { id: 'e1-3', source: '1', target: '3' },
    { id: 'e2-4', source: '2', target: '4' },
    { id: 'e3-4', source: '3', target: '4' },
  ]);
</script>

<div class="workflow-container">
  <h1>Workflow Editor</h1>
  <p>Workflow ID: <code>{workflowId}</code></p>

  <div class="flow-wrapper">
    <SvelteFlow {nodes} {edges} fitView>
      <Controls />
      <Background />
      <MiniMap />
    </SvelteFlow>
  </div>
</div>

<style>
  .workflow-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 1rem;
  }

  .flow-wrapper {
    flex: 1;
    border: 1px solid #ddd;
    border-radius: 8px;
    overflow: hidden;
  }

  h1 {
    margin: 0 0 0.5rem 0;
  }

  p {
    margin: 0 0 1rem 0;
  }
</style>

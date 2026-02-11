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
  import { resolve } from '$app/paths';
  import FlowCanvas from '$lib/ui/FlowCanvas.svelte';
  import NodeSettings from '$lib/ui/NodeSettings.svelte';
  import Sidebar from '$lib/ui/Sidebar.svelte';
  import { type Edge, type Node, SvelteFlowProvider } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';

  const props = $props();

  // Make isDirectory reactive so it updates when navigating
  const isDirectory = $derived(props.data.type === 'directory');
  const workflowId = props.data.workflowId;

  // View mode toggle: 'graph' or 'yaml'
  let viewMode = $state<'graph' | 'yaml'>('graph');

  // YAML content (always editable in YAML view)
  let editedYaml = $state('');

  let nodeId = $state(5); // Counter for generating unique node IDs

  // Initialize editedYaml when switching to YAML view
  $effect(() => {
    if (viewMode === 'yaml' && props.data.type === 'workflow') {
      editedYaml = props.data.workflowYaml;
    }
  });

  let nodes = $state.raw<Node[]>([]);
  let edges = $state.raw<Edge[]>([]);

  // Update nodes and edges when data changes
  $effect(() => {
    if (props.data.type === 'workflow') {
      nodes = props.data.graph.nodes;
      edges = props.data.graph.edges;
    } else {
      nodes = [];
      edges = [];
    }
  });

  // Track selected node
  const selectedNode = $derived(nodes.find((node) => node.selected));

  // Handle closing the settings panel by deselecting all nodes
  function handleCloseSettings() {
    nodes = nodes.map((node) => ({ ...node, selected: false }));
  }

  // Save edited YAML
  async function saveYaml() {
    if (props.data.type !== 'workflow') return;

    try {
      const response = await fetch(`/workflows/${workflowId}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          yaml: editedYaml,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        alert(`Failed to save: ${error}`);
        return;
      }

      // Reload the page to show the updated workflow
      window.location.reload();
    } catch (error) {
      alert(`Error saving workflow: ${error}`);
    }
  }
</script>

{#if isDirectory}
  <div class="directory-list">
    <h1>
      Directory: {props.data.type === 'directory'
        ? props.data.currentPath || '/'
        : '/'}
    </h1>

    {#if props.data.type === 'directory' && props.data.entries.length === 0}
      <p>Directory empty</p>
    {:else if props.data.type === 'directory'}
      <ul>
        {#each props.data.entries as entry (entry.path)}
          <li>
            <a href={resolve(`/workflows/${entry.path}`)}>
              {entry.name}{entry.isDirectory ? '/' : ''}
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{:else}
  <div class="workflow-editor">
    <header>
      <div class="header-content">
        <div>
          <h1>Workflow Editor</h1>
          <p>Workflow ID: <code>{workflowId}</code></p>
        </div>
        <button
          class="view-toggle"
          onclick={() => (viewMode = viewMode === 'graph' ? 'yaml' : 'graph')}
        >
          {viewMode === 'graph' ? 'View YAML' : 'View Graph'}
        </button>
      </div>
    </header>

    {#if viewMode === 'graph'}
      <SvelteFlowProvider>
        <div class="editor-layout">
          <Sidebar />
          <FlowCanvas bind:nodes bind:edges bind:nodeId />
          {#if selectedNode}
            <NodeSettings node={selectedNode} onClose={handleCloseSettings} />
          {/if}
        </div>
      </SvelteFlowProvider>
    {:else}
      <div class="yaml-view">
        <div class="yaml-controls">
          <button class="yaml-button save" onclick={saveYaml}>Save</button>
        </div>

        <textarea class="yaml-editor" bind:value={editedYaml} spellcheck="false"
        ></textarea>
      </div>
    {/if}
  </div>
{/if}

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

    .header-content {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    h1 {
      margin: 0 0 $spacing-xs 0;
    }

    p {
      margin: 0;
    }

    .view-toggle {
      padding: $spacing-sm $spacing-md;
      background-color: $color-primary;
      color: white;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;
      transition: background-color 0.2s;

      &:hover {
        background-color: darken($color-primary, 10%);
      }
    }
  }

  .editor-layout {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .directory-list {
    padding: $spacing-lg;
  }

  .directory-list h1 {
    margin: 0 0 $spacing-lg 0;
  }

  .directory-list ul {
    list-style: none;
    padding: 0;
  }

  .directory-list li {
    margin: $spacing-sm 0;
  }

  .directory-list a {
    color: $color-text;
    text-decoration: none;
  }

  .directory-list a:hover {
    text-decoration: underline;
  }

  .yaml-view {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: #1e1e1e;

    .yaml-controls {
      display: flex;
      gap: $spacing-sm;
      padding: $spacing-md;
      border-bottom: 1px solid #333;
    }

    .yaml-button {
      padding: $spacing-xs $spacing-md;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;
      transition: background-color 0.2s;

      &.save {
        background-color: #28a745;
        color: white;

        &:hover {
          background-color: darken(#28a745, 10%);
        }
      }
    }

    .yaml-editor {
      flex: 1;
      padding: $spacing-lg;
      background-color: #1e1e1e;
      color: #d4d4d4;
      font-family: 'Courier New', monospace;
      font-size: 14px;
      line-height: 1.5;
      border: none;
      resize: none;
      outline: none;
    }
  }
</style>

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

<!--
  FlowNode — custom Svelte Flow node registered as nodeType 'flow'.

  Task nodes: horizontal layout with type badge and name.

  Structural nodes (switch, fork, try, loop): vertical layout with a fixed
  header row followed by clickable navigation rows — one per branch or section.
  Clicking a row calls the curried callback supplied by Canvas via node data.
  Buttons use `stopPropagation` to prevent SvelteFlow from also handling the
  click, and carry the `nodrag` class so dragging cannot start on them.

  Node dimensions are set by Canvas via the `style` prop on the SvelteFlow
  node spec; this component fills 100% of that wrapper.
-->

<script lang="ts">
  import { Handle, Position } from '@xyflow/svelte';

  // ---------------------------------------------------------------------------
  // Data shape populated by Canvas.buildNodeData
  // ---------------------------------------------------------------------------

  type NavRow = {
    id: string;
    label: string;
    onclick: () => void;
  };

  type NodeData = {
    label: string;
    nodeType: string; // 'task' | 'switch' | 'fork' | 'try' | 'loop'
    typeLabel: string;
    // Structural nodes provide nav rows; task nodes do not.
    navRows?: NavRow[];
    // Synchronous click callback for URL updates — called during the DOM
    // click event so Playwright detects the history change immediately.
    onselect?: (id: string) => void;
  };

  interface Props {
    id: string;
    data: NodeData;
    selected?: boolean;
  }

  let { id, data, selected = false }: Props = $props();

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  const isStructural = $derived(data.nodeType !== 'task');

  function accentColor(type: string): string {
    switch (type) {
      case 'switch':
        return '#d97706';
      case 'fork':
        return '#2563eb';
      case 'try':
        return '#dc2626';
      case 'loop':
        return '#16a34a';
      default:
        return '#6b7280';
    }
  }
</script>

<div
  class="flow-node"
  class:flow-node--structural={isStructural}
  class:flow-node--selected={selected}
  style="--accent: {accentColor(data.nodeType)}"
  onclick={() => data.onselect?.(id)}
  onkeydown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') data.onselect?.(id);
  }}
  role="button"
  tabindex="0"
>
  <Handle type="target" position={Position.Top} />

  {#if isStructural}
    <!-- Structural layout: fixed header + clickable nav rows -->
    <div class="flow-node-header">
      <span class="flow-node-type">{data.typeLabel}</span>
      <span class="flow-node-name">{data.label}</span>
    </div>

    {#if data.navRows && data.navRows.length > 0 && !selected}
      <ul class="flow-node-rows" role="list">
        {#each data.navRows as row (row.id)}
          <li>
            <button
              class="flow-node-row nodrag"
              type="button"
              onclick={(e) => {
                e.stopPropagation();
                row.onclick();
              }}
            >
              <span class="flow-node-row-arrow" aria-hidden="true">→</span>
              {row.label}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {:else}
    <!-- Task layout: horizontal, centered -->
    <div class="flow-node-body">
      <span class="flow-node-type">{data.typeLabel}</span>
      <span class="flow-node-name" data-selected={selected ? 'true' : null}
        >{data.label}</span
      >
    </div>
  {/if}

  <Handle type="source" position={Position.Bottom} />
</div>

<style>
  /* -------------------------------------------------------------------------
     Base
  ------------------------------------------------------------------------- */

  .flow-node {
    width: 100%;
    height: 100%;
    background: #fff;
    border: 1px solid #ddd;
    border-left: 3px solid var(--accent);
    border-radius: 6px;
    box-sizing: border-box;
    display: flex;
    align-items: center;
    overflow: hidden;
    font-family: system-ui, sans-serif;
    font-size: 0.75rem;
    cursor: pointer;
    transition: box-shadow 0.1s;
  }

  .flow-node--structural {
    flex-direction: column;
    align-items: stretch;
    background: #fafafa;
  }

  .flow-node--selected {
    box-shadow: 0 0 0 2px var(--accent);
    border-color: var(--accent);
  }

  /* -------------------------------------------------------------------------
     Task layout (horizontal)
  ------------------------------------------------------------------------- */

  .flow-node-body {
    padding: 0 0.625rem;
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
    flex: 1;
  }

  /* -------------------------------------------------------------------------
     Structural layout (vertical: header + rows)
  ------------------------------------------------------------------------- */

  .flow-node-header {
    height: 28px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0 0.5rem;
    border-bottom: 1px solid #eee;
  }

  .flow-node-rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    flex: 1;
  }

  .flow-node-row {
    height: 22px;
    width: 100%;
    padding: 0 0.5rem;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    background: transparent;
    border: none;
    border-top: 1px solid #f0f0f0;
    cursor: pointer;
    font-size: 0.7rem;
    color: #555;
    font-family: inherit;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: background 0.08s;
  }

  .flow-node-row:hover {
    background: #f0f4ff;
    color: #1a56cc;
  }

  .flow-node-row-arrow {
    color: var(--accent);
    flex-shrink: 0;
    font-size: 0.65rem;
  }

  /* -------------------------------------------------------------------------
     Shared type / name spans (used in both layouts)
  ------------------------------------------------------------------------- */

  .flow-node-type {
    font-size: 0.62rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--accent);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-shrink: 0;
  }

  .flow-node-name {
    font-weight: 500;
    color: #111;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>

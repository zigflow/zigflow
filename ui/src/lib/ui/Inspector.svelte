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
  import type { Node } from '$lib/tasks/model';

  // ---------------------------------------------------------------------------
  // Props
  // ---------------------------------------------------------------------------

  interface Props {
    node: Node | null;
    canMoveUp?: boolean;
    canMoveDown?: boolean;
    onmoveup?: () => void;
    onmovedown?: () => void;
    ondelete?: () => void;
    // Navigation into subgraphs
    onenternode?: (nodeId: string) => void; // loop body
    onenterbranch?: (nodeId: string, branchId: string) => void; // switch/fork/try
    // Branch management (switch / fork)
    onaddbranch?: (nodeId: string) => void;
    onremovebranch?: (nodeId: string, branchId: string) => void;
    // Try-specific: add catchGraph section
    onaddcatch?: (nodeId: string) => void;
  }

  let {
    node,
    canMoveUp = false,
    canMoveDown = false,
    onmoveup,
    onmovedown,
    ondelete,
    onenternode,
    onenterbranch,
    onaddbranch,
    onremovebranch,
    onaddcatch,
  }: Props = $props();

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  function configSummary(n: Node): string {
    if (n.type !== 'task') return '';
    switch (n.config.kind) {
      case 'set':
        return `Sets ${Object.keys(n.config.assignments).length} variable(s)`;
      case 'call-http':
        return `${n.config.method.toUpperCase()} ${n.config.endpoint}`;
      case 'call-grpc':
        return `${n.config.serviceName}/${n.config.method}`;
      case 'call-activity':
        return n.config.name;
      case 'run-container':
        return n.config.image;
      case 'run-script':
        return `${n.config.language} script`;
      case 'run-shell':
        return n.config.command;
      case 'run-workflow':
        return `${n.config.namespace}/${n.config.name}@${n.config.version}`;
      case 'wait':
        return `Wait ${JSON.stringify(n.config.duration)}`;
      case 'raise':
        return `HTTP ${n.config.errorStatus}`;
      case 'listen':
        return `${n.config.mode} of ${n.config.events.length} event(s)`;
    }
  }

  function structuralSummary(n: Node): string {
    switch (n.type) {
      case 'switch':
        return `${n.branches.length} branch(es)`;
      case 'fork':
        return `${n.branches.length} branch(es)${n.compete ? ' — compete' : ''}`;
      case 'try':
        return ['try', n.catchGraph ? 'catch' : null]
          .filter(Boolean)
          .join(' / ');
      case 'loop':
        return `for each in ${n.in}`;
      default:
        return '';
    }
  }

  // Minimum branch count: fork requires at least 2; switch requires at least 1.
  function minBranches(n: Node): number {
    return n.type === 'fork' ? 2 : 1;
  }
</script>

<aside class="inspector">
  {#if node === null}
    <p class="inspector-empty">Select a node to inspect it.</p>
  {:else}
    <header class="inspector-header">
      <span class="inspector-type">{node.type}</span>
      <h2 class="inspector-name">{node.name}</h2>
    </header>

    <section class="inspector-section">
      <dl>
        <dt>ID</dt>
        <dd class="mono">{node.id}</dd>

        {#if node.type === 'task'}
          <dt>Kind</dt>
          <dd>{node.config.kind}</dd>
          <dt>Detail</dt>
          <dd>{configSummary(node)}</dd>
          {#if node.if}
            <dt>Condition</dt>
            <dd class="mono">{node.if}</dd>
          {/if}
        {:else}
          <dt>Structure</dt>
          <dd>{structuralSummary(node)}</dd>
          {#if node.if}
            <dt>Condition</dt>
            <dd class="mono">{node.if}</dd>
          {/if}
        {/if}
      </dl>
    </section>

    <!-- -----------------------------------------------------------------------
      Structural node management: branches, sections, body navigation.
      Replaces the generic "coming soon" hint for all structural node types.
    ----------------------------------------------------------------------- -->

    {#if node.type === 'switch' || node.type === 'fork'}
      <section class="inspector-branches">
        <h3 class="inspector-section-title">Branches</h3>
        <ul class="branch-list" role="list">
          {#each node.branches as branch (branch.id)}
            <li class="branch-item">
              <button
                class="branch-label-btn"
                onclick={() => onenterbranch?.(node.id, branch.id)}
                type="button"
              >
                {branch.label}
              </button>
              {#if node.branches.length > minBranches(node)}
                <button
                  class="branch-remove-btn"
                  onclick={() => onremovebranch?.(node.id, branch.id)}
                  type="button"
                  aria-label="Remove branch {branch.label}"
                >
                  ✕
                </button>
              {/if}
            </li>
          {/each}
        </ul>
        <button
          class="branch-add-btn"
          onclick={() => onaddbranch?.(node.id)}
          type="button"
        >
          + Add branch
        </button>
      </section>
    {:else if node.type === 'try'}
      <section class="inspector-branches">
        <h3 class="inspector-section-title">Sections</h3>
        <div class="try-sections">
          <button
            class="try-section-btn"
            onclick={() => onenterbranch?.(node.id, 'tryGraph')}
            type="button"
          >
            Enter try body
          </button>
          {#if node.catchGraph !== undefined}
            <button
              class="try-section-btn"
              onclick={() => onenterbranch?.(node.id, 'catchGraph')}
              type="button"
            >
              Enter catch block
            </button>
          {:else}
            <button
              class="try-add-catch-btn"
              onclick={() => onaddcatch?.(node.id)}
              type="button"
            >
              + Add catch block
            </button>
          {/if}
        </div>
      </section>
    {:else if node.type === 'loop'}
      <section class="inspector-branches">
        <button
          class="loop-enter-btn"
          onclick={() => onenternode?.(node.id)}
          type="button"
        >
          Enter loop body
        </button>
      </section>
    {:else}
      <p class="inspector-hint">Full editing UI coming soon.</p>
    {/if}

    <div class="move-row">
      <button
        class="move-btn"
        disabled={!canMoveUp}
        onclick={() => onmoveup?.()}
        type="button"
        aria-label="Move task up"
      >
        ↑ Move up
      </button>
      <button
        class="move-btn"
        disabled={!canMoveDown}
        onclick={() => onmovedown?.()}
        type="button"
        aria-label="Move task down"
      >
        ↓ Move down
      </button>
    </div>

    <button class="delete-btn" onclick={() => ondelete?.()} type="button">
      Delete task
    </button>
  {/if}
</aside>

<style>
  .inspector {
    width: 260px;
    min-width: 260px;
    border-left: 1px solid #ddd;
    background: #fff;
    display: flex;
    flex-direction: column;
    padding: 1rem;
    overflow-y: auto;
    font-size: 0.875rem;
  }

  .inspector-empty {
    color: #888;
    font-style: italic;
    margin: 0;
  }

  .inspector-header {
    margin-bottom: 1rem;
  }

  .inspector-type {
    display: inline-block;
    background: #e8f0fe;
    color: #1a56cc;
    border-radius: 4px;
    padding: 1px 6px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.25rem;
  }

  .inspector-name {
    font-size: 1rem;
    font-weight: 600;
    margin: 0;
    word-break: break-all;
  }

  .inspector-section {
    flex: 1;
  }

  dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.25rem 0.75rem;
    margin: 0;
  }

  dt {
    color: #666;
    font-weight: 500;
    white-space: nowrap;
  }

  dd {
    margin: 0;
    word-break: break-all;
    color: #111;
  }

  .mono {
    font-family: monospace;
    font-size: 0.8em;
    color: #555;
  }

  .inspector-hint {
    margin-top: 1.5rem;
    font-size: 0.75rem;
    color: #999;
    font-style: italic;
  }

  /* -------------------------------------------------------------------------
     Branch / structural management
  ------------------------------------------------------------------------- */

  .inspector-branches {
    margin-top: 1rem;
    padding-top: 0.75rem;
    border-top: 1px solid #eee;
  }

  .inspector-section-title {
    margin: 0 0 0.5rem;
    font-size: 0.75rem;
    font-weight: 600;
    color: #444;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .branch-list {
    list-style: none;
    margin: 0 0 0.5rem;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .branch-item {
    display: flex;
    align-items: center;
    gap: 0.375rem;
  }

  .branch-label-btn {
    flex: 1;
    min-width: 0;
    padding: 0.25rem 0.5rem;
    background: #f0f4ff;
    border: 1px solid #c5d5f5;
    border-radius: 4px;
    font-size: 0.8rem;
    font-family: inherit;
    color: #1a56cc;
    cursor: pointer;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .branch-label-btn:hover {
    background: #e0eaff;
    border-color: #1a56cc;
  }

  .branch-remove-btn {
    padding: 0.2rem 0.375rem;
    background: transparent;
    border: 1px solid #e0a0a0;
    border-radius: 4px;
    color: #c0392b;
    font-size: 0.72rem;
    cursor: pointer;
    line-height: 1;
  }

  .branch-remove-btn:hover {
    background: #fff0f0;
  }

  .branch-add-btn {
    width: 100%;
    padding: 0.3rem 0.5rem;
    background: transparent;
    border: 1px dashed #aaa;
    border-radius: 4px;
    color: #555;
    font-size: 0.78rem;
    cursor: pointer;
    text-align: center;
  }

  .branch-add-btn:hover {
    border-color: #1a56cc;
    color: #1a56cc;
    background: #f0f4ff;
  }

  /* Try sections */

  .try-sections {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .try-section-btn {
    width: 100%;
    padding: 0.35rem 0.6rem;
    background: #f5f5f5;
    border: 1px solid #ddd;
    border-radius: 4px;
    color: #333;
    font-size: 0.8rem;
    cursor: pointer;
    text-align: left;
  }

  .try-section-btn:hover {
    background: #eee;
    border-color: #aaa;
  }

  .try-add-catch-btn {
    width: 100%;
    padding: 0.3rem 0.5rem;
    background: transparent;
    border: 1px dashed #aaa;
    border-radius: 4px;
    color: #555;
    font-size: 0.78rem;
    cursor: pointer;
    text-align: center;
  }

  .try-add-catch-btn:hover {
    border-color: #dc2626;
    color: #dc2626;
    background: #fff5f5;
  }

  /* Loop body */

  .loop-enter-btn {
    width: 100%;
    padding: 0.35rem 0.6rem;
    background: #f5f5f5;
    border: 1px solid #ddd;
    border-radius: 4px;
    color: #333;
    font-size: 0.8rem;
    cursor: pointer;
    text-align: left;
  }

  .loop-enter-btn:hover {
    background: #eee;
    border-color: #aaa;
  }

  /* -------------------------------------------------------------------------
     Move / delete controls
  ------------------------------------------------------------------------- */

  .move-row {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .move-btn {
    flex: 1;
    padding: 0.375rem 0.5rem;
    background: transparent;
    border: 1px solid #ddd;
    border-radius: 6px;
    color: #444;
    font-size: 0.8rem;
    cursor: pointer;
    transition:
      background 0.1s,
      border-color 0.1s;
  }

  .move-btn:hover:not(:disabled) {
    background: #f5f5f5;
    border-color: #aaa;
  }

  .move-btn:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .delete-btn {
    margin-top: 0.75rem;
    width: 100%;
    padding: 0.375rem 0.75rem;
    background: transparent;
    border: 1px solid #e0a0a0;
    border-radius: 6px;
    color: #c0392b;
    font-size: 0.8rem;
    cursor: pointer;
    transition:
      background 0.1s,
      border-color 0.1s;
  }

  .delete-btn:hover {
    background: #fff0f0;
    border-color: #c0392b;
  }
</style>

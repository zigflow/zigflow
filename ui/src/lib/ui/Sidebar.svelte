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
  const nodeTypes = [
    { type: 'input', label: 'Start', description: 'Workflow entry point' },
    { type: 'default', label: 'Task', description: 'Generic task node' },
    { type: 'output', label: 'End', description: 'Workflow exit point' },
  ];

  function onDragStart(event: DragEvent, nodeType: string) {
    if (!event.dataTransfer) return;

    event.dataTransfer.setData('application/svelteflow', nodeType);
    event.dataTransfer.effectAllowed = 'move';
  }
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <h2>Nodes</h2>
    <p class="subtitle">Drag and drop to add</p>
  </div>

  <div class="node-list">
    {#each nodeTypes as node (node.type)}
      <div
        class="node-item"
        role="button"
        tabindex="0"
        draggable="true"
        ondragstart={(e) => onDragStart(e, node.type)}
      >
        <div
          class="node-icon"
          class:input={node.type === 'input'}
          class:output={node.type === 'output'}
        >
          <span class="node-type"
            >{node.type === 'input'
              ? '▶'
              : node.type === 'output'
                ? '⏹'
                : '●'}</span
          >
        </div>
        <div class="node-info">
          <strong>{node.label}</strong>
          <span class="description">{node.description}</span>
        </div>
      </div>
    {/each}
  </div>
</aside>

<style lang="scss">
  @use '../../styles/tokens' as *;

  .sidebar {
    width: 280px;
    height: 100%;
    background-color: $color-bg;
    border-right: 1px solid $color-border;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  .sidebar-header {
    padding: $spacing-lg;
    border-bottom: 1px solid $color-border;

    h2 {
      margin: 0 0 $spacing-xs 0;
      font-size: $font-size-lg;
      font-weight: $font-weight-bold;
    }

    .subtitle {
      margin: 0;
      font-size: $font-size-sm;
      color: $color-text-muted;
    }
  }

  .node-list {
    padding: $spacing-md;
    display: flex;
    flex-direction: column;
    gap: $spacing-sm;
  }

  .node-item {
    display: flex;
    align-items: center;
    gap: $spacing-md;
    padding: $spacing-md;
    background-color: $color-bg;
    border: 2px solid $color-border;
    border-radius: $radius-md;
    cursor: grab;
    transition:
      border-color $transition-fast,
      box-shadow $transition-fast,
      transform $transition-fast;

    &:hover {
      border-color: $color-primary;
      box-shadow: $shadow-sm;
      transform: translateY(-2px);
    }

    &:active {
      cursor: grabbing;
      transform: translateY(0);
    }
  }

  .node-icon {
    width: 40px;
    height: 40px;
    border-radius: $radius-md;
    background-color: $color-bg-alt;
    border: 2px solid $color-border;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: $font-size-lg;
    flex-shrink: 0;
    transition: border-color $transition-fast;

    &.input {
      border-color: $color-success;
      color: $color-success;
    }

    &.output {
      border-color: $color-danger;
      color: $color-danger;
    }
  }

  .node-info {
    display: flex;
    flex-direction: column;
    gap: $spacing-xs;
    min-width: 0;

    strong {
      font-size: $font-size-base;
      font-weight: $font-weight-medium;
    }

    .description {
      font-size: $font-size-sm;
      color: $color-text-muted;
      line-height: $line-height-tight;
    }
  }
</style>

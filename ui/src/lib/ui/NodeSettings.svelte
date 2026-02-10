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
  import type { Node } from '$lib/types/flow';

  let { node, onClose }: { node: Node; onClose: () => void } = $props();
</script>

<aside class="node-settings">
  <div class="settings-header">
    <div class="header-content">
      <h2>Node Settings</h2>
      <p class="subtitle">Configure selected node</p>
    </div>
    <button
      class="close-button"
      onclick={onClose}
      aria-label="Close settings panel"
    >
      ×
    </button>
  </div>

  <div class="settings-content">
    <div class="setting-group">
      <div class="setting-label">Node ID</div>
      <div class="setting-value">{node.id}</div>
    </div>

    <div class="setting-group">
      <div class="setting-label">Node Type</div>
      <div class="setting-value">{node.type || 'default'}</div>
    </div>

    <div class="setting-group">
      <div class="setting-label">Label</div>
      <div class="setting-value">{node.data.label}</div>
    </div>

    <div class="setting-group">
      <div class="setting-label">Position</div>
      <div class="setting-value">
        x: {Math.round(node.position.x)}, y: {Math.round(node.position.y)}
      </div>
    </div>
  </div>

  <div class="settings-footer">
    <p class="info-text">Settings configuration coming soon</p>
  </div>
</aside>

<style lang="scss">
  @use '../../styles/tokens' as *;

  .node-settings {
    width: 320px;
    height: 100%;
    background-color: $color-bg;
    border-left: 1px solid $color-border;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  .settings-header {
    padding: $spacing-lg;
    border-bottom: 1px solid $color-border;
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: $spacing-md;

    .header-content {
      flex: 1;
      min-width: 0;
    }

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

  .close-button {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid $color-border;
    border-radius: $radius-sm;
    color: $color-text-muted;
    font-size: $font-size-2xl;
    line-height: 1;
    cursor: pointer;
    transition:
      background-color $transition-fast,
      border-color $transition-fast,
      color $transition-fast;

    &:hover {
      background-color: $color-bg-alt;
      border-color: $color-danger;
      color: $color-danger;
    }

    &:active {
      background-color: $color-danger;
      border-color: $color-danger;
      color: $color-bg;
    }
  }

  .settings-content {
    flex: 1;
    padding: $spacing-lg;
    display: flex;
    flex-direction: column;
    gap: $spacing-lg;
  }

  .setting-group {
    display: flex;
    flex-direction: column;
    gap: $spacing-xs;
  }

  .setting-label {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $color-text-muted;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .setting-value {
    padding: $spacing-md;
    background-color: $color-bg-alt;
    border: 1px solid $color-border;
    border-radius: $radius-sm;
    font-size: $font-size-base;
    font-family: $font-family-mono;
  }

  .settings-footer {
    padding: $spacing-lg;
    border-top: 1px solid $color-border;
    background-color: $color-bg-alt;

    .info-text {
      margin: 0;
      font-size: $font-size-sm;
      color: $color-text-muted;
      text-align: center;
      font-style: italic;
    }
  }
</style>

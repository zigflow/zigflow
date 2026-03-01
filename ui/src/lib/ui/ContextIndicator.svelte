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
  ContextIndicator — shows the current editing context above the canvas.

  Receives a pre-built `label` string from the page (which has full access to
  WorkflowFile and GraphPath and can produce type-accurate context strings).
  Examples: "Editing Workflow: my-workflow", "Editing Switch Branch: fast-path".

  Hidden when label is empty.
-->

<script lang="ts">
  interface Props {
    label: string;
  }

  let { label }: Props = $props();
</script>

{#if label}
  <div class="context-indicator" role="status" aria-live="polite">
    {label}
  </div>
{/if}

<style>
  .context-indicator {
    padding: 0.3rem 1rem;
    background: #f0f4ff;
    border-bottom: 1px solid #dde4f5;
    font-size: 0.78rem;
    font-weight: 500;
    color: #2a4580;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>

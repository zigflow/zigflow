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

  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
</script>

<div class="workflows-page">
  <h1>Workflows</h1>

  {#if data.workflowFiles.length > 0}
    <ul class="workflow-list">
      {#each data.workflowFiles as file (file)}
        <li>
          <a class="workflow-link" href={resolve(`/workflows/${file}`)}>
            {file.replace(/\.ya?ml$/, '')}
          </a>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="empty">No workflow files found.</p>
  {/if}

  <form method="POST" class="new-workflow-form">
    <input
      class="name-input"
      name="name"
      type="text"
      placeholder="workflow-name"
      required
    />
    <button class="new-btn" type="submit">New workflow</button>
  </form>
</div>

<style>
  .workflows-page {
    padding: 2rem;
    font-family: system-ui, sans-serif;
    max-width: 600px;
  }

  h1 {
    font-size: 1.5rem;
    margin-bottom: 1.5rem;
  }

  .workflow-list {
    list-style: none;
    padding: 0;
    margin: 0 0 2rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .workflow-link {
    display: block;
    padding: 0.5rem 0.75rem;
    background: #f5f5f5;
    border: 1px solid #ddd;
    border-radius: 6px;
    color: #111;
    text-decoration: none;
    font-size: 0.9rem;
  }

  .workflow-link:hover {
    background: #e8f0fe;
    border-color: #1a56cc;
    color: #1a56cc;
  }

  .empty {
    color: #888;
    font-style: italic;
    margin-bottom: 2rem;
  }

  .new-workflow-form {
    display: flex;
    gap: 0.5rem;
  }

  .name-input {
    flex: 1;
    padding: 0.4rem 0.65rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.9rem;
    font-family: inherit;
  }

  .name-input:focus {
    outline: none;
    border-color: #1a56cc;
  }

  .new-btn {
    padding: 0.4rem 0.9rem;
    background: #1a56cc;
    color: #fff;
    border: none;
    border-radius: 6px;
    font-size: 0.9rem;
    cursor: pointer;
    white-space: nowrap;
  }

  .new-btn:hover {
    background: #1344a8;
  }
</style>

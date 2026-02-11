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

  const props = $props();
  const { entries } = props.data;
</script>

<div class="workflows-list">
  <h1>Workflows</h1>

  {#if entries.length === 0}
    <p>Directory empty</p>
  {:else}
    <ul>
      {#each entries as entry (entry.path)}
        <li>
          <a href={resolve(`/workflows/${entry.path}`)}>
            {entry.name}{entry.isDirectory ? '/' : ''}
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .workflows-list {
    padding: 2rem;
  }

  h1 {
    margin-bottom: 1rem;
  }

  ul {
    list-style: none;
    padding: 0;
  }

  li {
    margin: 0.5rem 0;
  }

  a {
    color: var(--color-text);
    text-decoration: none;
  }

  a:hover {
    text-decoration: underline;
  }
</style>

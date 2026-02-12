# Task State Management Usage

This document explains how to use the task state management system with the
Serverless Workflow SDK.

## Overview

Each task type in Zigflow has a corresponding state model that combines:

- **Common properties** (metadata, export, output) - shared by all tasks
- **Task-specific properties** (e.g., `set` for SetTask) - unique to each task type

The state is managed using Svelte stores and validated using the Serverless
Workflow SDK classes.

## Basic Usage

### 1. Create a Task Instance

```typescript
import SetTask from '$lib/tasks/set';

const setTask = new SetTask();
```

### 2. Get Default State

```typescript
// Get the default state as a plain object
const defaultState = setTask.createDefaultState();
console.log(defaultState);
// Output:
// {
//   metadata: {},
//   export: {},
//   output: {},
//   set: {
//     hello: 'world'
//   }
// }
```

### 3. Create a State Store (for Svelte components)

```typescript
// Create a reactive Svelte store
const stateStore = setTask.createStateStore();

// In a Svelte component:
$stateStore.set.hello = 'universe';
```

### 4. Validate State

```typescript
const state = {
  metadata: {},
  export: {},
  output: {},
  set: {
    myVar: 'myValue',
  },
};

try {
  setTask.validate(state);
  console.log('✓ State is valid');
} catch (error) {
  console.error('✗ Validation failed:', error.message);
}
```

### 5. Create SDK Instance

```typescript
// Create an SDK class instance from state
const sdkInstance = setTask.createSDKInstance(state);

// SDK instances have validate() and normalize() methods
sdkInstance.validate();

// Convert to JSON for YAML generation
const json = JSON.stringify(sdkInstance, null, 2);
```

## Example: NodeSettings Component

Here's how you might use this in a NodeSettings component:

```svelte
<script lang="ts">
  import SetTask from '$lib/tasks/set';
  import type { Node } from '@xyflow/svelte';

  let { node, onClose }: { node: Node; onClose: () => void } = $props();

  // Get the task instance based on node type
  const task = new SetTask();

  // Create or load state store
  const stateStore = task.createStateStore();

  // Handle save
  function handleSave() {
    try {
      // Validate before saving
      task.validate($stateStore);

      // Create SDK instance
      const sdkInstance = task.createSDKInstance($stateStore);

      // Save to node data
      node.data.taskState = sdkInstance;

      console.log('✓ Saved successfully');
      onClose();
    } catch (error) {
      console.error('✗ Validation failed:', error.message);
    }
  }
</script>

<div class="settings">
  <h2>Set Task Settings</h2>

  <!-- Common properties -->
  <div class="section">
    <h3>Metadata</h3>
    <input type="text" bind:value={$stateStore.metadata.description} />
  </div>

  <!-- Task-specific properties -->
  <div class="section">
    <h3>Variables</h3>
    <input type="text" bind:value={$stateStore.set.hello} />
  </div>

  <button onclick={handleSave}>Save</button>
</div>
```

## State Structure

The state is a flat object that merges common and task-specific properties:

```typescript
type TaskState = {
  // Common properties (all tasks)
  metadata?: Record<string, unknown>;
  export?: {
    as?: Record<string, unknown>;
  };
  output?: {
    as?: string;
  };

  // Task-specific properties (e.g., for SetTask)
  set?: Record<string, unknown>;

  // Other task-specific properties...
  // wait?: { seconds: number };
  // call?: 'http' | 'grpc' | 'activity';
  // etc.
};
```

## Available Methods

Each task class extends the base `Task` class and provides:

| Method                     | Description                                       |
| -------------------------- | ------------------------------------------------- |
| `getSDKClass()`            | Get the SDK class constructor                     |
| `getDefaultSpecificData()` | Get default task-specific properties              |
| `getDefaultCommonData()`   | Get default common properties                     |
| `createDefaultState()`     | Create a complete default state object            |
| `createStateStore()`       | Create a Svelte writable store with default state |
| `createSDKInstance(state)` | Create an SDK class instance from state           |
| `validate(state)`          | Validate state using SDK validation               |

## TypeScript Types

The system is fully typed:

```typescript
// Generic task state type
type TaskState = Record<string, unknown>;

// Task class is generic over the SDK class type
class SetTask extends Task<InstanceType<typeof sdk.Classes.SetTask>> {
  // ...
}
```

This ensures type safety when working with task states and SDK instances.

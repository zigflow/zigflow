---
sidebar_position: 5
description: "How data moves through a Zigflow workflow: task results, output.as, export.as, $context and the workflow return value."
---

# Data Flow

## What you will learn

- How a task's result is exposed to later tasks via `$output`
- What `output.as` does and how it differs from `export.as`
- How `$context` is written and preserved across tasks
- What a workflow returns to its caller
- How to choose between `export` and `output`

:::tip
This page builds on the variables and expression syntax in
[Data and Expressions](/docs/concepts/data-and-expressions). Read that first
if `${ }` and `$context` are new to you.
:::

---

## Two channels for data

Zigflow moves data through a workflow on two separate channels. Almost every
mistake comes from confusing them.

| Channel | Variable | Lifetime | Shaped by |
| --- | --- | --- | --- |
| The result of the most recent task | `$output` | Overwritten by every task | `output.as` |
| Shared data carried across tasks | `$context` | Persists until you replace it | `export.as` |

Think of `$output` as a baton passed hand to hand: only the latest runner holds
it. Think of `$context` as a shared whiteboard. Each `export` rewrites the
whiteboard unless you explicitly merge the existing contents.

The workflow returns the baton. The whiteboard is for passing values between
tasks along the way.

---

## How a task processes data

When a task runs, Zigflow performs these steps in order:

1. The task executes and produces a **raw result**.
2. `output.as` is evaluated against that raw result and the value is stored in
   `$output`. With no `output.as`, the raw result becomes `$output` unchanged.
3. `export.as` is evaluated against the same raw result and the value
   **replaces** `$context`. With no `export.as`, `$context` is left alone.

---

:::warning
Both `output.as` and `export.as` see the task's **raw result** as `.`, not each
other's result. `output.as` runs first, so inside `export.as` the `$output`
variable already holds the shaped output, but `.` is still the raw result.
:::

## Shaping the baton with `output.as`

`output.as` transforms a task's result before it becomes `$output`. Use it to
keep only the fields you need.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: shape-output
  version: 0.0.1
do:
  - getUser:
      output:
        as:
          id: ${ .id }
          displayName: ${ .name }
      set:
        id: 7
        name: Ziggy
```

The `set` task produces `{ id: 7, name: Ziggy }`. `output.as` reshapes it, so
`$output` becomes `{ id: 7, displayName: Ziggy }`.

---

## Writing to the whiteboard with `export.as`

`export.as` writes to `$context` so later tasks can read the value. The result
of the expression becomes the **entire** new context.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: export-for-later
  version: 0.0.1
do:
  - fetchUser:
      export:
        as: '${ $context + { userId: .id } }'
      set:
        id: 42
        name: Ziggy
  - greet:
      set:
        message: ${ "Hello user " + ($context.userId | tostring) }
```

`fetchUser` exports `userId` into `$context`. The `greet` task reads it back
with `$context.userId`, even though it ran later.

:::warning
`export.as` replaces `$context` wholesale. It does not merge automatically.
Writing `export: { as: ${ . } }` discards everything already in the context.
:::

---

## Accumulating context

Because `export` replaces the context, you accumulate values by merging the
existing context with the new data, using `${ $context + { ... } }`.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: accumulate-context
  version: 0.0.1
do:
  - first:
      export:
        as: '${ $context + { first: .value } }'
      set:
        value: 1
  - second:
      export:
        as: '${ $context + { second: .value } }'
      set:
        value: 2
  - report:
      output:
        as: ${ $context }
      set:
        done: true
```

After `first`, the context is `{ first: 1 }`. After `second`, it is
`{ first: 1, second: 2 }`. The `report` task copies the whole context into
`$output` so it becomes the workflow result.

---

## The workflow return value

A workflow returns the value of `$output` after its final task. In practice:

- The workflow return value is the output of the final task.
- To shape what the workflow returns, set `output.as` on the final task.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: finalise-return
  version: 0.0.1
do:
  - finalise:
      output:
        as: ${ $context }
      set:
        done: true
```

:::info
Top-level `output.schema` describes the expected workflow output shape. To shape
the returned value, use `output.as` on the final task.
:::

---

## `export` or `output`?

This is the most common point of confusion. Use this rule:

- Use **`output.as`** when you want to shape the result that flows to the next
  task or that the workflow returns.
- Use **`export.as`** when you want to keep a value available to tasks that run
  later, regardless of what the next task returns.

### Incorrect: using `output` to pass a value to a later task

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: incorrect-output
  version: 0.0.1
do:
  - fetchUser:
      output:
        as: '${ { userId: .id } }'
      set:
        id: 42
  - doWork:
      set:
        note: working
  - useUser:
      # Wrong: $output is now the result of doWork, not fetchUser
      set:
        message: ${ $output.userId }
```

`$output` only holds the most recent task's result, so by the time `useUser`
runs the user id is gone.

### Correct: export the value into `$context`

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: correct-export
  version: 0.0.1
do:
  - fetchUser:
      export:
        as: '${ $context + { userId: .id } }'
      set:
        id: 42
  - doWork:
      set:
        note: working
  - useUser:
      set:
        message: ${ "user " + ($context.userId | tostring) }
```

`$context` persists across `doWork`, so `useUser` can still read `userId`.

---

## Common mistakes

:::warning
**Expecting `export` to become the workflow return value.** The workflow returns
`$output`, not `$context`. Copy the context into the output on the final task
with `output: { as: ${ $context } }` if you want to return it.
:::

:::warning
**Expecting `export` to merge.** Each `export.as` replaces the whole context.
Use `${ $context + { ... } }` to add to it without losing earlier values.
:::

:::warning
**Reading `$output` after intervening tasks.** `$output` is only the most recent
task's result. To use a value several steps later, export it to `$context`.
:::

:::warning
**A routing `switch` keeps the previous `$output`.** A `switch` that only chooses
a branch does not produce its own result, so `$output` still holds the value
from the task before it. Add `output.as` to the `switch` if you need to change
it.
:::

---

## Related pages

- [Data and Expressions](/docs/concepts/data-and-expressions): jq expressions
- [Set task](/docs/dsl/tasks/set): storing data in `$data`
- [DSL reference](/docs/dsl/intro): `input`, `output` and `export` properties

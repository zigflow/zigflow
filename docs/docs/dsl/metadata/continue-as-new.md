# Continue-As-New

Continue-As-New allows you to checkpoint your Workflow's state and start a fresh
Workflow.

For more information, see the [Temporal documentation](https://docs.temporal.io/workflow-execution/continue-as-new).

## Location

* Document

## Metadata

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| `canMaxHistoryLength` | `integer` | `no` | Allows you to test the Continue-As-New functionality by specifying the max history length before triggering. |

## When it happens

Zigflow checks whether to Continue-As-New before it starts each top-level task,
and again before the next task once the current one has finished. When Temporal
suggests continuation (or `canMaxHistoryLength` is exceeded) the workflow starts
a fresh run and resumes at the task it stopped before. Completed tasks are
skipped on the new run.

:::info
Continue-As-New is always performed by the registered workflow that owns the
execution, using that workflow's own type. Structural tasks such as
[`for`](/docs/dsl/tasks/for), [`fork`](/docs/dsl/tasks/fork) and
[`try`](/docs/dsl/tasks/try) run inline within that workflow (see
[How Zigflow runs](/docs/concepts/how-zigflow-runs)), so they never start their
own Continue-As-New run.
:::

## Structural tasks

Because `for`, `fork` and `try` run inline, their bodies share the owning
workflow's history. Zigflow checks for Continue-As-New at safe checkpoints
around them:

- **Before** a structural task begins. If continuation is already suggested the
  workflow continues as new before any iteration, branch or body has run, and
  the fresh run restarts the structural task from the beginning.
- **After** the structural task completes, as part of the check before the next
  task.

These checkpoints keep continuation safe: the resume point is always a whole
top-level task that has either not started or fully finished, so no iteration,
branch or body is ever half-completed across a run boundary.

:::warning
Continue-As-New does not happen part way through a `for` loop, a `fork` or a
`try` body. A single structural task must fit within one workflow run. If a loop
performs enough work to approach Temporal's history limits within a single run,
split the work into a top-level task per stage, or process the collection in
smaller batches, so a checkpoint can fall between top-level tasks.
:::

## Common mistakes

**Expecting a long `for` loop to checkpoint between iterations.** It does not.
Continuation is evaluated before the loop starts and after it finishes, not
between iterations.

**Assuming a structural body starts its own workflow.** Only the document root
and an explicit [`run.workflow`](/docs/dsl/tasks/run#workflow) target are
separate Temporal workflow executions.

## Related pages

- [How Zigflow runs](/docs/concepts/how-zigflow-runs)
- [For](/docs/dsl/tasks/for)
- [Fork](/docs/dsl/tasks/fork)
- [Try](/docs/dsl/tasks/try)

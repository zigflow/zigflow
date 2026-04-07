# Do

The Do task is specified in every single workflow as it represents the basic
definition of the workflow. In Temporal, a workflow is a sequence of steps that
must follow deterministic constraints:

:::info[Determinism]
The same steps are executed, in the same order, with the same
input and output data
:::

The Do task is used by many other tasks, such as in a [child workflow](https://docs.temporal.io/child-workflows).
These will be discussed in detail in those tasks.

## When to use this

Use Do at the top level to define one or more named workflows.
Use it nested inside Fork, For and Try to define a sequential
sequence of steps within those tasks.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| do | [`map[string, task]`](/docs/dsl/tasks/intro) | `no` | The tasks to perform sequentially. |

## Examples

### Single Workflow

:::tip
If you're not sure how to start building your workflow, start with a single
workflow definition.
:::

When defined at the root level, if a single is specified then a single Temporal
workflow is registered, with the workflow name set by the
`document.workflowType` property.
In this example, the workflow name is `single-workflow`:

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: single-workflow
  version: 0.0.1
do:
  - setData:
      set:
        key: ${ uuid }
  - wait:
      wait:
        seconds: 5
  - getUser:
      output:
        as:
          user: ${ . }
      call: http
      with:
        method: get
        headers:
          uuid: ${ $data.key }
        endpoint: https://jsonplaceholder.typicode.com/users/2
```

This workflow is very basic, but explain the steps in details:
1. **setData**: this sets some data, which is randomly generated UUID key. This
   is available on `${ $data.key }`. See [Set](/docs/dsl/tasks/set) for a
   detailed explanation of why generated data should be used sparingly.
2. **wait**: pause the workflow, using a [Durable Timer](https://docs.temporal.io/workflow-execution/timers-delays)
3. **getUser**: call an HTTP endpoint, using the previously generated UUID as a
   header.

### Multiple Workflows

:::tip
Workflow names must be unique.
:::

When defined at the root level, you can use the Do task to create multiple
workflows by specifying multiple Do tasks at the top level. If you do this, the
workflow names will be set by the task name and the `document.workflowType` property
will be ignored.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: multiple-workflows # This property is ignored
  version: 0.0.1
do:
  - workflow1:
      do:
        - wait:
            wait:
              seconds: 5
        - getUser:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/users/1
  - workflow2:
      do:
        - wait:
            wait:
              seconds: 10
        - getUser:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/users/2
```

These workflow are basic, with a sleep and an HTTP call. These two workflows are
independent and are named `workflow1` and `workflow2`.

## Gotchas

**`document.workflowType` is ignored when multiple Do tasks are defined at the root
level.** Workflow type names are taken from the task names (`workflow1`,
`workflow2`) instead.

**Each task in `do` must have a unique name within its scope.** Duplicate task
names cause a validation error.

## Related pages

- [Fork](/docs/dsl/tasks/fork): run tasks in parallel rather than in sequence
- [Try](/docs/dsl/tasks/try): error handling within a sequence
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs): execution
  model

# Set

A task used to set data to the workflow's state.

## When to use this

Use Set whenever you need to write data into the workflow state.
**Always use Set for generated values** (UUIDs, timestamps) to
ensure determinism during Temporal replay.

:::warning
A Temporal workflow **MUST** be [deterministic](https://docs.temporal.io/workflow-definition#deterministic-constraints).

You are strongly advised to use the Set task when setting data, especially generated
data (such as `${ uuid }`), rather than invoking it in a task. For example, the
following [CallHTTP](/docs/dsl/tasks/call#http) task would return a
Non-Determinism Error (NDE)
if it was replayed:

```yaml
# Bad ❌
- updateUser:
    call: http
    output:
      as: '${ { response: . } }'
    with:
      method: put
      headers:
        content-type: application/json
      endpoint: https://echo.free.beeceptor.com
      body:
        id: ${ uuid } # This value is different every time this task is run
        hello: world

```

However, as this one generates the UUID in a Set task, [this is wrapped](https://docs.temporal.io/develop/go/side-effects)
in such a way that it's saved to the Temporal state and is safe to replay.

```yaml
- set:
    export:
      as:
        data: ${ . }
    set:
      id: ${ uuid }
- updateUser:
    call: http
    output:
      as: '${ $context + { response: . } }'
    with:
      method: put
      headers:
        content-type: application/json
      endpoint: https://echo.free.beeceptor.com
      body:
        id: ${ $data.id }
        hello: world

```

**With great power comes great responsibility**
:::

## Properties

| Name | Type | Required | Description |
| :--- | :---: | :---: | :--- |
| set | `map` | `yes` | The data to set. |

## Example

The data is saved to a `state` object and can be retrieved in a later task by
calling `${ $data.<key> }`. Once set, it remains in the state and can be overidden
by a later Set task.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - baseData:
      # Set the output to the context
      export:
        as: ${ . }
      set:
        # This value will be overidden later
        progress: 0
        # Set a variable from an envvar
        envvar: ${ $env.EXAMPLE_ENVVAR }
        # Generate a UUID
        uuid: ${ uuid }
        # Insert something from the input
        inputUserId: ${ $input.userId }
        # Maps can be used
        object:
          hello: world
          uuid: ${ uuid }
        # As can arrays
        array:
          - ${ uuid }
          - hello: world
  - updateProgress:
      # Merge this set with the context and output everything together
      output:
        as: ${ $context + . }
      set:
        # Overidden from above. Everything else remains the same
        progress: 100
```

## Gotchas

**Generated values (UUID, timestamps) used outside a `set` task cause
Non-Determinism Errors on replay.** See the warning at the top of this page.

**`set` data persists for the lifetime of the workflow.** A later `set` task
with the same key overwrites the value; all other keys remain.

## Related pages

- [Concepts: data and expressions](/docs/concepts/data-and-expressions):
  expression syntax and variable reference
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs):
  determinism and replay
- [Examples: hello world](/docs/examples/hello-world):
  set task in action

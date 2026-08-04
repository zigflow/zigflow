# Try

Serves as a mechanism within workflows to handle errors gracefully, potentially
retrying failed tasks before proceeding with alternate ones.

## When to use this

Use Try when a task may fail and you want to recover without
failing the entire workflow. Common uses include optional HTTP
calls and handling external service errors.

## Properties {/*#try-properties*/}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| try | [`map[string, task]`](/docs/dsl/tasks/intro) | `yes` | The task(s) to perform inline in the parent workflow. |
| catch | [`catch`](#catch) | `yes` | Configures the errors to catch and how to handle them. |

## Example

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - user:
      try:
        - getUser:
            call: http
            output:
              as:
                user: ${ . }
            with:
              method: get
              # This URL returns a 404
              endpoint: https://jsonplaceholder.typicode.com/users/2000
      catch:
        do:
          - setError:
              output:
                as:
                  error: ${ . }
              set:
                message: some error
```

This outputs:

```json
{
  "user": {
    "error": {
      "message": "some error"
    }
  }
}
```

## Definitions

### Catch

Defines the configuration of a catch clause used to handle errors.

#### Properties {/*#catch-properties*/}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| do | [`map[string, task]`](/docs/dsl/tasks/intro) | `yes` | The task(s) to run inline in the parent workflow when catching an error. |
| as | `string` | `no` | The key under `$data` where the caught error is stored. Defaults to `error`. |

## Gotchas

**The `catch` block catches all errors.** There is no filtering by error type.
To handle different error types differently, inspect the error object inside
the `catch` block.

**The `try` and `catch` blocks run inline in the parent workflow.** Their
commands contribute to the parent's Temporal history. They do not have
independent child workflow histories or retry policies. Zigflow does not
attempt Continue-As-New while it is inside either block. Inner tasks still emit
task lifecycle events, but the blocks do not emit `workflow.started` or
`workflow.completed` events.

**Inner tasks keep their own retry settings.** An activity inside `try` uses its
own activity options and document defaults. The `catch` block runs only after
those retries are exhausted.

**Both blocks use isolated state.** The `try` and `catch` blocks each start from
a fresh clone of the parent state. Partial state changes from a failed `try`
block are not visible in `catch`. Zigflow adds the caught error to the catch
clone, under `$data.error` by default. Inner state changes do not leak directly
to the parent. The Try task's result, `output` and `export` control what the
parent receives.

**`then: end` is not a caught error.** An end directive in `try` bypasses
`catch` and terminates the enclosing workflow. An end directive in `catch`
propagates in the same way.

## Related pages

- [Raise](/docs/dsl/tasks/raise): raising explicit errors
- [Call](/docs/dsl/tasks/call): HTTP and activity calls that may fail
- [Concepts: error handling and retries](/docs/concepts/error-handling-and-retries):
  error model overview
- [Examples: error handling](/docs/examples/error-handling):
  try/catch walkthrough

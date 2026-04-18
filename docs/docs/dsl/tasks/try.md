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
| try | [`map[string, task]`](/docs/dsl/tasks/intro) | `yes` | The task(s) to perform. This will be run as a [child workflow](https://docs.temporal.io/child-workflows). |
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

Defines the configuration of a catch clause, which a concept used to catch
errors.

#### Properties {/*#catch-properties*/}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| do | [`map[string, task]`](/docs/dsl/tasks/intro) | `yes` | The definition of the task(s) to run when catching an error. This will be run as a [child workflow](https://docs.temporal.io/child-workflows). |

## Gotchas

**The `catch` block catches all errors.** There is no filtering by error type.
To handle different error types differently, inspect the error object inside
the `catch` block.

**The `try` block runs as a child workflow.** Its history and retries are
independent from the parent workflow. The retry policy configured on inner
tasks still applies before the `catch` block runs.

## Related pages

- [Raise](/docs/dsl/tasks/raise): raising explicit errors
- [Call](/docs/dsl/tasks/call): HTTP and activity calls that may fail
- [Concepts: error handling and retries](/docs/concepts/error-handling-and-retries):
  error model overview
- [Examples: error handling](/docs/examples/error-handling):
  try/catch walkthrough

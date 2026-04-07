---
sidebar_position: 4
---

# Error Handling

Catch a failing HTTP call and continue with a fallback path instead
of failing the whole workflow.

## What you will learn

- How to use `try`/`catch` to intercept task failures
- How to execute a recovery path when an error occurs
- How to raise a structured error explicitly using `raise`

## Workflow

This workflow calls an endpoint that returns a 404. The `catch`
block intercepts the error and sets a fallback value.

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: try-catch
  version: 0.0.1
do:
  - user:
      try:
        - getUser:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/users/2000
      catch:
        do:
          - setError:
              set:
                err: some error
```

## Explanation

| Part | Purpose |
| --- | --- |
| `try` | Runs inner tasks as a child workflow |
| `catch.do` | Runs if any task inside `try` fails |
| `setError` | Sets a fallback value when an error is caught |

**The `catch` block catches all errors.** There is no DSL-level
filtering by error type. To handle different errors differently,
inspect the error object inside the `catch` block.

**The `try` block runs as a child workflow.** Inner tasks are
retried according to their configured retry policy before the
`catch` block is entered.

## Raising errors explicitly

Use the `raise` task to fail a workflow with a structured error:

```yaml
- validate:
    raise:
      error:
        type: >-
          https://serverlessworkflow.io/spec/1.0.0/errors/validation
        status: 400
        title: Missing required field
        detail: ${ "userId is required, got: " + ($input.userId | tostring) }
```

## How to run

1. Start a Temporal development server:

   ```sh
   temporal server start-dev
   ```

2. Start the worker:

   ```sh
   zigflow run -f workflow.yaml
   ```

3. Trigger the workflow:

   ```sh
   temporal workflow start \
     --type try-catch \
     --task-queue zigflow \
     --workflow-id catch-1
   ```

4. View the result:

   ```sh
   temporal workflow show --workflow-id catch-1
   ```

## Expected output

```json
{
  "err": "some error"
}
```

## Common mistakes

**Not all retries are exhausted before `catch` runs.**
The default retry policy applies inside `try`. The `catch` block
only runs after all retries are exhausted.

---

## Related pages

- [Try](/docs/dsl/tasks/try): `try`/`catch` reference
- [Raise](/docs/dsl/tasks/raise): raising explicit errors
- [Concepts: error handling](/docs/concepts/error-handling-and-retries):
  retry policy and error model

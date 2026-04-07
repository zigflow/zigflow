---
sidebar_position: 3
---

# HTTP Call

Fetch data from an external HTTP API and store the response in
workflow state.

## What you will learn

- How to make an HTTP GET request from a workflow using `call: http`
- How to capture and shape the HTTP response body
- How to pass dynamic values to an endpoint using runtime expressions

## Workflow

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: fetch-user
  version: 0.0.1
do:
  - getUser:
      call: http
      output:
        as:
          user: ${ . }
      with:
        method: get
        endpoint: https://jsonplaceholder.typicode.com/users/1
```

## Explanation

The `call: http` task runs as a Temporal activity. It makes the
request, then returns the response body as the task output.
`output.as` maps that body into `{ "user": ... }`.

**Retry behaviour.** Failed HTTP calls (non-2xx responses) are
retried using the workflow's retry policy. Wrap the call in a
[`try` task](/docs/dsl/tasks/try) to intercept specific errors.

## Passing dynamic values to the endpoint

Use a runtime expression in `endpoint`:

```yaml
with:
  method: get
  endpoint: >-
    ${ "https://jsonplaceholder.typicode.com/users/"
    + ($input.userId | tostring) }
```

The `$input` variable holds the data passed when the workflow was
triggered.

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
     --type fetch-user \
     --task-queue zigflow \
     --workflow-id fetch-1
   ```

4. View the result:

   ```sh
   temporal workflow show --workflow-id fetch-1
   ```

## Expected output

```json
{
  "user": {
    "id": 1,
    "name": "Leanne Graham"
  }
}
```

## Common mistakes

**The call succeeds locally but fails in production.**
Ensure the Zigflow worker process has network access to the
target endpoint.

**Non-2xx responses fail the workflow.**
HTTP errors raise by default. To recover, wrap the call in a
`try` task. See [Error Handling](/docs/examples/error-handling).

---

## Related pages

- [Call](/docs/dsl/tasks/call): full `call: http` reference
- [Error Handling](/docs/examples/error-handling): handling HTTP errors
- [Concepts: error handling](/docs/concepts/error-handling-and-retries):
  retry policy configuration

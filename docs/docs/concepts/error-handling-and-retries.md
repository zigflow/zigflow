---
sidebar_position: 5
---

# Error Handling and Retries

## What you will learn

- How Zigflow handles activity failures
- The default retry policy
- How to configure retries
- The `try` and `catch` pattern
- The `raise` task
- Current limitations

---

## Activity retries

Activities in Zigflow (HTTP calls, external activity calls, container runs)
are retried automatically by Temporal when they fail.

The default retry policy is:

| Setting | Default |
| --- | --- |
| Initial interval | 1 second |
| Backoff coefficient | 2.0 |
| Maximum interval | 1 minute |
| Maximum attempts | 5 |

A failing activity retries at 1s, 2s, 4s, 8s and 16s (capped at 60s) before
giving up.

---

## Configuring retries

Override the retry policy for a task using `metadata.activityOptions.retryPolicy`.
Durations are expressed as objects with `seconds`, `minutes`, `hours` or `days`
keys:

```yaml
- callApi:
    metadata:
      activityOptions:
        retryPolicy:
          initialInterval:
            seconds: 5
          backoffCoefficient: 1.5
          maximumInterval:
            seconds: 30
          maximumAttempts: 3
          nonRetryableErrorTypes:
            - validation-error
    call: http
    with:
      method: post
      endpoint: https://api.example.com/process
```

`nonRetryableErrorTypes` accepts a list of error type strings. Tasks that fail
with those error types are not retried.

Set `maximumAttempts: 1` to disable retries entirely for a task.

---

## The try/catch pattern

Use the `try` task to catch failures and handle them gracefully.

The `try` block runs as a child workflow. If any task inside it fails, the
`catch` block runs instead.

```yaml
- fetchUser:
    try:
      - getUser:
          call: http
          output:
            as:
              user: ${ . }
          with:
            method: get
            endpoint: https://api.example.com/users/999
    catch:
      do:
        - handleMissing:
            output:
              as:
                error: ${ . }
            set:
              message: User not found
```

The `catch` block receives the error as its input. The output of the `try`
task is whatever the `catch` block returns.

---

## The raise task

Use the `raise` task to throw an explicit error and stop the workflow.

Errors follow the [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807)
Problem Details format.

```yaml
- checkPermission:
    switch:
      - denied:
          when: ${ $data.role != "admin" }
          then: rejectRequest
      - default:
          then: processRequest

- rejectRequest:
    raise:
      error:
        type: https://serverlessworkflow.io/spec/1.0.0/errors/authorization
        status: 403
        title: Forbidden
        detail: Only admin users can perform this action
```

Standard error types from the Serverless Workflow specification:

| Type | Status |
| --- | --- |
| `https://serverlessworkflow.io/spec/1.0.0/errors/configuration` | 400 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/validation` | 400 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/expression` | 400 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/authentication` | 401 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/authorization` | 403 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/timeout` | 408 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/communication` | 500 |
| `https://serverlessworkflow.io/spec/1.0.0/errors/runtime` | 500 |

The status shown is the recommended default. Use the HTTP status code that
best describes the error.

---

## Current limitations

- The `catch` block cannot filter by error type. It catches all errors from
  the `try` block.
- There is no `finally` equivalent. Clean-up logic must go after the `try`
  task in the main `do` list.

---

## Common mistakes

**Expecting retries to continue after the maximum attempt count.**
Once `maximumAttempts` is exhausted, the activity fails permanently. The
error propagates up. Wrap the task in a `try` block to handle this case.

**Using `raise` inside a `try` block and expecting the catch to handle it.**
A `raise` inside the `try` block is caught by the `catch` block. Use this
intentionally only if you want to normalise errors to a consistent format.

**Not setting `startToCloseTimeout` for long-running activities.**
The default start-to-close timeout is 15 seconds. Long-running activities
(such as container executions or waiting on external systems) should increase
this via `metadata.activityOptions.startToCloseTimeout`.

---

## Related pages

- [Try task](/docs/dsl/tasks/try): full reference
- [Raise task](/docs/dsl/tasks/raise): full reference
- [Activity options](/docs/dsl/metadata/activity-options): retry policy reference
- [How Zigflow runs](/docs/concepts/how-zigflow-runs): execution model

---
sidebar_position: 4
---

# Data and Expressions

## What you will learn

- The expression syntax
- The variables available in expressions
- Built-in functions

:::tip
For how task results, `output.as`, `export.as` and `$context` move data through
a workflow, see [Data Flow](/docs/concepts/data-flow).
:::

---

## Expressions

Zigflow uses a jq-style expression syntax wrapped in `${ }`.

```yaml
message: ${ $input.name }
```

The expression inside `${ }` is evaluated as a jq filter against the current
input. The result replaces the expression value.

Expressions can appear in most YAML values: task properties, output
transformations, conditions and export statements.

Bare strings are treated as literal values:

```yaml
message: hello  # literal string "hello"
```

Expressions with string interpolation require quotes:

```yaml
endpoint: ${ "https://api.example.com/users/" + ($data.userId | tostring) }
```

---

## Variables

| Variable | Description |
| --- | --- |
| `$context` | The current workflow context, written by `export` calls |
| `$data` | Internal workflow and execution state maintained by Zigflow, including workflow and activity metadata. Use `$context` for values exported by workflow tasks |
| `$env` | Environment variables available to the worker |
| `$input` | The original workflow input supplied by the caller. This value does not change as tasks execute. |
| `$output` | The output of the most recent task |

### `$input`

The data passed in when the workflow execution was started. It is set once
and does not change.

```yaml
- greet:
    set:
      name: ${ $input.userName }
```

### `$data`

The workflow's accumulating data store. It persists for the whole workflow run
and is never reset between tasks. Writes merge into it by top-level key, so a new
write overwrites a key with the same name but leaves other keys untouched.

Two things write to `$data`:

- A `set` task merges its keys directly. `set: { userId: ... }` becomes
  `$data.userId`.
- A task that runs an activity stores its result under the task name. An `http`
  call named `fetchUser` becomes `$data.fetchUser`, holding the response body.

Workflow and activity metadata is also exposed here, under `$data.workflow` and
`$data.activity`.

```yaml
- fetchUser:
    call: http
    with:
      method: get
      endpoint: https://api.example.com/users/42
- greet:
    set:
      message: ${ "Hello " + $data.fetchUser.name }
```

Here `fetchUser` runs an HTTP call. Its response is stored at `$data.fetchUser`,
which the later `greet` task reads by task name.

`$data` differs from the other variables in scope. `$context` is replaced
wholesale by each `export` call, whereas `$data` accumulates. `$output` holds
only the most recent task's result, whereas `$data` keeps every task's result
keyed by name.

### `$context`

The current workflow context, written by `export` calls. Each `export`
replaces `$context` unless you explicitly merge into it. Use it to carry
structured data forward without polluting `$data`.

```yaml
- step1:
    export:
      as: '${ $context + { step1Result: . } }'
    set:
      foo: bar
```

### `$env`

Environment variables available to the worker process. Variables prefixed
with `ZIGGY_` (default prefix) are loaded from the environment and accessible
without the prefix.

```yaml
- readConfig:
    set:
      apiUrl: ${ $env.API_BASE_URL }
```

Set `--env-prefix` to change the prefix (default is `ZIGGY`).

### `$output`

The output of the most recently completed task. Use it to chain task results
without storing them explicitly.

---

## Built-in functions

These functions are available inside `${ }` expressions:

| Function | Returns | Notes |
| --- | --- | --- |
| `uuid` | Random UUID v4 | Must be used inside a `set` task for determinism |
| `timestamp` | Unix timestamp (integer) | Must be used inside a `set` task for determinism |
| `timestamp_iso8601` | ISO 8601 timestamp string | Must be used inside a `set` task for determinism |

### Why must generated values be in a `set` task?

Temporal replays workflow history. If you generate a UUID inside an HTTP
call body, a different UUID is generated on replay. Temporal detects this
as non-determinism and raises an error.

The `set` task wraps generated values in a Temporal side-effect, which
records the value in the history so it is stable across replays.

```yaml
# Bad: UUID differs on replay
- callApi:
    call: http
    with:
      method: post
      body:
        requestId: ${ uuid }  # Different value on replay

# Good: UUID is stable across replays
- generateId:
    set:
      requestId: ${ uuid }
- callApi:
    call: http
    with:
      method: post
      body:
        requestId: ${ $data.requestId }
```

---

## Output and export

:::tip
`output.as` shapes `$output` and `export.as` writes to `$context`. Both are
evaluated against the task's raw result. For the full model, the difference
between them and worked examples, see [Data Flow](/docs/concepts/data-flow).
:::

---

## Workflow metadata in expressions

:::tip
For the full list of metadata fields, see the [DSL reference](/docs/dsl/intro).
:::

Inside workflow execution, metadata is accessible via `$data.workflow` and
`$data.activity`:

```yaml
- logAttempt:
    set:
      attempt: ${ $data.workflow.attempt }
      workflowId: ${ $data.workflow.workflow_execution_id }
```

---

## Common mistakes

**Using `uuid` or `timestamp` outside a `set` task.**
This causes a Non-Determinism Error on workflow replay. Always generate
values in a `set` task and reference them via `$data`.

**Accessing a `$data` key before the task that defines it.**
A `$data` key is only available after the task that wrote it has run. A `set`
task merges its keys directly, and an activity-backed task such as an HTTP call
stores its result under the task name. Access them in a later task.

**Confusing `$output` with `$data`.**
`$output` is the output of the last task only. `$data` accumulates workflow data
from task execution, keyed by name, and unlike `$context` is never replaced by
`export`.

---

## Related pages

- [Data Flow](/docs/concepts/data-flow): how data moves between tasks
- [Set task](/docs/dsl/tasks/set): storing data
- [DSL reference](/docs/dsl/intro): full expression context
- [How Zigflow runs](/docs/concepts/how-zigflow-runs): determinism and replay

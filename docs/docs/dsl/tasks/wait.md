# Wait

Pauses or delays workflow execution. Both forms compile to a Temporal
[Durable Timer](https://docs.temporal.io/workflow-execution/timers-delays).

## When to use this

Use wait to introduce a durable delay into your workflow. The timer
survives worker restarts. Typical uses:

- Cooldown periods between retries or external calls
- Pauses inside a `for` loop to space out iterations
- Waiting until a specific moment in time, for example a publication
  deadline or a scheduled action

## Open Workflow Specification form

The form defined by the Open Workflow Specification (formerly Serverless
Workflow). A workflow using only this form is portable across any Open
Workflow Specification runtime.

### Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| wait | duration object | `yes` | The duration to wait. |

The `wait` body is a duration object with at least one numeric field.

| Field | Type | Description |
| --- | :---: | --- |
| days | integer | Number of days. |
| hours | integer | Number of hours. |
| minutes | integer | Number of minutes. |
| seconds | integer | Number of seconds. |
| milliseconds | integer | Number of milliseconds. |

### Example: literal duration

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: wait-literal-duration
  version: 0.0.1
do:
  - pause:
      wait:
        seconds: 5
```

## Zigflow extensions

Zigflow adds two forms to the wait task that are not in the
Open Workflow Specification. The keyword stays `wait:`, so the
YAML still looks like a wait task, but a workflow that uses these
forms will only run in Zigflow.

### Expression-aware duration fields

Each numeric duration field accepts a runtime expression in addition
to a literal integer. The expression must resolve to a number at
workflow execution time. Useful when the wait duration depends on
workflow input or earlier task data.

| Field | Type | Description |
| --- | :---: | --- |
| days | runtime expression | Number of days, resolved at runtime. |
| hours | runtime expression | Number of hours, resolved at runtime. |
| minutes | runtime expression | Number of minutes, resolved at runtime. |
| seconds | runtime expression | Number of seconds, resolved at runtime. |
| milliseconds | runtime expression | Number of milliseconds, resolved at runtime. |

#### Example: expression duration

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: wait-cooldown
  version: 0.0.1
do:
  - cooldown:
      wait:
        seconds: ${ $input.cooldownSeconds }
```

### Until form

Wait until an absolute moment in time. The body accepts a single
`until` field, either an RFC 3339 string or a runtime expression that
resolves to one.

| Field | Type | Description |
| --- | :---: | --- |
| until | RFC 3339 string or runtime expression | The absolute moment to wait until. |

The until form and the duration form are mutually exclusive. Mixing
them fails schema validation.

#### Example: literal until

Pause until an absolute RFC 3339 timestamp. The same workflow always
waits for the same moment regardless of when it starts.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: wait-literal-until
  version: 0.0.1
do:
  - waitForDeadline:
      wait:
        until: 2026-12-31T23:59:59Z
```

#### Example: expression until

The until value can be a runtime expression that resolves to an RFC 3339
string at workflow execution time. The expression typically reads from
workflow input or data set earlier in the workflow.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: wait-expression-until
  version: 0.0.1
do:
  - waitForDeadline:
      wait:
        until: ${ $input.deadline }
```

## Gotchas

:::warning
**Past until values are a no-op.** If `until` resolves to a moment that
has already passed, the wait task continues immediately. This is logged
at debug level but is otherwise silent. If you rely on the wait to
enforce a minimum delay, double-check that the resolved value is in the
future.
:::

:::warning
**Runtime expressions must resolve to the right type.** Numeric duration
fields require numbers. `until` requires a string that parses as RFC 3339.
Zigflow does not coerce strings to numbers or accept arbitrary date
formats; a wrong type fails the workflow with a clear error rather than
being silently translated.
:::

:::warning
**Non-deterministic functions are rejected in wait expressions.** Zigflow
refuses to register a workflow whose wait expression uses `uuid`, `timestamp`
or `timestamp_iso8601`. These values change on every replay and would break
Temporal determinism. References to `$input`, `$data`, `$context` and `$env`
are safe because they are already in workflow history. If you need a
generated value, compute it in a preceding [set](/docs/dsl/tasks/set) task
first and reference the result here.
:::

**The timer is durable.** A wait of hours or days survives worker restarts.
Temporal holds the timer state. This is intended behaviour.

**There is no maximum duration.** Very long timers are supported by Temporal
but increase workflow history length.

## Related pages

- [For](/docs/dsl/tasks/for): using wait inside iteration loops.
- [Set](/docs/dsl/tasks/set): precomputing values so that non-deterministic
  generators can be used outside the wait task.
- [Listen](/docs/dsl/tasks/listen): waiting for external events instead of
  a fixed duration or absolute moment.
- [Concepts: data and expressions](/docs/concepts/data-and-expressions):
  how `${ ... }` expressions are evaluated.
- [Concepts: temporal prerequisites](/docs/concepts/temporal-prereqs):
  durable timers explained.
- [Extending the DSL](/docs/guides/extending-the-dsl): how the until form
  and expression-aware duration fields are implemented, as a template for
  adding further Zigflow extensions.

---
sidebar_position: 6
description: "The recurring mistakes that break Zigflow workflows: invalid task types, expression syntax, runtime variables, durations, determinism and document structure, with the correct construct for each."
---

# Common Mistakes

## What you will learn

- The recurring mistakes that cause workflows to fail validation or behave
  unexpectedly
- Why each mistake is tempting, and where the wrong habit usually comes from
- The correct Zigflow construct to use instead

:::tip
This page assumes the data model from
[Data and Expressions](/docs/concepts/data-and-expressions) and
[Data Flow](/docs/concepts/data-flow). The data-flow mistakes below are
summarised here and explained in full on those pages.
:::

Most invalid workflows break the same handful of rules. The mistakes below
come from habits carried over from other tools: the CNCF Open Workflow
Specification (formerly Serverless Workflow), GitHub Actions, Jinja
templating or the `jq` command line.
Zigflow implements a deliberately small, strict subset and rejects anything
outside it, so a habit that works elsewhere often fails here.

---

## 1. Invalid task types and call shapes

A task's type is the single task key its body contains. Zigflow supports
exactly eleven task types: `call`, `do`, `for`, `fork`, `listen`, `raise`,
`run`, `set`, `switch`, `try` and `wait`. There is no `http`, `parallel`,
`function`, `delay` or `emit` task.

**Why people get this wrong.** Other workflow engines and the wider Serverless
Workflow ecosystem expose task types like `http` or `parallel`. In Zigflow an
HTTP request is a `call`, concurrency is a `fork` and a delay is a `wait`.

### Incorrect: an `http:` task

```yaml
do:
  - fetchUser:
      http:
        url: https://api.example.com/users/2
```

### Correct: a `call: http` task

```yaml
do:
  - fetchUser:
      call: http
      with:
        method: get
        endpoint: https://api.example.com/users/2
```

Use this mapping when a task type from elsewhere does not exist in Zigflow:

| Habit from elsewhere | Zigflow equivalent |
| --- | --- |
| `http:` task | `call: http` |
| `function:` task | `call: activity` |
| `parallel:` task | [`fork`](/docs/dsl/tasks/fork) |
| `delay:` task | [`wait`](/docs/dsl/tasks/wait) |
| `emit:` task | no equivalent, not supported |

### `url` instead of `endpoint`

An HTTP call addresses its target with `endpoint`, not `url`. There is no
`url` property. The name comes from the Open Workflow Specification, but
most HTTP tooling calls it a URL, so `url` is a frequent slip.

```yaml
# Incorrect: url is not a recognised field, endpoint is missing
with:
  method: post
  url: https://api.example.com/orders

# Correct
with:
  method: post
  endpoint: https://api.example.com/orders
```

:::warning
Unsupported constructs are rejected, not ignored. Zigflow validates a workflow
before it runs and fails with messages such as `unsupported task type` or
`unsupported call type`. A feature being present in the CNCF Serverless
Workflow specification does not mean Zigflow implements it. When in doubt,
check the [task reference](/docs/dsl/tasks/intro).
:::

---

## 2. Expression syntax and runtime variables

Runtime expressions are [jq](https://jqlang.org/) (via gojq) wrapped in
`${ ... }`. A string without `${ }` is a literal value, not an expression.

**Why people get this wrong.** Every templating tool uses a different wrapper.
GitHub Actions uses `${{ }}`, Jinja and Handlebars use `{{ }}`, and the `jq`
command line uses a bare `.` or `$.` path. None of these work in Zigflow.

### Incorrect: the wrong wrappers

```yaml
set:
  a: '{{ $input.name }}'   # Jinja or Handlebars
  b: '${{ $input.name }}'  # GitHub Actions
  c: '$.input.name'        # bare jq path
```

### Correct: `${ ... }`

```yaml
set:
  greeting: ${ $input.name }
```

### Invented runtime variables

Only five variables exist in an expression: `$context`, `$data`, `$env`,
`$input` and `$output`. Names such as `$workflow`, `$task`, `$steps`, `$vars`
or `$now` do not exist and fail at evaluation.

:::warning
There is no `$now`. Use `${ timestamp }` or `${ timestamp_iso8601 }`, and only
inside a [`set`](/docs/dsl/tasks/set) task, so the value is stable across
Temporal replays. See section 5 below.
:::

Map the common invented variables to the real ones:

| Invented variable | Use instead |
| --- | --- |
| `$workflow`, `$workflow.id` | `$data.workflow`, `$data.workflow.workflow_execution_id` |
| `$task`, `$steps` | `$output` for the last task, or `$data.<taskName>` by name |
| `$vars` | `$context` for exported data, `$data` for workflow state |
| `$now` | `${ timestamp }` or `${ timestamp_iso8601 }` inside a `set` task |

---

## 3. Misreading the data-flow model

:::tip
These four mistakes are the same root confusion between Zigflow's two data
channels. [Data Flow](/docs/concepts/data-flow) explains the model with worked
examples. This is the short version.
:::

- **Confusing `export` and `output`.** `output.as` shapes `$output`, the result
  that flows to the next task and that the workflow returns. `export.as` writes
  to `$context`, which persists for later tasks. They are different channels.
- **Assuming `export` merges into `$context`.** Each `export.as` replaces
  `$context` wholesale. To add to it without losing earlier values, merge
  explicitly with `${ $context + { ... } }`.
- **Reading `$output` after intervening tasks.** `$output` only holds the most
  recent task's result. To use a value several steps later, `export` it to
  `$context`.
- **Expecting outputs to flow through a task's input.** A task does not
  automatically receive the previous task's result as its working input. Chain
  results explicitly through `$output`, or read a named result from
  `$data.<taskName>`.

:::warning
`$data` and `$context` are not the same. `$data` accumulates across tasks and
is never replaced, but how a task writes to it depends on the task type. A
`set` task merges its fields directly, so a `set` that defines `requestId` is
read as `$data.requestId`. An activity-backed task stores its result under the
task name, so an HTTP task named `fetchUser` is read as `$data.fetchUser`.
`$context` is written only by `export` and is replaced on each `export`. A
`$data` key is only readable after the task that wrote it has run.
:::

---

## 4. Durations

A duration is an object with plural integer keys. The valid keys are `days`,
`hours`, `minutes`, `seconds` and `milliseconds`. ISO 8601 strings such as
`PT1M` are not the supported format, and singular keys such as `minute` are not
recognised.

**Why people get this wrong.** ISO 8601 durations appear throughout the
Open Workflow Specification and many standards, and singular keys read
naturally in English.

### Incorrect: ISO 8601 and singular keys

```yaml
do:
  - pauseIso:
      wait: PT1M        # ISO 8601 string, not supported
  - pauseSingular:
      wait:
        minute: 1       # singular key, rejected by validation
```

### Correct: plural keys

```yaml
do:
  - pause:
      wait:
        minutes: 1
        seconds: 30
```

:::warning
Unknown duration keys are validation errors. A typo such as `minute` is
rejected because wait durations only support the documented plural keys. Always
use the plural keys exactly. See the [wait task](/docs/dsl/tasks/wait).
:::

---

## 5. Non-deterministic values in the wrong place

Generated values such as `${ uuid }`, `${ timestamp }` and
`${ timestamp_iso8601 }` must be produced inside a [`set`](/docs/dsl/tasks/set)
task and referenced later through `$data`.

**Why people get this wrong.** It is natural to drop a `${ uuid }` straight into
an HTTP body or a `wait` expression. Temporal replays workflow history, so a
value generated outside a recorded side effect changes on replay and raises a
non-determinism error.

### Incorrect: generated inside a call

```yaml
do:
  - callApi:
      call: http
      with:
        method: post
        endpoint: https://api.example.com/orders
        body:
          requestId: ${ uuid }   # regenerated on replay
```

### Correct: generated in a `set` task

```yaml
do:
  - generateId:
      set:
        requestId: ${ uuid }     # recorded as a side effect
  - callApi:
      call: http
      with:
        method: post
        endpoint: https://api.example.com/orders
        body:
          requestId: ${ $data.requestId }
```

:::warning
A `wait` expression rejects these functions outright. Zigflow refuses to
register a workflow whose `wait` duration uses `uuid`, `timestamp` or
`timestamp_iso8601`. Compute the value in a preceding `set` task and reference
the result. See [Data and Expressions](/docs/concepts/data-and-expressions) for
the full determinism rules.
:::

---

## 6. Workflow document structure

The `document` object is closed, the `do` list has a fixed shape, and several
fields have strict formats. These are caught at validation time.

### Unknown `document` fields

`document` rejects unknown fields. There is no `name`, `description` or
`author`. Put a human-readable name in `title`, prose in `summary` and anything
else under `metadata` or `tags`.

```yaml
# Incorrect
document:
  name: My Workflow
  description: Does things
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: my-workflow
  version: 0.0.1

# Correct
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: my-workflow
  version: 0.0.1
  title: My Workflow
  summary: Does things
```

### The `do` list shape

`do` is an array of single-key maps, each `{ stepName: { task } }`. It is not a
map of steps, and a step must contain exactly one task key.

```yaml
# Incorrect: do as a map
do:
  stepOne:
    set: { a: 1 }
  stepTwo:
    set: { b: 2 }

# Correct: do as a list of single-key maps
do:
  - stepOne:
      set: { a: 1 }
  - stepTwo:
      set: { b: 2 }
```

### Invalid names and versions

Zigflow requires `taskQueue` and `workflowType` to be RFC 1123 DNS labels:
letters, digits and hyphens, starting and ending with a letter or digit. No
underscores, dots or spaces. This is a Zigflow validation rule, not a Temporal
one. `dsl` and `version` must be semantic versions.

```yaml
# Incorrect
taskQueue: my_queue.v2

# Correct
taskQueue: my-queue-v2
```

---

## Quick reference

| Instead of | Use |
| --- | --- |
| `http:`, `parallel:`, `function:`, `delay:`, `emit:` tasks | `call: http`, `fork`, `call: activity`, `wait` |
| `url:` | `endpoint:` |
| `{{ }}`, `${{ }}`, `$.foo` | `${ ... }` |
| `$workflow`, `$task`, `$steps`, `$vars`, `$now` | `$data`, `$output`, `$context`, `$data.workflow` |
| `PT1M`, `minute: 1` | `minutes: 1` |
| `${ uuid }` in a call or wait | `${ uuid }` in a `set` task, then `$data` |
| `document.name`, `document.description` | `document.title`, `document.summary` |
| `do` as a map | `do` as a list of single-key maps |

---

## Related pages

- [Data Flow](/docs/concepts/data-flow): `$output`, `$context`, `export` and
  `output`
- [Data and Expressions](/docs/concepts/data-and-expressions): expression
  syntax, variables and determinism
- [Tasks: introduction](/docs/dsl/tasks/intro): the eleven task types
- [Call task](/docs/dsl/tasks/call): HTTP, gRPC and activity calls
- [Wait task](/docs/dsl/tasks/wait): durations and the `until` form
- [DSL reference](/docs/dsl/intro): the full `document` and task schema

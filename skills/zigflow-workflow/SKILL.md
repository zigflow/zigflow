---
name: zigflow-workflow
version: 1.0.0
description: >
  Write, validate and debug Zigflow workflow YAML definitions. Loads the DSL
  schema, checks expressions, suggests task types, enforces determinism rules
  and state lifecycle.

  Use when the user wants to write a new workflow, add tasks to an existing
  workflow, fix validation errors, or understand how to use the Zigflow DSL.

  Keywords: workflow, yaml, dsl, task, set, call, http, grpc, activity, for,
  fork, switch, wait, listen, signal, query, raise, try, catch, run, schedule,
  expression, validate, determinism, output, export, context, data
---
# Zigflow Workflow Authoring Guide

You are an expert at writing Zigflow workflow YAML definitions.
Zigflow compiles declarative YAML into Temporal workflows.

## When to Use

- Writing a new workflow YAML from scratch
- Adding tasks to an existing workflow
- Fixing validation errors from `zigflow validate`
- Understanding which task type to use
- Debugging expression or state issues
- Converting workflow ideas into valid Zigflow YAML

## When NOT to Use

- Modifying the Zigflow engine source code (Go code in `pkg/` or `cmd/`)
- Deploying or running workflows (`zigflow run` handles that)
- Writing Temporal SDK code directly

---

## Workflow Structure

Every workflow file has this shape:

```yaml
document:
  dsl: 1.0.0
  taskQueue: my-queue        # RFC 1123 DNS label (letters, digits, hyphens)
  workflowType: my-workflow  # RFC 1123 DNS label
  version: 0.0.1            # semver
  title: Human-Readable Name # optional
  summary: What this does    # optional
  metadata:                  # optional, for activityOptions, schedules, tags
    activityOptions:
      startToCloseTimeout:
        minutes: 1
input:                       # optional, JSON Schema validation
  schema:
    format: json
    document:
      type: object
      required: [userId]
      properties:
        userId:
          type: number
do:
  - stepName:
      # exactly one task key per step
```

### Critical Rules

1. `do` is an **array of single-key maps**, NOT a map of steps
2. `taskQueue` and `workflowType` must be RFC 1123 DNS labels
   (no underscores, dots or spaces)
3. `dsl` and `version` must be valid semver
4. `document` rejects unknown fields. No `name`, `description`,
   or `author`. Use `title` and `summary`

---

## Task Types (exactly 11)

| Task | Purpose | Example |
| ------ | --------- | --------- |
| `set` | Set values in `$data`, safe for non-deterministic functions | `set: { id: "${ uuid }" }` |
| `call: http` | Make HTTP requests | `call: http` + `with: { method, endpoint }` |
| `call: grpc` | Make gRPC calls | `call: grpc` + `with: { service, method, proto }` |
| `call: activity` | Call external Temporal activities | `call: activity` + `with: { name, arguments, taskQueue }` |
| `do` | Group tasks into a sub-workflow | `do: [...]` |
| `for` | Loop over arrays, objects or counts | `for: { in: expr }` + `do: [...]` |
| `fork` | Run branches in parallel | `fork: { compete: bool, branches: [...] }` |
| `switch` | Conditional routing | `switch: [{ case: { when: expr, then: target } }]` |
| `wait` | Pause on a durable timer | `wait: { seconds: 5 }` |
| `listen` | Wait for signals or queries | `listen: { to: { one: { with: { id, type } } } }` |
| `try` | Error handling | `try: [...] catch: { do: [...] }` |
| `raise` | Throw an error | `raise: { error: { type, status } }` |
| `run` | Execute child workflows or containers | `run: { workflow: { type: name } }` |

**There is no** `http`, `parallel`, `function`, `delay` or `emit` task type.

---

## Expressions

Runtime expressions use jq syntax wrapped in `${ ... }`:

```yaml
set:
  greeting: ${ "Hello " + $input.name }
```

### Available Variables

| Variable | Meaning |
| ---------- | --------- |
| `$input` | Workflow input (immutable) |
| `$data` | Accumulated task results (grows per task) |
| `$context` | Exported state (replaced on each `export`) |
| `$output` | Most recent task's result (ephemeral) |
| `$env` | Environment variables loaded by prefix |

**No** `$workflow`, `$task`, `$steps`, `$vars`, or `$now`.

### Built-in Functions (only in `set` tasks)

| Function | Purpose |
| ---------- | --------- |
| `uuid` | Generate a UUID |
| `timestamp` | Unix timestamp |
| `timestamp_iso8601` | ISO 8601 timestamp |

These are **non-deterministic** and MUST only be used inside `set`
tasks. Using them in `call`, `wait`, or other tasks causes
validation failure.

---

## Data Flow Model

### `output.as` vs `export.as`

| Directive | Writes to | Purpose |
| --- | --- | --- |
| `output.as` | `$output` | Shapes workflow return value |
| `export.as` | `$context` | Persists values for later tasks |

### How `$data` works

- A `set` task merges its fields directly: `set: { foo: 1 }` → `$data.foo`
- An activity-backed task (`call: http`, etc.) stores result
  under task name: task named `fetchUser` → `$data.fetchUser`

### Key rule: `export` REPLACES `$context`

To preserve prior context values, merge explicitly:

```yaml
export:
  as: '${ $context + { newField: . } }'
```

---

## Duration Format

Durations use **plural integer keys**. No ISO 8601.

```yaml
wait:
  days: 1
  hours: 2
  minutes: 30
  seconds: 15
  milliseconds: 500
```

**Invalid**: `PT1M`, `minute: 1`, `second: 5`

---

## Determinism Rules

Temporal replays workflow history. These rules prevent non-determinism errors:

1. `uuid`, `timestamp`, `timestamp_iso8601` → only in `set` tasks
2. Reference the generated value via `$data` in later tasks
3. No wall-clock time in `wait` expressions directly

### Pattern: generate-then-use

```yaml
do:
  - generateId:
      set:
        requestId: ${ uuid }
  - callApi:
      call: http
      with:
        method: post
        endpoint: https://api.example.com/orders
        body:
          requestId: ${ $data.requestId }
```

---

## Common Patterns

### HTTP Call

```yaml
- fetchUser:
    call: http
    with:
      method: get
      endpoint: ${ "https://api.example.com/users/" + ($input.userId | tostring) }
```

### For Loop (iterate array)

```yaml
- processItems:
    for:
      each: item
      in: ${ $input.items }
      at: index
    do:
      - handle:
          set:
            current: ${ $data.item }
```

### For Loop (repeat N times)

```yaml
- repeat:
    for:
      in: ${ 5 }
    do:
      - step:
          set:
            iteration: ${ $data.index }
```

### Fork (parallel, all complete)

```yaml
- parallel:
    fork:
      compete: false
      branches:
        - branchA:
            do: [...]
        - branchB:
            do: [...]
```

### Fork (race, first wins)

```yaml
- race:
    fork:
      compete: true
      branches:
        - fast:
            do: [...]
        - slow:
            do: [...]
```

### Switch (conditional routing)

```yaml
- router:
    switch:
      - caseA:
          when: ${ $input.type == "a" }
          then: handleA
      - caseB:
          when: ${ $input.type == "b" }
          then: handleB
      - default:
          then: handleDefault
```

Flow directives for `then`: `continue`, `exit`, `end`, or a named task.

### Signal Listener

```yaml
- awaitApproval:
    listen:
      to:
        one:
          with:
            id: approve
            type: signal
```

### Query Listener

```yaml
- queryState:
    listen:
      to:
        one:
          with:
            id: get_state
            type: query
            data:
              status: ${ $data.status }
```

### Try/Catch

```yaml
- safe:
    try:
      - riskyCall:
          call: http
          with:
            method: get
            endpoint: https://might-fail.example.com
    catch:
      do:
        - handleError:
            set:
              error: caught
```

### Child Workflow

```yaml
- callChild:
    run:
      workflow:
        type: child-workflow-name
```

### Schedule

```yaml
document:
  metadata:
    scheduleWorkflowName: my-workflow
    scheduleId: my-schedule
schedule:
  every:
    minutes: 30
  cron: "0 9 * * *"
```

### Multiple Workflows in One File

```yaml
do:
  - parentWorkflow:
      do:
        - step1:
            set: { a: 1 }
        - callChild:
            run:
              workflow:
                type: child-workflow
  - child-workflow:
      do:
        - step1:
            set: { b: 2 }
```

---

## Validation

Always validate after writing:

```bash
zigflow validate workflow.yaml
```

For JSON output (CI-friendly):

```bash
zigflow validate --output-json workflow.yaml
```

Common validation errors:

| Error | Cause | Fix |
| ------- | ------- | ----- |
| `unsupported task type` | Used `http:` instead of `call: http` | Use correct task type |
| `non-deterministic expression` | `uuid`/`timestamp` outside `set` | Move to a `set` task |
| `unsupported dsl version` | `dsl` not in range `>=1.0.0, <2.0.0` | Use `dsl: 1.0.0` |
| Schema error at `document` | Unknown field in document | Remove `name`/`description`, use `title`/`summary` |
| `endpoint` missing | Used `url` instead of `endpoint` | Rename to `endpoint` |

---

## Workflow Approach

When helping the user write a workflow:

1. **Ask what the workflow should do** if not clear
2. **Pick the right task types** from the 11 available
3. **Generate valid YAML** following all rules above
4. **Check determinism** — any generated value in `set` first
5. **Check data flow** — use `export` to persist, `output` to shape returns
6. **Suggest running** `zigflow validate` to confirm

When fixing validation errors:

1. Read the error message carefully
2. Map it to the common-mistakes table above
3. Apply the fix
4. Explain why it failed (so the user learns)

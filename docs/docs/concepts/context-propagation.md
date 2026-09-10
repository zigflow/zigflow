---
sidebar_position: 6
description: "How Zigflow exposes Temporal context propagation to workflows through the read-only $propagated object, and how a client supplies the values."
---

# Context Propagation

## What you will learn

- What `$propagated` contains and where the values come from
- How `$propagated` differs from `$context`
- How a client supplies propagated values
- What happens to propagated values across Temporal execution boundaries

:::tip
For the full list of runtime variables and the expression syntax, see
[Data and Expressions](/docs/concepts/data-and-expressions).
:::

---

## What context propagation is

:::info
For more information about context propagation in Temporal, see the
[Temporal docs](https://docs.temporal.io/encyclopedia/context-propagation)
:::

Temporal can carry caller-supplied values alongside a workflow execution. The
values travel in Temporal headers rather than in the workflow input, so they
are available to the workflow without changing its input contract.

Zigflow registers a Temporal context propagator and exposes whatever it
receives as the read-only root-level `$propagated` object.

This is useful for values that describe the caller rather than the work: a
tenant identifier, a request correlation identifier, a locale, a region.

---

## Reading `$propagated`

`$propagated` is an object of key and value pairs. Read a key with an ordinary
runtime expression.

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: propagated-example
  version: 1.0.0
do:
  - recordCaller:
      set:
        tenantId: ${ $propagated.tenantId }
        correlationId: ${ $propagated.correlationId }
```

Propagated values are ordinary workflow data. Use them anywhere an expression
is valid, including conditions, task inputs, HTTP bodies and exports.

```yaml
- fetchTenantConfig:
    call: http
    with:
      method: get
      endpoint: ${ "https://api.example.com/tenants/" + $propagated.tenantId }
```

If nothing was propagated, `$propagated` is an empty object and reading a key
returns `null`. Guard against a missing key the same way you would any other
optional value:

```yaml
- recordTenant:
    set:
      tenantId: ${ $propagated.tenantId // "unknown" }
```

The propagated payload must decode to an object. If a client sends something
else, such as a bare string, the workflow task fails and Temporal retries it,
so the execution makes no progress until the client is corrected.

---

## `$propagated` is not `$context`

The two are easy to confuse. They come from different places and behave
differently.

| | `$propagated` | `$context` |
| --- | --- | --- |
| Source | The caller, through Temporal context propagation | Tasks in the workflow, through `export.as` |
| Written by | Nothing in the DSL. It is read-only | Each task's `export.as` |
| Type | Always an object | Any value |
| Changes during the run | No | Yes, each `export` replaces it |

`$context` remains the user-controlled workflow context described in
[Data Flow](/docs/concepts/data-flow). Context propagation does not change it.

---

## Lifecycle

Propagated values are established once, when the workflow execution starts, and
do not change for the rest of the run.

- They are read from Temporal context propagation as the workflow begins.
- They are retained in Zigflow's workflow state, so they survive
  [Continue-As-New](/docs/dsl/metadata/continue-as-new).
- They are carried across Temporal execution boundaries, such as activities and
  child workflows, by the registered context propagator.
- Nested `do` tasks share the parent workflow's state, so they see the same
  values.

There is no DSL mechanism to add to or modify `$propagated` during a run. To
carry values you compute yourself, use `export.as` and `$context`.

---

## Supplying propagated values from a client

Propagated values are supplied by the application that starts the workflow, not
by Zigflow.

Zigflow reads a single Temporal header field:

```text
zigflow.propagated
```

The field holds a payload encoding an object of key and value pairs, written
with the Temporal SDK's default payload converter. Because the values become a
Temporal payload, they must be portable, serialisable data. Strings, numbers,
booleans, and objects and arrays of those, are all fine. Language-specific
objects, open connections and handles are not.

Zigflow does not expose a language-specific `context` object to the workflow.
The portable contract is the propagated data itself.

### SDK differences

Temporal SDKs expose this in different ways:

- Go and Java have a native context propagator concept. A Go client can
  register Zigflow's propagator directly.
- Other SDKs, including Python and TypeScript, use a client interceptor that
  sets the `zigflow.propagated` header when starting a workflow.

Either approach produces the same header, so the workflow sees the same
`$propagated` object.

The `examples/` directory in the Zigflow repository contains working clients for
Go, Python and TypeScript.

---

## Choosing what to propagate

:::warning
Avoid propagating sensitive values such as access tokens, credentials or
personal data. Propagated values are stored in Temporal history and, by
default, use Temporal's standard payload conversion rather than Zigflow's
configured payload codec.

Context propagation is intended for small pieces of execution metadata such as
correlation IDs, tenant identifiers, locale and region.
:::

Keep the set small and stable. Propagation is for identifiers and small
descriptive values, not for workflow input. Anything the workflow operates on
belongs in the workflow input, where it is schema-checked and explicit.

Only `$propagated.correlationId` receives special treatment, and only in
logging. See
[Observability: correlation IDs in logs](/docs/deployment/observability#correlation-ids-in-logs).

---

## Common mistakes

**Trying to write to `$propagated`.**
It is read-only. No task writes to it. Use `set` to write `$data` or `export`
to write `$context`.

**Expecting `$propagated` to change during a run.**
Values are fixed when the execution starts. A later caller cannot change them.

**Confusing `$propagated` with `$context`.**
`$propagated` comes from the caller and never changes. `$context` comes from
your own tasks and is replaced by every `export`.

**Assuming every propagated value is logged.**
Only a string `correlationId` is added to log entries automatically. Everything
else is readable through `$propagated` but is not logged.

**Propagating non-serialisable data.**
The values must survive conversion to a Temporal payload.

---

## Related pages

- [Data and Expressions](/docs/concepts/data-and-expressions): the runtime
  variables and expression syntax
- [Data Flow](/docs/concepts/data-flow): how `$output` and `$context` move data
  between tasks
- [Observability](/docs/deployment/observability#correlation-ids-in-logs):
  automatic `correlationId` logging
- [Continue-As-New](/docs/dsl/metadata/continue-as-new): long-running workflows
  and history size
- [How Zigflow runs](/docs/concepts/how-zigflow-runs): the execution model

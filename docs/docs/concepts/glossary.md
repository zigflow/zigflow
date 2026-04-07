---
sidebar_position: 6
---

# Glossary

Short definitions of key terms used throughout the Zigflow documentation.

---

## Activity

A unit of work that can have side effects: HTTP calls, database writes, shell
commands. Activities run outside the deterministic Temporal workflow context
and are retried automatically on failure.

In Zigflow, tasks such as `call: http`, `call: activity` and `run` execute
as Temporal activities.

See also: [Temporal docs: activities](https://docs.temporal.io/activities)

---

## Child workflow

A workflow started by a parent workflow. It runs with its own history and
can be awaited by the parent. Used internally by `fork`, `try` and certain
`run` modes.

See also: [Temporal docs: child workflows](https://docs.temporal.io/child-workflows)

---

## DSL

Domain-specific language. In this context, the YAML format that Zigflow uses
to define workflows.

Zigflow's DSL is based on the [CNCF Serverless Workflow v1.0 specification](https://serverlessworkflow.io).

---

## Durable timer

A timer that survives worker restarts. When the timer fires, the workflow
resumes. Implemented in Zigflow via the [`wait`](/docs/dsl/tasks/wait) task.

---

## Flow directive

A value that controls workflow execution flow. Valid values:

- `"continue"`: proceed to the next task (default)
- `"exit"`: exit the current scope
- `"end"`: end the workflow
- A task name: jump to that named task

Used in the [`switch`](/docs/dsl/tasks/switch) task.

---

## Namespace (Temporal)

An isolation boundary in Temporal. The default is `default`. Set at runtime
with `--temporal-namespace`.

---

## Query

A read-only operation on a running workflow. The caller receives an
immediate response. Queries cannot modify workflow state.

In Zigflow, handled by the [`listen`](/docs/dsl/tasks/listen) task with
`type: query`.

---

## Runtime expression

An expression inside `${ }` that is evaluated at execution time. Uses jq
syntax.

See [Data and expressions](/docs/concepts/data-and-expressions).

---

## Signal

A one-way message sent to a running workflow. The sender receives no
response. The workflow can react to the signal.

In Zigflow, handled by the [`listen`](/docs/dsl/tasks/listen) task with
`type: signal`.

---

## Task queue

A named channel in Temporal that connects clients and workers. Workers poll
a task queue for work. Clients start executions on a task queue.

In Zigflow, `document.taskQueue` sets the task queue name.

---

## Temporal

A durable execution platform. Persists workflow state, handles retries and
coordinates workers. Zigflow requires a running Temporal server.

See [Temporal prerequisites](/docs/concepts/temporal-prereqs).

---

## Update

A validated read/write operation on a running workflow. Unlike a signal,
the caller receives a response. Unlike a query, an update can modify state.

In Zigflow, handled by the [`listen`](/docs/dsl/tasks/listen) task with
`type: update`.

---

## Worker

A process that polls Temporal for tasks and executes them. `zigflow run`
starts a Temporal worker.

---

## Workflow type

The name Temporal uses to identify a workflow. In Zigflow, this is
`document.workflowType`. Clients must use this name when starting an execution.

---

## Related pages

- [Overview](/docs/concepts/overview): mental model
- [Temporal prerequisites](/docs/concepts/temporal-prereqs): Temporal
  concepts in detail
- [DSL reference](/docs/dsl/intro): full YAML specification

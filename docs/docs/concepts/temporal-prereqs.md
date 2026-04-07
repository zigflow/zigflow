---
sidebar_position: 2
---

# Temporal Prerequisites

## What you will learn

- The minimum Temporal knowledge required to use Zigflow
- Definitions of key Temporal concepts
- Where to learn more

---

## How much Temporal do you need to know?

You do not need to be a Temporal expert to use Zigflow. However, understanding
a few core concepts will help you write correct workflows and debug problems
when they arise.

---

## Core concepts

### Temporal server

The Temporal server stores workflow state, dispatches tasks to workers, and
enforces retry policies. It is a separate process from Zigflow.

For local development, the fastest way to start one is with the
[Temporal CLI](https://docs.temporal.io/cli):

```sh
temporal server start-dev
```

For production, use [Temporal Cloud](https://temporal.io/cloud) or a
[self-hosted deployment](https://docs.temporal.io/self-hosted-guide).

---

### Namespace

A Temporal namespace is an isolation boundary for workflows. The default
namespace is `default`.

In Zigflow, the Temporal namespace is set at runtime with the
`--temporal-namespace` flag, not in the YAML definition.

Do not confuse this with `document.taskQueue`, which sets the task queue name.

---

### Task queue

A task queue is a named channel between clients and workers. Workers poll a
task queue for work. Clients start executions on a task queue.

In Zigflow, `document.taskQueue` in your YAML sets the task queue name. When
you run `zigflow run`, the worker registers itself on that task queue.

---

### Workflow

A Temporal workflow is a durable, deterministic function. Temporal replays
the workflow history to resume after failure or restart.

In Zigflow, the workflow is your YAML definition. Zigflow compiles it into a
Temporal workflow function.

---

### Activity

An activity is a unit of work that can have side effects: HTTP calls, database
writes, shell commands. Activities run outside the deterministic workflow
context.

In Zigflow, tasks such as `call: http` and `call: activity` execute as
Temporal activities.

---

### Worker

A worker is a process that polls Temporal for tasks and executes them.

`zigflow run` starts a Temporal worker. The worker polls the task queue
specified by `document.taskQueue` and executes workflow and activity tasks.

---

### Signal

A signal is a one-way message sent to a running workflow. The workflow can
receive the signal and react to it, but the sender receives no response.

In Zigflow, signals are handled with the `listen` task using `type: signal`.

---

### Query

A query is a synchronous read from a running workflow. The caller receives
an immediate response. Queries do not modify workflow state.

In Zigflow, queries are handled with the `listen` task using `type: query`.

---

### Update

An update is a validated, synchronous read/write operation on a running
workflow. Unlike a signal, the caller receives a response. Unlike a query,
an update can modify workflow state.

In Zigflow, updates are handled with the `listen` task using `type: update`.

---

### Durable timer

A durable timer suspends a workflow for a specified duration. Unlike a sleep
call, a durable timer survives worker restarts.

In Zigflow, the `wait` task creates a Temporal durable timer.

---

### Child workflow

A child workflow is a workflow started by a parent workflow. It runs
independently and can be awaited or left to run concurrently.

In Zigflow, some tasks (such as `fork`, `try` and `run`) create child
workflows internally.

---

## Where to learn more

- [Temporal 101 course](https://learn.temporal.io/courses/temporal_101/): a
  practical introduction to Temporal concepts
- [Temporal documentation](https://docs.temporal.io): full reference
- [Temporal encyclopedia](https://docs.temporal.io/encyclopedia): deep dives
  into specific topics

---

## Common mistakes

**Starting a workflow execution before the worker is running.**
The Temporal server will queue the execution, but it will not start until a
worker is polling the task queue.

**Using the wrong task queue name.**
Check that `document.taskQueue` in your YAML matches what your client uses as
the task queue.

---

## Related pages

- [Overview](/docs/concepts/overview): Zigflow in one page
- [How Zigflow runs](/docs/concepts/how-zigflow-runs): what happens when you
  run a workflow
- [Listen task](/docs/dsl/tasks/listen): signals, queries and updates in Zigflow

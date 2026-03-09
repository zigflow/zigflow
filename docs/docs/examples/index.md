---
sidebar_position: 1
---

# Examples

A curated set of examples that demonstrate common Zigflow patterns.

## Beginner

| Example | Description | Key concepts |
| --- | --- | --- |
| [Hello World](/docs/examples/hello-world) | Minimal single-task workflow | `set`, `output.as` |
| [HTTP Call](/docs/examples/http-call) | External HTTP request | `call: http`, retries |

## Intermediate

| Example | Description | Key concepts |
| --- | --- | --- |
| [Error Handling](/docs/examples/error-handling) | Catch errors and recover | `try`, `catch`, `raise` |
| [Parallel Tasks](/docs/examples/parallel-tasks) | Run branches concurrently | `fork`, `compete` |
| [Signal-Driven Workflow](/docs/examples/signal-driven) | Pause for an external signal | `listen`, `signal` |

---

## Repository examples

The
[examples directory](https://github.com/zigflow/zigflow/tree/main/examples)
in the repository contains additional patterns:

- `basic`: combined set, wait, fork and HTTP in one workflow
- `for-loop`: iterating over arrays, maps and numbers
- `child-workflows`: calling one workflow from another
- `query`: exposing workflow state via Temporal queries
- `update`: read/write handlers for running workflows
- `schedule`: scheduled workflow triggers
- `money-transfer`: compensating transaction logic
- `external-calls`: HTTP and gRPC in a single fork

---

## Related pages

- [Quickstart](/docs/getting-started/quickstart): your first workflow
- [DSL reference](/docs/dsl/intro): full workflow YAML reference
- [Concepts: overview](/docs/concepts/overview): mental model

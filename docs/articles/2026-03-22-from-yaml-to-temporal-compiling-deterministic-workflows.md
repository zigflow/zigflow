---
title: "From YAML to Temporal: compiling deterministic workflows"
description: How Zigflow validates and compiles YAML workflows into deterministic Temporal executions.
slug: how-zigflow-maps-yaml-to-temporal
authors: mrsimonemms
---
Zigflow allows workflows to be defined declaratively in YAML.

Under the hood, everything still runs on Temporal.

So how does that mapping actually work?

{/*truncate*/}

## The core idea

Zigflow does not interpret YAML at runtime.

Instead, it follows a pipeline:

1. Parse the workflow definition
2. Validate it
3. Compile it
4. Execute it as a Temporal workflow

This separation is key.

## Validation first

Before execution, Zigflow validates the workflow:

- schema validation ensures structure is correct
- specification validation checks compatibility
- Zigflow-specific constraints enforce determinism

Invalid workflows are rejected before execution.

In addition, Zigflow detects patterns that may lead to non-deterministic behaviour.

For example, using values like `${ uuid }` in unsupported contexts will trigger
warnings or validation errors before the workflow is ever run.

This shifts feedback from runtime failures to definition-time validation.

## Compilation

Once validated, the workflow is compiled into a Temporal workflow implementation.

At this stage:

- control flow is mapped to deterministic patterns
- activity calls are defined explicitly
- state transitions are derived from the DSL

The result is not dynamic interpretation.

It is a fixed execution model.

## Execution

At runtime:

- Zigflow starts a Temporal worker
- the compiled workflow is registered
- activities are executed by external workers

Zigflow handles orchestration.

Your services handle the work.

## Visualising workflows

Because workflows are defined declaratively, Zigflow can generate a visual
representation of the workflow structure.

This can be rendered as a Mermaid graph, making it easier to:

- understand workflow flow
- review changes
- debug behaviour
- communicate workflow design

Instead of reading code, workflows can be inspected as diagrams.

## Determinism guarantees

Temporal workflows must be deterministic.

Zigflow enforces this by:

- disallowing arbitrary code execution
- modelling all side effects as activities
- ensuring state transitions are declarative
- validating or warning on potentially non-deterministic constructs

This reduces the likelihood of subtle replay errors that would otherwise only
appear at runtime.

Workflows are not just constrained, they are checked before execution.

## Why this approach matters

By separating:

- definition
- validation
- compilation
- execution

Zigflow creates a system that is:

- easier to reason about
- safer to run in production
- more suitable for automation and tooling

It also enables higher-level abstractions such as visual workflow builders.

## Final thought

Zigflow is not a runtime.

It is a compilation and orchestration layer.

Temporal remains the execution engine.

Zigflow provides a different way to define workflows on top of it.

---
title: When not to use Zigflow
description: Zigflow is not the right tool for every use case. Here's when you should use Temporal directly.
slug: when-not-to-use-zigflow
authors: mrsimonemms
---
Zigflow is a declarative layer on top of Temporal.

It simplifies workflow orchestration by treating workflows as configuration rather
than code.

Understanding when not to use Zigflow is just as important as understanding when
you should.

{/*truncate*/}

## If you need full SDK control

Temporal's SDKs provide complete control over workflow execution.

If your use case depends on:

- advanced workflow APIs
- fine-grained retry or timeout behaviour
- custom workflow execution patterns

then the SDK is the better choice.

Zigflow intentionally does not expose the full surface area of Temporal.

## If your workflows are highly dynamic

Zigflow workflows are declarative.

The workflow structure is defined up front.

If your workflows:

- generate structure at runtime
- rely on complex dynamic branching
- behave more like an execution engine

then a code-first approach is a better fit.

## If your team prefers code-first development

Some teams rely heavily on:

- debugging workflow code directly
- unit testing workflow logic
- using language-level abstractions

Zigflow changes that model. Workflow logic moves from code into declarative definitions.

Workflows become configuration rather than code.

For some teams, that's a benefit.
For others, it introduces friction.

## If your workflows contain complex business logic

Zigflow is optimised for orchestration.

It works best when:

- workflows coordinate activities
- business logic lives in services

If most of your logic lives inside the workflow itself, writing workflows in code
may be clearer.

## The key point

Zigflow is not trying to replace Temporal.

It is an opinionated layer on top of it.

If you need maximum flexibility, use Temporal directly.

If you want structured, declarative orchestration with stronger guardrails,
Zigflow may be a better fit.

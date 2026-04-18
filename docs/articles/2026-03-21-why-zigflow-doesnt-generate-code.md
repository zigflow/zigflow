---
title: Why Zigflow doesn't generate code
description: Why Zigflow avoids generating Go code and instead executes workflows directly from declarative definitions.
slug: why-zigflow-doesnt-generate-code
authors: mrsimonemms
---
One of the more common questions about Zigflow is:

> Why not convert the YAML into Go code and compile it into a Temporal workflow?

At first glance, that seems like a natural approach.

You keep full access to the Temporal SDK, while still allowing workflows to be
defined declaratively.

But Zigflow deliberately does not do this.

{/*truncate*/}

## The obvious approach: generate code

If you treat the YAML as a source format, you could:

1. Parse the workflow definition
2. Generate Go code
3. Compile it into a Temporal worker
4. Run it like any other workflow

This would preserve the full flexibility of the SDK.

It's a reasonable design.

## Why Zigflow doesn't do this

Zigflow is not trying to be a code generator.

It is trying to remove the need for code entirely.

The target use case is not:

> developers writing workflows more conveniently

It is closer to:

> defining workflows as configuration that can be validated and executed directly

Introducing a code generation step pushes the system back into a
developer-centric model:

- you now have generated code to manage
- you need a build step
- you need a compilation pipeline
- you reintroduce SDK-level complexity

At that point, you've lost most of the benefits of a declarative system.

## Why use an existing specification

Zigflow does not define its own workflow language from scratch.

Instead, it builds on the [CNCF Serverless Workflow specification](https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md).

This is a deliberate choice.

Designing a workflow DSL involves a large number of subtle decisions:

- how to represent control flow
- how to model data and state
- how to handle retries and timeouts
- how to express parallelism and error handling

These problems have already been explored by a broader community.

By aligning with an existing specification, Zigflow benefits from:

- a well-thought-out model
- familiarity for users already aware of the spec
- reduced risk of introducing inconsistent or ad hoc behaviour

It also avoids the trap of inventing a custom DSL that solves immediate problems
but creates long-term limitations.

Zigflow is opinionated in how it maps the specification to Temporal, but it
builds on a foundation that is already widely understood.

## Two different models

There are effectively two approaches:

### Code-first (Temporal SDK)

- maximum flexibility
- full access to Temporal primitives
- requires understanding determinism and SDK constraints

### Declarative-first (Zigflow)

- constrained, opinionated model
- workflows defined as data
- validation before execution
- reduced cognitive load

Trying to combine these into a single system often leads to a confusing middle
ground.

Zigflow chooses to stay firmly in the second model.

## The trade-off

Not generating code means:

- you cannot access every SDK feature directly
- you are limited to the constructs exposed by the DSL

That is intentional.

Zigflow trades flexibility for:

- predictability
- consistency
- earlier feedback through validation
- a simpler execution model

If you need full control over Temporal, writing workflows in code is the better
choice.

## Where this matters

The distinction becomes important in scenarios like:

- allowing non-developers to define workflows
- building internal platforms
- generating workflows dynamically
- enforcing consistent patterns across teams

In these cases, a declarative system with guardrails is often more useful than a
fully flexible one.

## Final thought

Zigflow is not a thin wrapper around the Temporal SDK.

It is a different way of defining workflows on top of Temporal.

If your goal is flexibility, use the SDK.

If your goal is structure, validation and declarative workflows, Zigflow is
designed for that.

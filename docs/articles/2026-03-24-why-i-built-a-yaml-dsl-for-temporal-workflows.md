---
title: Why I built a YAML DSL for Temporal workflows
description: Why I built Zigflow, a YAML-based DSL for Temporal workflows, and the trade-offs involved.
slug: why-i-built-a-yaml-dsl-for-temporal-workflows
authors: mrsimonemms
---
Temporal is a powerful platform for building reliable, long-running workflows.

But the real trigger for Zigflow wasn't frustration with the SDK.

It was conversations.

In the space of a week, I spoke to three different customers who all said the same
thing:

> We love Temporal, but we want to allow our non-technical users to define workflows.

In practice, there are usually two answers in that situation:

1. use the SDK
2. build your own DSL

I thought the Temporal community should provide a third option. Here's one approach.

{/*truncate*/}

## The idea

Temporal workflows are typically written in code. That's powerful, but it also
creates friction.

- Understanding the SDK
- Understanding determinism
- Deploying and maintaining versioning
- Requiring engineers to define workflows

For many use cases, that's overkill.

So the idea behind Zigflow was simple:

> What if workflows were defined as data instead of code?

Zigflow is a declarative DSL built on top of Temporal.

Instead of writing workflows in Go, Java or TypeScript, you define them in YAML
using a structure inspired by the [CNCF Serverless Workflow specification](https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md).

Zigflow then compiles that definition into a Temporal workflow implementation.

## The hard part: determinism

Temporal requires workflows to be deterministic.

That means:

- no uncontrolled side effects
- no hidden randomness
- no behaviour that changes on replay

In code, this is easy to get wrong.

Zigflow takes a different approach:

- workflows are declarative
- side effects are always modelled as activities
- validation happens before execution

This shifts the problem from:

> don't make mistakes in code

to:

> don't allow invalid constructs in the first place

## Trade-offs

Zigflow is intentionally opinionated.

It trades flexibility for:

- predictability
- consistency
- faster development

This means it's not suitable for every use case.

If you need:

- full SDK control
- highly dynamic runtime behaviour
- deep customisation

you should use the Temporal SDK directly.

## Where it fits

Zigflow works best when workflows are:

- orchestration-focused
- structured
- repeatable
- defined as configuration

It's particularly useful for:

- internal platforms
- automation systems
- tools that generate workflows dynamically
- scenarios where non-developers need to define behaviour

## The result

Zigflow provides:

- declarative workflow definitions
- validation before execution
- compilation into deterministic Temporal workflows
- Kubernetes-native deployment
- support for multi-language activity workers

And importantly, it enables new workflows (no pun intended), including visual
editors and higher-level tooling.

## Closing thoughts

Temporal is incredibly powerful.

Zigflow doesn't replace it.

It builds on top of it.

If you want a more straightforward way to build durable workflows, Zigflow may
be worth exploring.

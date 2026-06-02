---
title: "Zigflow vs Temporal SDK"
sidebar_position: 9
description: "How Zigflow compares to a Temporal SDK, what each approach optimises for and when to choose declarative YAML workflows over SDK code."
---

This page helps you decide whether to use Zigflow or a Temporal SDK directly.
Both produce durable workflows on Temporal, but they target different
audiences and make different trade-offs.

Zigflow is built on top of Temporal and shares the same execution model. This
page is guidance on when to use a Temporal SDK directly and when to use a
higher-level abstraction like Zigflow.

## What Zigflow is

Zigflow is a workflow worker that reads YAML, validates it and runs it on
Temporal. You define a workflow as data using a structure inspired by the
[CNCF Serverless Workflow specification](https://serverlessworkflow.io).
Zigflow compiles that definition into a Temporal workflow at startup and
registers a worker against a task queue.

Workflows are defined as data rather than application code. Validation runs
before execution, and unsupported constructs are rejected with a clear error
rather than failing at runtime.

## What a Temporal SDK is

A Temporal SDK is a library for writing workflows and activities in code.
Temporal is typically used through one of its language SDKs. You import the SDK,
write a workflow function, register it with a worker and deploy it.

The SDK exposes the full Temporal feature set, including signals, queries,
child workflows, sagas, side effects and continue-as-new. Developers are
responsible for preserving determinism and handling workflow evolution as their
code changes over time.

## Key differences

| Aspect | Zigflow | Temporal SDK |
| --- | --- | --- |
| Workflow definition | YAML, declarative | Native code (Go, Java, TypeScript, Python, .NET) |
| Primary audience | Platform engineers and authors of workflow configuration | Application engineers writing Temporal logic |
| Determinism | Enforced structurally by the DSL and validation | Enforced by SDK conventions and code review |
| Validation | Runs before execution, fails fast on invalid constructs | Detected at runtime or through tests |
| Coverage | Subset of CNCF Serverless Workflow plus Zigflow extensions | Full Temporal API surface |
| Build pipeline | None. The Zigflow binary loads YAML at startup | Compile, package and deploy your worker |
| Customisation | Constrained to the DSL | Full programmatic flexibility |
| Authoring tools | Editable as configuration. Suitable for non-developers with review | Requires the toolchain of the chosen language |

## When to use Zigflow

Zigflow is a good fit when:

- Workflows are orchestration-focused, structured and repeatable.
- You want non-developers to author workflow definitions, with strong
  validation as a guardrail.
- You are building an internal platform and want consistent patterns across
  teams.
- You are generating workflow definitions dynamically and want to execute
  them directly, without a code-generation pipeline.
- You value early failure through validation over runtime flexibility.

In these cases the constraints of the DSL are useful. Workflows are easier
to review, easier to validate and easier to share between technical and
non-technical contributors.

## When to use a Temporal SDK

Choose a Temporal SDK directly when:

- You need the full Temporal feature set, including primitives that Zigflow
  does not expose.
- Your workflows contain complex branching, dynamic behaviour or runtime
  decisions that are awkward to express in YAML.
- You need to integrate workflows tightly with application code, libraries
  or in-process state.
- Your team is comfortable enforcing determinism through code review and
  SDK conventions.
- You require deep customisation of activity execution, interceptors or
  low-level Temporal options.

The SDKs are the canonical way to build on Temporal and give the most
control. Zigflow is not a replacement.

## Summary

Use a Temporal SDK when you need flexibility and full access to Temporal
primitives. Use Zigflow when you want declarative workflows with structural
validation and a shorter path from definition to execution. Both run on
Temporal. Both produce durable workflows. They differ in who writes the
workflow, what the authoring surface looks like and where mistakes are
caught.

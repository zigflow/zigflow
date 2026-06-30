# Contributing to Zigflow

Thanks for your interest in **Zigflow**.

Zigflow is a community-maintained DSL for Temporal. It aims to be
stable, predictable and production-ready. Contributions are welcome, but
changes should align with the long-term design goals of the project.

If you are planning a significant change, please open an issue first to
discuss it before submitting a pull request.

---

## Project philosophy

Zigflow is intentionally opinionated. Its goal is not to expose every capability
of the underlying Temporal SDK, but to make Temporal more accessible and easier
to get started with by providing a clear, declarative DSL for workflow orchestration.

One of the core design principles is to _make it easy to do the right thing_.
Where there are multiple ways to achieve the same outcome, Zigflow will generally
favour the approach recommended by Temporal. Features that encourage workarounds,
obscure workflow behaviour or make workflows harder to reason about are unlikely
to be included, even if they are technically possible.

This means that new features are evaluated not only on whether they can be
implemented, but also on whether they align with Zigflow's philosophy and fit
naturally within the DSL. The aim is to build a language that is consistent,
maintainable and production-ready, rather than exposing every feature of the
underlying SDK.

---

## Naming

The project name is **Zigflow**.

Use this exact capitalisation in documentation, comments and user-facing
text.

Lowercase `zigflow` is only acceptable in CLI commands, package names or
URLs.

---

## Getting started

The recommended development environment is a [Dev Container](https://code.visualstudio.com/docs/devcontainers/containers).
Opening the repository in a Dev Container provides a fully configured
environment.

Before submitting a pull request:

- Ensure the project builds successfully
- Ensure all tests pass
- Add or update tests where behaviour changes
- Update documentation if needed

---

## Development principles

Zigflow aims to:

- Provide a clear and expressive DSL for Temporal workflows
- Maintain backwards compatibility wherever possible
- Avoid unnecessary abstraction or complexity
- Prefer explicit behaviour over implicit magic

Breaking changes should be rare and must be discussed in advance.

---

## Specification compatibility

Zigflow is influenced by and aims to remain broadly compatible with the
[ServerlessWorkflow.io](https://serverlessworkflow.io) specification.

Where possible, new features and behavioural changes **should** align
with the Open Workflow Specification (formerly Serverless Workflow) to
minimise conceptual friction and improve interoperability.

However, Zigflow is designed specifically for Temporal. In cases where
the Open Workflow Specification does not map cleanly to Temporal's
execution model, determinism requirements or workflow semantics, Zigflow
may diverge.

When proposing changes that affect compatibility:

- Clearly explain how the proposal relates to the Open Workflow
  Specification
- Identify any areas of divergence
- Justify deviations based on Temporal's design constraints

Compatibility is a goal, but not at the expense of correctness,
determinism or clarity within Temporal.

---

## How to contribute

1. Fork the repository
1. Create a feature branch
1. Make focused, well-scoped changes
1. Add or update tests
1. Open a pull request with a clear description of the change and its
   motivation

Pull requests that introduce new features should explain:

- The problem being solved
- Why this approach was chosen
- How it impacts existing users

Large architectural changes without prior discussion may not be
accepted.

---

### Issue assignment policy

To keep issues moving, Zigflow generally does **not** assign issues to contributors.

If you're interested in working on an issue:

- For straightforward fixes, feel free to open a pull request without asking to
  be assigned.
- For larger changes, especially those affecting the DSL or public API, please
  discuss the proposed approach first.

Comments such as "assign me" don't reserve an issue. This helps avoid situations
where work is unintentionally blocked if someone's plans change.

Contributions are always welcome. The goal is simply to encourage discussion
first where it matters and keep the project open to everyone.

---

## Commit style

All commits must follow the [Conventional
Commits](https://www.conventionalcommits.org) format:

    <type>[optional scope]: <description>

    [optional body]

    [optional footer(s)]

Common types include:

- `feat`
- `fix`
- `docs`
- `refactor`
- `test`
- `chore`

Use `!` for breaking changes and document them clearly.

---

## Code of conduct

Please read and follow our [Code of Conduct](./CODE_OF_CONDUCT.md).

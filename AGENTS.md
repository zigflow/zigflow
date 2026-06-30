# Zigflow agent instructions

This repository contains **Zigflow**, a workflow engine built on top of
**Temporal** and the **CNCF Open Workflow Specification (formerly Serverless Workflow)**.

Zigflow prioritises:
- Determinism
- Correctness
- Explicit validation
- Clear, inspectable behaviour

Treat Zigflow as **infrastructure**, not an application UI. These instructions
apply to any agent working in this repository.

---

## Naming convention

The project name is **Zigflow**, capitalised as shown, with a lowercase 'f'.

Do not write:
- `ZigFlow` (incorrect capitalisation)
- `zigflow` (only acceptable in lowercase-only contexts such as CLI commands,
  package names, or URLs)
- `Zig Flow` (two words)

Always use **Zigflow** in documentation, code comments, error messages, and any
user-facing text.

---

## Core intent

Zigflow exists to:
- Load workflow definitions (YAML / JSON)
- Validate them rigorously
- Execute them deterministically on Temporal
- Surface errors early and explicitly

The system is designed to fail fast on invalid or unsupported constructs.

---

## Technology stack (authoritative)

- **Language:** Go
- **Workflow engine:** Temporal
- **Workflow spec:** CNCF Open Workflow Specification
- **Configuration:** YAML / JSON
- **CLI:** Cobra + Viper
- **Logging:** zerolog
- **Runtime expressions:** jq-style expressions evaluated at runtime

Do not:
- Introduce non-deterministic behaviour into workflows
- Bypass Temporal constraints
- Add hidden side effects
- Assume this is a general-purpose workflow runner

---

## Architectural principles

- Workflow definitions are **data**, not code
- Validation happens **before execution**
- Unsupported workflow features must error explicitly
- Deterministic execution is non-negotiable
- Clarity beats flexibility

Prefer:
- Explicit errors over silent behaviour
- Validation over permissiveness
- Small, composable functions

---

## Validation philosophy

Validation is a **first-class feature**.

Validation should:
- Be deterministic
- Produce human-readable error messages
- Fail early
- Clearly state what is unsupported vs invalid

If a construct is not implemented:
- It should be rejected clearly
- The error message should be actionable

---

## Workflow handling rules

- Workflow schemas are validated separately from execution
- Structural validation must reject unsupported task types
- Runtime expressions must be parsed and evaluated predictably
- Output and context mutation must be explicit

Do not:
- Add support for new Open Workflow Specification task types without discussion
- Infer semantics not present in the spec or implementation
- Relax validation to "make things work"

---

## State management rules

- State (`$context`, `$data`, `$output`) must have explicit and predictable
  lifecycle boundaries
- Task implementations must not leak internal state into parent workflow state
  unless explicitly defined
- Loop constructs (e.g. `for`) must clearly separate:
  - intra-iteration state
  - inter-iteration state
  - parent workflow state
- Implicit state propagation is not allowed
- State promotion to parent workflow state must be explicit and occur only
  through defined task-level mechanisms (e.g. output, export)

---

## Temporal-specific constraints

- Workflow code must remain deterministic
- No reliance on wall-clock time inside workflows
- Side effects belong in activities, not workflows
- Activity inputs and outputs must be serialisable

Do not:
- Introduce randomness
- Use non-replay-safe operations in workflows
- Leak Temporal internals into user-facing abstractions

---

## CLI and configuration behaviour

- CLI flags must be explicit and discoverable
- Defaults should be safe and conservative
- Validation should run by default where possible
- Errors should fail the command, not warn

Avoid:
- Implicit behaviour based on environment
- Magic configuration
- Overloaded flags

---

## Code style and structure

- Prefer explicit types and structs
- Keep functions focused and testable
- Avoid deep nesting where possible
- Name things for clarity, not brevity

Go-specific guidance:
- Avoid cleverness
- Prefer standard library solutions
- Be explicit about error handling

---

## Linting and testing

Run these after making changes:
- `pre-commit run` for pre-commit checks
- `go test ./...` for unit tests
- `task e2e` for end-to-end tests

A task is not complete until unit tests, end-to-end tests, linting, and
validation all pass.

Do not disable linters or add `nolint` directives without discussion.

If pre-commit is not installed, run: `pip install pre-commit && pre-commit install`

Key checks that will run include:
- `golangci-lint` for Go code quality
- `go vet` for Go correctness
- `gofumpt` for Go formatting
- `go-err-check` for error handling
- `go-static-check` for static analysis
- YAML/JSON validation
- Markdown linting
- License header checks
- Trailing whitespace and end-of-file fixes

---

## Documentation expectations

- Documentation is part of the product
- Examples should reflect real, supported behaviour
- Unsupported features should be clearly documented as such
- Avoid aspirational or speculative docs

Docs should explain:
- What Zigflow does
- What Zigflow explicitly does not do
- How it maps to Temporal concepts

---

## Documentation drift and behavioural changes

Documentation represents the **intended and supported behaviour** of Zigflow.

When behaviour changes, determine whether the change is:

1. **Intentional (feature, improvement, or breaking change)**
   - Update documentation to reflect the new behaviour.
   - Ensure examples, CLI output, validation messages, and deployment guidance
     remain accurate.
   - If the change is breaking, ensure documentation clearly reflects the new
     contract.

2. **Unintentional (regression or accidental behaviour change)**
   - Do NOT update documentation to match the regression.
   - Fix the implementation to restore documented behaviour.
   - If uncertain whether behaviour is correct, stop and ask.

Documentation must never be altered solely to match incorrect or unintended
behaviour.

If code and documentation disagree:
- Determine whether the implementation or the documentation reflects the
  intended contract.
- Tests, validation logic, and architectural principles are authoritative
  indicators.
- If ambiguity exists, ask before proceeding.

No change is complete unless:
- Behaviour and documentation are aligned.
- Examples validate successfully.
- Validation messages reflect actual output.
- Determinism and correctness guarantees remain intact.

If a behavioural change affects public contracts, treat it as a versioning
decision.

---

## Writing style

- Do not use em dashes. Use full stops, commas, or sentence restructuring
  instead.
- Use British English spelling and punctuation.
- Prefer clear, direct sentences over stylistic punctuation.
- Avoid dramatic or marketing-style prose.
- Maintain a clean, technical documentation tone.

---

## Scope and uncertainty

Unless explicitly instructed otherwise, assume:
- This is a **workflow engine**, not a framework
- Users value predictability over flexibility
- Features should be added cautiously
- Backwards compatibility matters

Do not:
- Invent new product features
- Assume future roadmap decisions
- Add abstractions "just in case"

When asked to build or modify functionality:
- Start with the smallest correct change
- Preserve existing behaviour
- Prefer incremental improvements

If something is unclear:
- Ask before expanding scope
- Prefer rejecting unsupported behaviour
- Leave TODOs with context rather than guessing

Correctness beats convenience. Determinism beats expressiveness.

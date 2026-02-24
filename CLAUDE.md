# Claude Instructions (zigflow)

This repository contains **Zigflow**, a workflow engine built on top of
**Temporal** and the **CNCF Serverless Workflow specification**.

Zigflow prioritizes:
- Determinism
- Correctness
- Explicit validation
- Clear, inspectable behavior

Claude should treat Zigflow as **infrastructure**, not an application UI.

---

## Naming convention

The project name is **Zigflow** — capitalised as shown, with a lowercase 'f'.

Do not write:
- `ZigFlow` (incorrect capitalisation)
- `zigflow` (only acceptable in lowercase-only contexts such as CLI commands,
  package names, or URLs)
- `Zig Flow` (two words)

Always use **Zigflow** in documentation, code comments, error messages, and any
user-facing text.

---

## Core intent of the project

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
- **Workflow spec:** CNCF Serverless Workflow
- **Configuration:** YAML / JSON
- **CLI:** Cobra + Viper
- **Logging:** zerolog
- **Runtime expressions:** jq-style expressions evaluated at runtime

Do not:
- Introduce non-deterministic behavior into workflows
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
- Explicit errors over silent behavior
- Validation over permissiveness
- Small, composable functions

---

## Workflow handling rules

- Workflow schemas are validated separately from execution
- Structural validation must reject unsupported task types
- Runtime expressions must be parsed and evaluated predictably
- Output and context mutation must be explicit

Claude should not:
- Add support for new Serverless Workflow task types without discussion
- Infer semantics not present in the spec or implementation
- Relax validation to “make things work”

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

## Temporal-specific constraints

- Workflow code must remain deterministic
- No reliance on wall-clock time inside workflows
- Side effects belong in activities, not workflows
- Activity inputs and outputs must be serializable

Do not:
- Introduce randomness
- Use non-replay-safe operations in workflows
- Leak Temporal internals into user-facing abstractions

---

## CLI & configuration behavior

- CLI flags must be explicit and discoverable
- Defaults should be safe and conservative
- Validation should run by default where possible
- Errors should fail the command, not warn

Avoid:
- Implicit behavior based on environment
- Magic configuration
- Overloaded flags

---

## Code style & structure

- Prefer explicit types and structs
- Keep functions focused and testable
- Avoid deep nesting where possible
- Name things for clarity, not brevity

Go-specific guidance:
- Avoid cleverness
- Prefer standard library solutions
- Be explicit about error handling

Linting & Testing:
- Always run pre-commit tests after making changes: `pre-commit run`
- Always run unit tests after making changes: `go test ./...`
- Always run end-to-end tests after making changes:
  1. Start the mock HTTP server in the background: `task http_mock`
  2. Run the e2e tests: `task e2e`
  3. Stop the mock HTTP server process
- All unit tests must pass before considering a task complete
- All end-to-end tests must pass before considering a task complete
- All linting and validation errors must be fixed before considering a task complete
- Key tests that will run include:
  - `golangci-lint` for Go code quality
  - `go vet` for Go correctness
  - `gofumpt` for Go formatting
  - `go-err-check` for error handling
  - `go-static-check` for static analysis
  - YAML/JSON validation
  - Markdown linting
  - License header checks
  - Trailing whitespace and end-of-file fixes
- Do not disable linters or add nolint directives without discussion
- If pre-commit is not installed, run: `pip install pre-commit && pre-commit install`

---

## Documentation expectations

- Documentation is part of the product
- Examples should reflect real, supported behavior
- Unsupported features should be clearly documented as such
- Avoid aspirational or speculative docs

Docs should explain:
- What Zigflow does
- What Zigflow explicitly does not do
- How it maps to Temporal concepts

---

## Creative scope & default assumptions

Unless explicitly instructed otherwise, assume:

- This is a **workflow engine**, not a framework
- Users value predictability over flexibility
- Features should be added cautiously
- Backwards compatibility matters

Claude should not:
- Invent new product features
- Assume future roadmap decisions
- Add abstractions “just in case”

When asked to build or modify functionality:
- Start with the smallest correct change
- Preserve existing behavior
- Prefer incremental improvements

---

## When unsure

If something is unclear:
- Ask before expanding scope
- Prefer rejecting unsupported behavior
- Leave TODOs with context rather than guessing

Correctness beats convenience.
Determinism beats expressiveness.

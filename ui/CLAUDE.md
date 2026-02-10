# Claude Instructions (zigflow.dev visual editor)

This repository contains a **visual flow editor for Zigflow**, built using:

- **SvelteKit**
- **TypeScript**
- **Svelte Flow**
- **App-defined SCSS (no general-purpose CSS framework)**
- **SvelteKit Node adapter**
- **Docker (production runtime)**

The UI is an implementation detail.
The **Flow Graph model (IR) is the source of truth**.

Claude should prioritize correctness, clarity, and alignment with existing
tooling over novelty or abstraction.

---

## Core principles

- Treat the flow graph / IR as authoritative
- UI state must be serializable and reproducible
- Prefer explicit, boring data structures over clever abstractions
- Optimize for correctness and debuggability over minimal code
- Developer experience matters — keep things understandable
- Assume this will run long-lived in Docker

---

## Technology stack (authoritative)

- **Framework:** SvelteKit
- **Adapter:** `@sveltejs/adapter-node`
- **Language:** TypeScript (required; no new JS-only files)
- **Graph / canvas:** Svelte Flow
- **Styling:** App-defined SCSS (no general-purpose CSS framework)
- **Runtime:** Node.js (Docker)
- **Linting:** ESLint (flat config)
- **Formatting:** Prettier (project-defined config)

Do not:

- Introduce React or React-specific patterns
- Introduce general-purpose CSS frameworks (Bulma, Tailwind, Bootstrap, etc.)
- Introduce utility-first CSS systems
- Bypass TypeScript with `any` unless unavoidable and documented
- Add alternative formatters or linters

---

## Runtime & deployment assumptions

- The app runs as a **Node server**, not a static site
- Docker is the primary production environment
- File system access must be treated as ephemeral
- All state must be serializable (DB, object storage, or exportable files)

Do not:

- Rely on in-memory state for persistence
- Assume serverless constraints
- Introduce adapter-specific hacks

---

## Project structure guidance

This project uses a small number of **conceptual root directories**.
Their purpose matters more than their exact contents.

### `src/lib/tasks/`

Authoritative flow graph model and schema.

- Node and edge types
- Graph structure and invariants
- Validation logic
- No UI code
- Must be usable from non-UI contexts (CLI, tests, exporters)

### `src/lib/export/`

Exporters and code generation.

- Pure functions only
- Accept validated graph models
- No UI dependencies
- Deterministic output

### `src/lib/ui/`

Reusable UI primitives and editor-specific components.

- Svelte components only
- Styling via app-owned SCSS
- May depend on Svelte Flow
- No graph semantics or validation logic

### `src/routes/`

SvelteKit routes and server endpoints.

- Page composition
- Load/save/export endpoints
- No business logic
- No graph validation or mutation logic

### `src/routes/workflows/<workflowId>/`

Workflow-specific editor routes.

- Each workflow is addressed by a stable `workflowId`
- Routes under this path are responsible for:
  - Loading workflow state
  - Rendering the editor UI
  - Invoking validation and export logic
- Workflow identity must come from the route, not inferred from UI state

Do not:

- Introduce generic `shared`, `common`, or `utils` directories
- Place domain logic under `routes`
- Couple graph logic to Svelte components
- Encode workflow identity implicitly in component state

### `src/styles/`

Application-owned SCSS.

- Design tokens (spacing, colours, typography)
- Layout helpers actually used by the app
- UI component styles
- Svelte Flow-specific styling

---

## Architecture rules

- Separate concerns strictly:
  - **Flow schema & validation** (no UI code)
  - **Exporters / code generation**
  - **UI (Svelte components, Svelte Flow, inspector panels)**
  - **Server routes (load/save/export)**
- Validation must operate on the flow graph, not UI components
- Exported output must never depend on UI-specific state
- UI should adapt to the model, not vice versa

---

## Flow graph model

- Nodes and edges must have **stable, explicit IDs**
- All node types must be explicitly typed
- No implicit behavior inferred from visual layout or position
- Graph must be fully serializable to JSON
- The graph schema must be usable outside the UI (CLI, tests, exporters)

Do not:

- Encode logic in Svelte component state
- Infer control flow from x/y positioning
- Allow invalid graphs to export

---

## Node behavior

Each node type must define:

- Allowed incoming edge count
- Allowed outgoing edge count
- Required configuration fields
- Validation rules specific to that node

Specific rules:

- Condition nodes must label outgoing edges
- Join nodes must validate fan-in count
- Entry node must be explicit and unique

---

## Validation expectations

Before export, the following must be enforced:

- Exactly one entry node
- All nodes reachable from the entry node
- No orphaned edges
- No cycles unless explicitly supported
- Node-specific structural constraints enforced

Validation errors must be:

- Deterministic
- Human-readable
- Mapped to node IDs where possible
- Suitable for UI display

---

## UI & styling guidelines

### Svelte + Svelte Flow

- Svelte Flow is the canonical canvas abstraction
- Use **Svelte Flow’s native CSS** for graph rendering
- Do not reimplement or override core Svelte Flow layout behavior
- Inspector panels edit **node config**, not graph topology
- UI must reflect validation state clearly
- Avoid auto-magic graph rewrites

---

### UI primitives & styling approach

This project does **not** use a general-purpose CSS framework.

Instead:

- UI styling is defined explicitly in app-owned SCSS
- Common UI elements are implemented as small, reusable Svelte components
- Styling exists to support correctness and clarity, not visual experimentation

Core expectations:

- Define a small set of UI primitives (buttons, inputs, panels, modals)
- Prefer composition over large, configurable components
- Keep styles predictable and easy to audit
- Avoid hidden behaviour encoded in CSS
- Layout should be explicit in Svelte components

Do not:

- Recreate a full component framework
- Introduce large sets of generic utility classes
- Encode application logic in CSS
- Add theming or design-token abstractions prematurely

---

### SCSS rules

- All custom styles must be written in **SCSS**
- Prefer variables, mixins, and nesting over repetition
- SCSS should express structure and intent, not act as a component framework
- Keep Svelte Flow styling separate from app UI styles
- Avoid global CSS leakage where possible

Encouraged:

- A small number of well-named SCSS entry points
- Clear separation between:
  - Design tokens (spacing, colours, typography)
  - Layout helpers actually used by the app
  - App-specific component styles
  - Svelte Flow-related styles

---

## Performance guidelines (important)

This editor must remain responsive for **medium-to-large graphs**.
Performance regressions are considered correctness issues.

### Graph & state management

- Treat the Flow Graph as immutable at the conceptual level
- Prefer targeted updates over full graph replacement
- Avoid deep cloning unless necessary
- Do not recompute validation on every minor UI interaction
- Debounce or batch expensive operations (validation, export)

---

### Svelte-specific guidance

- Avoid unnecessary reactive statements on large collections
- Be explicit about reactivity boundaries
- Prefer derived values over duplicated state
- Do not bind large objects directly to form inputs

---

### Deprecated APIs and patterns

**CRITICAL: Never use deprecated APIs or patterns.**

This applies to all dependencies and frameworks in the project.

Core principles:

- Use current, non-deprecated APIs for all new code
- If a deprecation warning appears, fix it immediately
- Do not ignore, suppress, or work around deprecation warnings
- Follow the official migration path for deprecated features
- Prefer the recommended pattern over backwards-compatible approaches

When in doubt:

- Check TypeScript hints and IDE warnings
- Consult official documentation for the relevant library or framework
- Use the pattern that produces no warnings
- Ask

---

### Svelte Flow usage

- Do not recreate node or edge arrays unnecessarily
- Avoid excessive custom node re-rendering
- Keep custom node components lightweight
- Avoid DOM-heavy node templates
- Be cautious with large numbers of edge labels

---

### CSS & layout performance

- Avoid expensive global selectors
- Minimize deeply nested SCSS rules
- Do not animate layout-affecting properties unnecessarily
- Prefer transform-based animations when needed

---

### Server & Docker considerations

- Server routes should be stateless
- Avoid long-running synchronous operations
- Export and validation should be fast and deterministic
- Assume multiple concurrent users in production

---

## Formatting & linting (do not fight the tools)

### Prettier

This project uses a **strict, predefined Prettier configuration**.

Key expectations:

- Single quotes
- Trailing commas
- 80-character line width
- Sorted imports
- Proper Svelte formatting via `prettier-plugin-svelte`

Claude should not introduce formatting changes that contradict Prettier output.

**IMPORTANT: Before completing any work, always run `npm run format` and
`npm run lint` to ensure all files are properly formatted. This is mandatory.**

The `npm run lint` command includes:

- Prettier code formatting check
- ESLint for JavaScript/TypeScript code
- Markdown linting (markdownlint-cli2) for all .md files

**IMPORTANT: After fixing format and lint issues, run `npm run dev` to verify
the application starts without errors. Fix any runtime errors before considering
the work complete. This is mandatory.**

---

### ESLint

- ESLint flat config is authoritative
- TypeScript + Svelte rules are enabled
- `no-undef` is intentionally disabled
- Svelte files are type-checked via `typescript-eslint`

**IMPORTANT: Before completing any work, always run `npm run lint` to ensure
all code passes linting. This is mandatory.**

Do not:

- Suppress lint rules casually
- Introduce alternative lint configs
- Disable rules without justification

Fix code, not rules.

---

## TypeScript rules

- TypeScript is mandatory for all new code
- Prefer explicit types at module boundaries
- Avoid `any`; if used, explain why in a comment
- Shared graph types must live outside UI components

---

## Export / output

- Export formats must be stable and versioned
- No UI-only metadata in exported output
- Exporters should be pure functions
- Prefer deterministic ordering for diff-friendliness

---

## Testing expectations

- Flow validation logic must be unit tested
- Exporters must have golden-file tests
- UI tests should focus on interaction and behavior, not layout

---

## Dependency management

- Prefer latest stable versions of dependencies
- Avoid unnecessary dependencies
- Document any major dependency decisions

---

## When unsure

If requirements are ambiguous:

- Ask before introducing new node semantics
- Prefer extending the schema over special cases
- Leave TODOs with context rather than guessing

Clarity beats speed.
Correctness beats cleverness.

---

## Creative scope & default assumptions

Unless explicitly instructed otherwise, assume:

- This project is a **developer-facing tool**, not a consumer product
- The goal is **clarity and correctness** over visual polish
- Default outputs should be:
  - Minimal but complete
  - Explicit rather than abstract
  - Easy to extend later

Claude should not:

- Invent product features or workflows without being asked
- Add UX affordances “just in case”
- Introduce configuration surfaces prematurely
- Assume multi-user or collaborative features by default

When asked to “build” something:

- Start with the smallest viable, end-to-end slice
- Prefer scaffolding that can grow over finished-looking systems

---
sidebar_position: 2
---
# Extending the DSL

Zigflow's DSL is based on the
[Open Workflow Specification (formerly Serverless Workflow)](https://serverlessworkflow.io).
The mechanism on this page is the **only** sanctioned way to add
semantics that the specification does not cover, and it exists for a
narrow purpose: bridging the gap between an Open Workflow Specification task
and a Temporal SDK feature that the spec cannot describe. Anything
broader belongs somewhere else.

The canonical example is `wait.until`. The Open Workflow Specification's
wait task is duration-only, but Temporal exposes deterministic
absolute-time sleep through `workflow.Sleep(until - workflow.Now())`.
The extension exposes that Temporal capability without leaving the
spec for unrelated reasons.

## What you will learn

- When the extension mechanism is the right tool, and when it isn't
- The five-piece pattern for adding a new extension
- How to test an extension across schema, normalisation, parsing and
  the workflow runtime

:::warning
**This mechanism is not for custom activities, new keywords, or
arbitrary business logic.** It is for extending existing Open Workflow
Specification task types with a small, surgical Temporal SDK capability the
spec cannot model. See [When NOT to use an extension](#when-not-to-use-an-extension)
below for the precise gate. If you are unsure, open an issue before
writing code.
:::

---

## When to use an extension

Use the extension mechanism when **all three** of the following are true:

1. **The Open Workflow Specification SDK can't accept the YAML you need.**
   Either the task isn't in the SDK at all, or the SDK's schema is too
   strict for the use case (for example, rejecting a runtime expression
   where the value needs to come from workflow state at execution time).
2. **The behaviour maps to a Temporal workflow primitive, not an
   activity.** Extensions exist to expose Temporal features the spec
   can't describe at the YAML level: durable timers, signals, queries,
   continue-as-new. If the behaviour fits inside an activity, write the
   activity and call it from a regular `call activity` task.
3. **The behaviour can only be expressed at the workflow-function
   level.** Anything that can sit behind a `call activity` boundary
   should. An extension is justified when access to Temporal's
   workflow-side APIs (deterministic clock, durable timer, signal
   channels) is the whole point.

The wait extension qualifies on all three: the SDK rejects `until` and
expression-valued duration fields (rule 1), the implementation uses
`workflow.Sleep` and `workflow.Now` directly (rule 2), and the
deterministic-timer behaviour cannot move into an activity without
breaking determinism (rule 3).

Attach the extension to an existing Open Workflow Specification task name
(`wait`, `set`, `for`, `try`, ...) rather than inventing a new top-level
keyword. The mechanism supports new names, but doing so adds DSL
vocabulary outside the spec and has no project precedent yet. Raise an
issue first if a new task name is the right answer.

## When NOT to use an extension

The extension mechanism is explicitly **not** for any of the
following. Use the indicated alternative instead.

| You want to... | Use this instead |
| --- | --- |
| Run a custom activity (`updateUser`, `sendEmail`, ...) | The `call: activity` task with `name:` and `taskQueue:`. Activities live in your Temporal workers and are invoked via the existing DSL; they are not part of the Zigflow DSL surface. See [activity call](/docs/dsl/tasks/call#activity). |
| Add a top-level keyword for business logic (`zigflowMail`, `doMagic`) | A custom activity via `call: activity`. Adding a brand-new top-level task name is technically possible via the extension mechanism, but only justified when the new name also maps to a Temporal workflow primitive (see rule 2 above). Business-logic names belong in activities, not in the DSL. |
| Add sidecar configuration to an existing task (retry options, heartbeat config, schedule details) | Task `metadata`. The Open Workflow Specification endorses `metadata` as the open extension point. See [activity options](/docs/dsl/metadata/activity-options) for the established pattern. |
| Rename a YAML key the SDK already accepts to a friendlier user-facing form | The normalise step. It rewrites user-facing keys to spec keys before the SDK parses; no extension is needed. |
| Embed arbitrary application logic in the workflow | Nothing inside the DSL. Move the logic to an activity. Zigflow's job is to declaratively drive Temporal, not to host business logic. |

---

## Why a dedicated mechanism is needed

The Open Workflow Specification Go SDK is strict about the shapes it accepts.
For example, the SDK's `Duration` type rejects unknown fields and
requires numeric values to be integers. Writing `wait.until` or
`wait.seconds: ${...}` directly fails JSON unmarshalling before
Zigflow's runtime code ever sees the workflow.

The extension mechanism lets Zigflow register its own Go type with
the SDK's task registry under an internal task-type key (prefixed
`__zigflow_ext_`). A normalise step then redirects the user-facing
key (`wait:`) to that internal key when the body uses Zigflow-only
semantics. Vanilla forms still flow through the SDK's own task type
unchanged.

---

## The pattern at a glance

Each extension consists of five pieces:

1. A **Go type** that implements `model.Task`. Lives in `pkg/zigflow/models/`.
2. A **registration** in `init()` via `extensions.RegisterExtension`,
   which performs both the SDK-side and Zigflow-side registrations.
3. **Claim logic** that decides whether the user's YAML uses the
   extension form or the spec form.
4. **Schema updates** in [github.com/zigflow/schema](https://github.com/zigflow/schema)
   so the new user-facing YAML form passes validation.
5. **A task builder** in `pkg/zigflow/tasks/` that converts the parsed
   task into a Temporal workflow function and calls the relevant
   Temporal SDK feature.

The wait extension is the canonical example. Refer to these files when
in doubt:

- `pkg/zigflow/models/wait_ext.go` (type, registration, claim)
- `pkg/zigflow/tasks/task_builder_wait_ext.go` (builder)
- `pkg/zigflow/tasks/task_builder.go` (factory wiring)
- [`github.com/zigflow/schema/definitions.go`](https://github.com/zigflow/schema/blob/main/definitions.go)
  (`waitTaskDefinition`)
- `pkg/zigflow/extensions/extension.go` (registry interface)

---

## Step 1: define the Go type

The Go type holds the unevaluated user input. Embed `model.TaskBase`
for the standard task fields (`if`, `input`, `output`, `metadata`, ...)
and add the extension body under the internal task-type key. The
struct tag on the body field must use the literal internal key
(prefixed `__zigflow_ext_`).

```go
// pkg/zigflow/models/mytask_ext.go
package models

import "github.com/serverlessworkflow/sdk-go/v3/model"

type MyExtTask struct {
    model.TaskBase `json:",inline"`
    Body *MyExtBody `json:"__zigflow_ext_mytask" validate:"required"`
}

func (m *MyExtTask) GetBase() *model.TaskBase { return &m.TaskBase }

type MyExtBody struct {
    NewField string `json:"newField,omitempty"`
    // Use `any` for fields that may be a literal or a runtime expression.
    Count any `json:"count,omitempty"`
}
```

:::tip
Use `any` for fields that may carry either a literal value or a
runtime expression string. Evaluation happens later in the builder.
:::

---

## Step 2: register the extension

`extensions.RegisterExtension` performs both registrations in one
call: the SDK side (so the SDK constructs the Zigflow Go type when it
sees the renamed key) and Zigflow's normalise-side extension registry.

```go
// pkg/zigflow/models/mytask_ext.go
import (
    "github.com/serverlessworkflow/sdk-go/v3/model"
    "github.com/zigflow/zigflow/pkg/zigflow/extensions"
)

func init() {
    extensions.RegisterExtension(myExtension{}, func() model.Task {
        return &MyExtTask{}
    })
}
```

:::warning
Registration must happen at `init()` time. Duplicate task types panic
at init, matching the Open Workflow Specification SDK's own behaviour for
task-type collisions.
:::

---

## Step 3: claim the user's YAML when it uses the extension

The extension type tells Zigflow which Open Workflow Specification task type
it extends and when it should take ownership. The Zigflow internal
key is derived automatically by prefixing the task type with
`extensions.ZigflowExtKeyPrefix`, so the extension only declares its
task type.

```go
// pkg/zigflow/models/mytask_ext.go
type myExtension struct{}

func (myExtension) TaskType() string { return "mytask" }

func (myExtension) Claims(body any) bool {
    m, ok := body.(map[string]any)
    if !ok {
        return false
    }
    // Claim only when the body uses Zigflow-only fields. The vanilla
    // form must continue to flow through the SDK unchanged.
    _, hasNewField := m["newField"]
    return hasNewField
}
```

The loader runs `extensions.Normalise` against every task body. If
`Claims` returns true, the loader rewrites the task-body key from the
task type to `ZigflowExtKeyPrefix + task type` before the SDK sees
the JSON. The SDK then constructs your `*MyExtTask` directly.

---

## Step 4: extend the schema

Update the relevant task definition in [`schema/definitions.go`](https://github.com/zigflow/schema/blob/main/definitions.go)
so the user-facing YAML form validates. The user always writes the
Open Workflow Specification task type (`mytask:` in this example); the
internal `__zigflow_ext_*` key never appears in user-facing input.

The wait task uses a `OneOf` between the existing duration form and
a new until form:

```go
// https://github.com/zigflow/schema/blob/main/definitions.go (inside waitTaskDefinition's AllOf[1].Properties)
"wait": {
    Title: "WaitTaskConfiguration",
    OneOf: []*jsonschema.Schema{
        waitDurationWithExpressionsDefinition,
        waitUntilDefinition,
    },
},
```

:::tip
Prefer extending the local task schema over editing shared definitions
like `durationDefinition`. Shared definitions are referenced by many
consumers (activity timeouts, retry intervals, ...) and widening them
silently changes contracts elsewhere.
:::

After changing the schema, regenerate `docs/static/schema.json` via
pre-commit or by running:

```sh
go run . schema --output json > docs/static/schema.json
```

---

## Step 5: add a task builder

The builder turns the parsed extension task into the Temporal workflow
function that actually runs. This is where the **Temporal SDK feature
the extension exposes** lives: `workflow.Sleep`, `workflow.Now`,
`workflow.SideEffect`, signal handling, and so on. If the builder
doesn't call into the Temporal SDK in some way the spec couldn't
already express, the extension is not justified.

```go
// pkg/zigflow/tasks/task_builder_mytask_ext.go
package tasks

import (
    "github.com/zigflow/zigflow/pkg/utils"
    "github.com/zigflow/zigflow/pkg/zigflow/models"
    "go.temporal.io/sdk/workflow"
    "github.com/serverlessworkflow/sdk-go/v3/model"
)

type MyExtTaskBuilder struct {
    builder[*models.MyExtTask]
}

func (t *MyExtTaskBuilder) Build() (TemporalWorkflowFunc, error) {
    return func(ctx workflow.Context, _ any, state *utils.State) (any, error) {
        // Clone the body, evaluate any ${ ... } expressions against state,
        // then call into the Temporal SDK feature the extension exposes.
        // See task_builder_wait_ext.go for the wait extension's full pattern.
        ...
    }, nil
}
```

Then wire the new case into `pkg/zigflow/tasks/task_builder.go`'s
`NewTaskBuilder` switch:

```go
// pkg/zigflow/tasks/task_builder.go (inside NewTaskBuilder's switch)
case *models.MyExtTask:
    return NewMyExtTaskBuilder(temporalWorker, t, taskName, doc, emitter, taskOpts)
```

And add the interface-assertion entry at the bottom of the same file:

```go
// pkg/zigflow/tasks/task_builder.go (inside the interface-assertion var block)
_ TaskBuilder = &MyExtTaskBuilder{}
```

:::info
The SDK's struct-level validator gates on a hardcoded list of task
types and reports `unknown_task` for anything else. Zigflow overrides
that validator in `pkg/utils/validation.go` so any task that
implements `model.Task` and passes its own struct-tag validation is
accepted. You do not need to touch the validator when adding a new
extension.
:::

---

## Testing an extension

:::tip
For Zigflow's wider testing approach, see
[Testing workflows](/docs/guides/testing-workflows).
:::

Follow the wait extension's test layout. There are five layers worth
covering:

1. **Schema** ([schema](https://github.com/zigflow/schema)):
   positive and negative cases for the new YAML forms, plus a structural
   test that asserts the definition shape (see `TestWaitTaskDefinitionShape`
   for the wait extension's equivalent).
2. **Normalise** (`pkg/zigflow/normalise_test.go`): the claim logic
   correctly rewrites the task body key for extension forms and leaves
   vanilla forms alone.
3. **Model** (`pkg/zigflow/models/mytask_ext_test.go`): JSON round-trip,
   the type satisfies `model.Task`, the SDK registry returns your type
   for the internal key.
4. **Builder** (`pkg/zigflow/tasks/task_builder_mytask_ext_test.go`):
   use `testsuite.WorkflowTestSuite` to drive the workflow function
   with controlled state and assert behaviour against `env.Now()`.
5. **End-to-end** (`tests/e2e/tests/mytask-<scenario>/`): one folder
   per scenario, each with a `workflow.yaml` and `test.go`, exercising
   the full load, normalise, parse and run pipeline against a real
   Temporal environment.

---

## Common mistakes

:::warning
**Adding an extension that doesn't map to a Temporal SDK feature.**
The extension mechanism is for exposing Temporal capabilities the
spec cannot describe. If your new field or shape doesn't ultimately
call into the Temporal SDK in some meaningful way, you're using the
wrong tool. Reconsider whether the use case is actually an activity,
a `metadata` sidecar, or no change at all.
:::

:::warning
**Claiming vanilla forms.** Your `Claims` predicate must return false
for the spec form so the SDK constructs its own task type. Returning
true unconditionally hides the SDK's behaviour and forces every
workflow through your extension, including ones that did not opt in.
:::

:::warning
**Editing shared schema definitions.** Widening `durationDefinition`
to permit runtime expressions sounds tempting, but it changes the
contract for every consumer that references it (activity timeouts,
retry intervals, schedule). Extend the local task schema instead.
:::

:::warning
**Forgetting to regenerate `docs/static/schema.json`.** The published
schema must be derived from [zigflow/schema](https://github.com/zigflow/schema).
The pre-commit hook regenerates it automatically; if you skip pre-commit,
run `go run . schema --output json > docs/static/schema.json`.
:::

---

## Related pages

- [Wait](/docs/dsl/tasks/wait): the canonical extension, documented as
  a user-facing task.
- [Call activity](/docs/dsl/tasks/call#activity): the right mechanism
  for running custom logic on your Temporal workers.
- [Activity options](/docs/dsl/metadata/activity-options): the
  metadata-sidecar extension pattern for tasks that do not need new
  semantics.
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs):
  the load pipeline that extensions plug into.
- [Concepts: data and expressions](/docs/concepts/data-and-expressions):
  how `${ ... }` expressions are evaluated at workflow time.
- [Testing workflows](/docs/guides/testing-workflows): the broader
  testing approach extensions slot into.

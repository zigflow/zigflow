<!-- markdownlint-disable MD041 -->
[![Zigflow](./designs/zigflow.png "Zigflow")](https://zigflow.dev?utm_source=github&utm_medium=readme&utm_campaign=header)

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/zigflow/zigflow/build.yml?style=flat&label=Build)](https://github.com/zigflow/zigflow/actions/workflows/build.yml)
[![GitHub Release](https://img.shields.io/github/v/release/zigflow/zigflow?label=Release)](https://github.com/zigflow/zigflow/releases/latest)
[![GitHub Repo stars](https://img.shields.io/github/stars/zigflow/zigflow?style=flat&label=Stars)](https://github.com/zigflow/zigflow)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/zigflow/zigflow/total?label=Downloads)](https://github.com/zigflow/zigflow/releases)
[![Licence](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/zigflow/zigflow/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/zigflow/zigflow)](https://goreportcard.com/report/github.com/zigflow/zigflow)
[![Docs](https://img.shields.io/badge/Docs-zigflow.dev-blue)](https://zigflow.dev)

**Declarative YAML workflows on Temporal. No SDK boilerplate.**

Zigflow lets you define and run [Temporal](https://temporal.io) workflows using
YAML, built on the [CNCF Serverless Workflow](https://serverlessworkflow.io)
specification. You write a workflow definition; Zigflow compiles it into a
fully-featured Temporal workflow with retries, state management and
deterministic execution. No Go, Java or TypeScript workflow code required.

> If this looks useful, a ⭐ helps others find the project.

---

## Quick start

Run your first workflow in a few minutes and see the result.

1. Install Zigflow

   ```sh
   brew tap zigflow/tap
   brew install --cask zigflow
   ```

   For other options, see [installation docs](https://zigflow.dev/docs/getting-started/installation).

2. Start a Temporal server

   > Requires the [Temporal CLI](https://docs.temporal.io/cli)

   ```sh
   temporal server start-dev
   ```

   The Temporal UI will be available at <http://localhost:8233>.

3. Create a workflow

   Save this as `workflow.yaml`:

   ```yaml
   document:
     dsl: 1.0.0
     taskQueue: zigflow
     workflowType: hello-world
     version: 1.0.0
   do:
     - greet:
         output:
           as:
             data: ${ . }
         set:
           message: Hello from Ziggy
   ```

4. Run it

   ```sh
   zigflow run -f workflow.yaml
   ```

5. Trigger the workflow

   ```sh
   temporal workflow start \
     --type hello-world \
     --task-queue zigflow \
     --workflow-id my-first-workflow

   temporal workflow result \
     --workflow-id my-first-workflow
   ```

   You should see the workflow output in the CLI.

### Next steps

- [Documentation](https://zigflow.dev/docs)
- [Examples](https://zigflow.dev/docs/examples)
- [CLI reference](https://zigflow.dev/docs/cli/commands/zigflow_run)

⭐ Star the repo if this was useful

---

## Why Zigflow exists

Writing Temporal workflows in code is powerful, but it comes with overhead:
learning the SDK, wiring up workers and keeping workflow code deterministic.
Zigflow provides an opinionated declarative layer on top of Temporal, built
around proven workflow patterns to simplify common orchestration use cases
while retaining Temporal’s reliability and execution model.

---

## How it compares

Zigflow is an opinionated, declarative layer on top of Temporal. It trades some
SDK-level flexibility for faster development and consistent workflow structure.

| Capability | <img src="./designs/temporal.png#gh-light-mode-only" alt="Temporal SDKs" height="50px" /><img src="./designs/temporal-dark.png#gh-dark-mode-only" alt="Temporal SDKs" height="50px" /> | Custom DSL | <img src="./designs/zigflow.png " alt="Zigflow" height="50px" /> |
| --- | :--: | :--: | :--: |
| **Workflow definition model** | Imperative code | Declarative | Declarative |
| **Control and flexibility** | Maximum | Depends on implementation | Opinionated and constrained |
| **Retries and durability** | ✅ Native Temporal | ⚠️ You build it | ✅ Native Temporal |
| **Continue-as-new** | ✅ Supported (you implement and tune it) | ⚠️ You build it | ✅ Automatic (Zigflow continues as new for you) |
| **CNCF spec alignment** | ❌ | ❌ Usually bespoke | ✅ Serverless Workflow v1.0+ |
| **Validation before execution** | Manual or custom | Depends on implementation | ✅ Built-in validation |
| **Boilerplate and worker setup** | Required | Depends on implementation | Minimal |
| **Kubernetes deployment** | Manual | Manual | ✅ Helm chart included |
| **Multi-language activities** | ✅ | Depends on implementation | ✅ Any Temporal SDK |

Use the SDK when you need maximum flexibility. Use Zigflow when you want
consistent, declarative orchestration with less boilerplate.

---

## Who is Zigflow for?

Zigflow is designed for teams who value speed, consistency and declarative
orchestration over maximum SDK flexibility.

- **Teams who want to ship workflows faster**: Zigflow removes SDK boilerplate
  and worker wiring, so you can define, validate and run workflows in minutes.
- **Platform teams building an internal orchestration layer**: centralise workflow
  execution without requiring every team to learn or embed a Temporal SDK.
- **Teams who prefer opinionated, declarative guardrails**: Zigflow constrains
  workflow structure in useful ways, reducing accidental complexity.
- **Workflow-as-configuration adopters**: store, version and review workflow
  definitions as data rather than application code.
- **Organisations already using Temporal** who want a simpler path for
  straightforward orchestration without sacrificing reliability.
- **Tool builders and automation platforms** that need a spec-compliant
  execution engine behind a UI or higher-level abstraction.

---

## When not to use Zigflow

- **You need advanced Temporal workflow APIs and fine tuning**: if you rely on
  nuanced child workflow behaviour, rich per-call options, or patterns that are
  easier to express directly in code, the Temporal SDK gives you the most
  flexibility.
- **You are building a deeply customised Temporal platform layer**: if your
  architecture depends heavily on SDK-level interceptors, custom middleware, or
  tight framework integrations, an SDK-first approach provides the broadest
  control surface.
- **Your workflows are highly dynamic at runtime**: if the structure of the
  workflow is generated as it runs, or you are effectively building an execution
  engine for arbitrary user logic, a code-first workflow is often a better fit.
- **Your team prefers workflow development as application code**: if stepping
  through workflow logic in a debugger, refactoring with compiler guarantees,
  and unit testing workflow functions directly are central to your development
  process, the SDK model may feel more natural.
- **Your workflow contains substantial domain logic rather than orchestration**:
  Zigflow excels at the declarative orchestration of activities and services. If
  most of your business logic lives inside the workflow itself, with complex
  in-memory decision making or algorithmic behaviour, writing workflows directly
  in an SDK may be clearer.

In those cases, use Temporal directly. Zigflow is an opinionated layer on top of
Temporal, not a replacement for its full SDK capabilities.

---

## How it works

1. You define a workflow using YAML based on the Serverless Workflow spec.
2. Zigflow validates the definition before execution.
3. Zigflow compiles the definition into a Temporal workflow implementation.
4. A worker is started for the configured namespace and task queue.
5. Activities are executed by your existing Temporal workers.

Zigflow handles orchestration while your services execute the work.

---

## Examples

### Send a monthly email

Sends an email every 30 days for 12 months using an HTTP endpoint.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: send-email
  version: 0.0.1
do:
  - for:
      for:
        in: ${ 12 } # repeat 12 times (once per month)
      do:
        - sendEmail:
            call: http
            with:
              method: POST
              endpoint: https://example.com/send-email
              headers:
                Content-Type: application/json
              body:
                to: user@example.com
                subject: Hello
                message: Hello from Zigflow
        - wait30Days:
            wait:
              days: 30
```

```bash
zigflow run -f ./workflow.yaml
```

Zigflow starts the worker. You can then trigger the workflow from any
[Temporal SDK](https://docs.temporal.io/encyclopedia/temporal-sdks), the
[Temporal UI](https://docs.temporal.io/web-ui#workflow-actions) or the
[Temporal CLI](https://docs.temporal.io/cli/workflow#start).

- [**Task Queue**](https://docs.temporal.io/task-queue): `zigflow`
- [**Workflow Type**](https://docs.temporal.io/workflows#intro-to-workflows): `send-email`

---

### Activity call

<details>
<summary><strong>Activity call example (click to expand)</strong></summary>

> Runnable version: [examples/activity-call](./examples/activity-call)

Call external Temporal activities from a declarative definition. Input is
validated against a JSON Schema before the workflow runs.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: activity-call
  version: 0.0.1
  metadata:
    activityOptions:
      startToCloseTimeout:
        minutes: 1
input:
  schema:
    format: json
    document:
      type: object
      required:
        - userId
      properties:
        userId:
          type: string
do:
  - captureInput:
      set:
        requestedUserId: ${ $input.userId }
        requestId: ${ uuid }
  - fetchProfile:
      call: activity
      with:
        name: activitycall.FetchProfile
        arguments:
          - ${ $data.requestedUserId }
          - ${ $data.requestId }
        taskQueue: activity-call-worker
  - generateWelcome:
      call: activity
      with:
        name: activitycall.GenerateWelcome
        arguments:
          - ${ $data.fetchProfile }
        taskQueue: activity-call-worker
  - finalize:
      set:
        workflowId: ${ $data.requestId }
        profile: ${ $data.fetchProfile }
        message: ${ $data.generateWelcome.message }
```

Activity workers can be written in any language with a Temporal SDK: Go,
Python, TypeScript, Java and more. Zigflow handles the workflow orchestration;
your existing services handle the work.

> If you find these examples useful, consider giving the repo a ⭐. It helps
> the project grow.

</details>

---

More examples (child workflows, fork/fan-out, signals, queries, try-catch,
schedules and more) are in [examples/](./examples).

---

## Features

- **Temporal DSL**: declarative YAML definitions that compile to Temporal workflows
- **CNCF standard**: aligned with Serverless Workflow v1.0+
- **Validation first**: definitions are validated before execution; invalid or
  unsupported constructs are rejected with actionable errors
- **Multi-language activities**: activity workers can use any Temporal SDK
- **Low-code and visual-ready**: suitable for UI workflow builders and
  orchestration tools
- **Kubernetes-native**: Helm chart included for cluster deployments
- **Open source**: Apache 2.0 licence, contributions welcome

---

## ⚡ What's with the name?

Zigflow is named after Ziggy, [Temporal's official mascot](https://temporal.io/blog/temporal-in-space).

Ziggy is a tardigrade, a microscopic animal that is
[basically indestructible](https://en.wikipedia.org/wiki/Environmental_tolerance_in_tardigrades).
Sound familiar?

Also, the colours are based on the Ziggy Stardust lightning bolt.

---

## Telemetry

Telemetry helps the maintainers understand whether Zigflow is being used in
real production environments. No personal or identifiable user data is collected.

When a worker starts, Zigflow sends:

- an anonymous installation ID (generated locally on first run, or derived from
  the container hostname)
- the Zigflow version
- basic runtime information (OS, architecture, container detection)
- approximate server country (2-letter code, derived once at startup)

The country value is derived once at startup and is not tied to any identity. No
IP addresses are stored.

When workflows are executed, Zigflow sends a periodic heartbeat (once per minute)
containing:

- the total number of workflow runs since the worker started
- the worker uptime in seconds

Heartbeats are only sent when the run count changes. Idle workers do not
emit repeated telemetry.

Zigflow does **not** collect:

- workflow definitions
- workflow inputs or outputs
- execution IDs
- task names
- hostnames
- environment variable values
- organisation identifiers
- IP addresses or precise location data

Telemetry exists solely to understand real-world adoption and usage.

**Opting out** is straightforward:

```sh
# Environment variable
DISABLE_TELEMETRY=true

# CLI flag
--disable-telemetry
```

---

## Project status

Zigflow is under active development and evolving steadily.

- The core execution engine and validation pipeline are stable.
- The DSL schema follows semantic versioning.
- Kubernetes deployment via Helm is supported.
- Backwards compatibility is maintained within major versions.
- Zigflow is actively used in internal tooling and early adopter environments.

Zigflow is designed for long-term use as the declarative orchestration layer on
top of Temporal.

Bug reports, feedback and contributions are welcome.

---

## Related projects

- [Temporal](https://temporal.io)
- [CNCF Serverless Workflow](https://serverlessworkflow.io)
- [Helm chart](./charts/zigflow)

---

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md) for
guidelines on naming conventions, dev setup and commit style.

---

## Contributors

<a href="https://github.com/zigflow/zigflow/graphs/contributors">
  <img alt="Contributors"
    src="https://contrib.rocks/image?repo=zigflow/zigflow&v=1769433596" />
</a>

Made with [contrib.rocks](https://contrib.rocks/preview?repo=zigflow%2Fzigflow).

[![Star History Chart](https://api.star-history.com/svg?repos=zigflow/zigflow&type=date&legend=top-left)](https://www.star-history.com/#zigflow/zigflow&type=date&legend=top-left)

---

## Licence

Distributed under the [Apache-2.0](./LICENSE) licence.

© 2025 - 2026 [Zigflow authors](https://github.com/zigflow/zigflow/graphs/contributors)

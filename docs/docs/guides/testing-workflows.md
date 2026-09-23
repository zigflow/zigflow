---
sidebar_position: 1
---
# Testing Workflows

Zigflow workflows are declarative YAML definitions. They do not contain
application code and there is no workflow SDK layer to unit test. The most
reliable way to verify that a workflow behaves as intended is to run it.

The recommended approach is integration testing: start a Temporal environment,
start a Zigflow worker with your workflow definitions, execute the workflows
with controlled inputs, and verify their outputs and behaviour.

## What you will learn

- How to run a Zigflow workflow against a local Temporal instance for testing
- How to structure integration tests for workflow behaviour
- How to apply the same approach in a CI pipeline
- How to test long-running workflows without waiting for real-world durations

---

## Testing Temporal workflows with Zigflow

Zigflow runs workflows on Temporal, so testing Zigflow workflows is
effectively testing Temporal workflow executions.

Instead of unit testing workflow code, Zigflow workflows are validated by
running them in a Temporal environment and verifying their behaviour.

This approach tests the real workflow lifecycle including retries, timers,
signals and task execution, mirroring production behaviour.

---

## Integration testing workflows

Integration tests execute workflow definitions against a real Temporal
environment. They validate the full execution path: YAML parsing,
validation, workflow execution in Temporal, task execution and output.

### Local setup

A typical local integration test environment requires three things:

1. A running Temporal server
2. A Zigflow worker with your workflow definitions loaded, **or** a one-shot
   `zigflow test` run (see below)
3. A script or test runner that starts workflow executions and checks results

**Start a local Temporal development server:**

```sh
temporal server start-dev
```

#### Option A: one-shot test with `zigflow test`

`zigflow test` validates a workflow file, starts a short-lived in-process
worker, runs the workflow once with JSON input, prints the result and exits.
It is suited to local checks and CI scripts when you do not need a long-lived
worker.

```sh
zigflow test workflow.yaml --input testdata/input.json
```

:::warning
`zigflow test` always uses the Temporal task queue `zigflow-test`. The
`document.taskQueue` value in the YAML is not used for registering the test
worker or for starting the run. That avoids competing with a separate
`zigflow run` process on the same queue. Tasks inside the workflow that
specify their own task queues are unchanged.
:::

Pass inline JSON with `--input-json` when a file is inconvenient. Use
`--timeout` for long-running workflows (shorten `wait` durations in test YAML
where possible; see [Testing long-running workflows](#testing-long-running-workflows)).

The public Go package `github.com/zigflow/zigflow/pkg/testrunner` exposes the
same orchestration for tests written in Go. Treat it as experimental until
documented as a stable API.

On failure, `zigflow test` still exits with a non-zero status and prints
`Failed` or `Timeout`, but it also prints an `output` section when structured
failure data is available (for example Open Workflow Specification errors from
`raise`). The Go `Result.Output` field is populated the same way. A short
`error:` line and Temporal inspect link are printed after the output when the
run did not complete successfully.

#### Option B: long-lived worker and Temporal CLI

**Start the Zigflow worker in a separate terminal:**

```sh
zigflow run -f workflow.yaml
```

**Start a workflow execution:**

```sh
temporal workflow start \
  --type my-workflow \
  --task-queue zigflow \
  --workflow-id test-run-1 \
  --input '{"name": "test"}'
```

**Inspect the result:**

```sh
temporal workflow show --workflow-id test-run-1
```

### Example

The following workflow calls an HTTP endpoint and returns the response:

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: fetch-user
  version: 0.0.1
do:
  - fetchUser:
      call: http
      with:
        method: GET
        endpoint: https://api.example.com/users/1
```

To test it, start the worker and trigger an execution. Then use
`temporal workflow show` or the Temporal Web UI to confirm the workflow
completed and that the output matches the expected structure.

Integration tests validate real workflow behaviour including automatic retries
on activity failure, conditional branching via `switch`, parallel execution via
`fork`, external event handling via `listen` and durable timer behaviour via
`wait`.

---

## End-to-end testing in CI

The same integration testing approach applies in CI pipelines. A typical CI
job follows these steps:

1. Start Temporal (using the [Temporal CLI](https://docs.temporal.io/cli),
   [Docker](https://hub.docker.com/r/temporalio/temporal) or the [GitHub Action](https://github.com/temporalio/setup-temporal))
2. Run workflows with test inputs (`zigflow test` or a long-lived worker plus
   a client)
3. Verify that workflow outputs match expected values
4. Exit with a non-zero code if any assertion fails

**Example CI steps using `zigflow test`:**

```sh
# Start Temporal in the background
temporal server start-dev &

# Run the workflow once; non-zero exit on failure
zigflow test workflow.yaml --input testdata/input.json --timeout 10m
```

**Example with a background worker and scripts:**

```sh
# Start Temporal in the background
temporal server start-dev &

# Start the Zigflow worker in the background
zigflow run -f workflow.yaml &

# Run test scripts
./scripts/run-integration-tests.sh
```

Use exit codes to signal pass or fail. If a workflow execution produces an
unexpected result or the worker exits with an error, the CI job should fail.

:::tip
The Zigflow repository uses this approach for its own end-to-end tests. See
the project's CI configuration for a working example.
:::

---

## Testing long-running workflows

In production, Temporal workflows can run for hours, days or longer. Waiting
for real-world durations in tests is not practical.

Three techniques keep test execution time short:

### Use signals to drive workflow progress

If a workflow is blocked waiting for an external event using `listen`, send a
signal during the test to advance it. This avoids waiting for a real external
trigger.

```sh
temporal workflow signal \
  --workflow-id test-run-1 \
  --name approve \
  --input '{"approved": true}'
```

This lets you test the full lifecycle of a signal-driven workflow in seconds.

### Modify workflow timers during tests

:::tip
This can be done using tools such as [yq](https://github.com/mikefarah/yq) in
test setup scripts.
:::

If a workflow contains long `wait` durations, tests should modify the workflow
definition so the timers complete quickly.

A common approach is to update the workflow YAML during test setup and restore
it after the test finishes. Most testing frameworks provide lifecycle hooks such
as `beforeEach` and `afterEach` that can be used for this purpose.

For example, a test might:

1. Copy the workflow definition
2. Modify the `wait` duration so timers run for a few seconds
3. Start the Zigflow worker using the modified workflow
4. Run the workflow and verify the result
5. Restore the original workflow definition after the test completes

This keeps the production workflow definition unchanged while allowing
integration tests to run quickly.

The exact implementation depends on the language and testing framework used.
For example:

- TypeScript projects might use [Jest](https://jestjs.io) or [Vitest](https://vitest.dev/)
- Go projects might use the standard `testing` package with [Testify](https://github.com/stretchr/testify)

Tests can modify the workflow definition during `beforeEach` and restore it
during `afterEach`.

### Validate lifecycle stages rather than waiting

For workflows with multiple stages, test each stage independently rather than
running the full path end to end. Verify that each task produces the correct
intermediate output before asserting on the final result.

This is particularly useful when individual tasks involve external services that
are slow or unavailable in test environments.

---

## Testing activities

Activities are the application code that Zigflow calls during workflow
execution. Because activities run outside the deterministic workflow context,
they can be tested using the normal testing tools for your language. This is
standard unit testing and does not require a running Temporal environment.

---

## Common mistakes

**Editing workflow YAML without restarting the worker.**
Zigflow loads workflow definitions when the worker starts. Changes to YAML files
take effect only after the worker is restarted.

**Reusing workflow IDs across test runs.**
If you reuse a workflow ID, Temporal may reject or deduplicate the execution
depending on its reuse policy. Use unique workflow IDs for each test run, or
set `--workflow-id-reuse-policy` when starting executions. `zigflow test`
generates a unique workflow ID for each invocation.

**Running `zigflow test` while `zigflow run` is active.**
Because `zigflow test` uses the `zigflow-test` queue, it does not share the
worker pool with `zigflow run` on `document.taskQueue`.

**Running multiple `zigflow test` commands at once.**
Each invocation registers only its own workflow on the shared `zigflow-test`
queue. Without coordination, Temporal could deliver a task to a worker that
does not have that workflow registered. Zigflow therefore acquires a
per-machine file lock keyed by Temporal address and namespace so only one
`zigflow test` process polls `zigflow-test` at a time for that server and
namespace. Additional invocations wait until the lock is released. The lock
file lives under your user cache directory (for example
`~/.cache/zigflow/test-locks` on Linux). Different namespaces, or different
Temporal servers, do not block each other.

**Hard-coding long timer durations.**
A `wait` task with a duration of hours will block test execution for hours.
Modify timer durations in the workflow definition used for tests so timers
complete quickly.

**Asserting on timing rather than output.**
Prefer asserting on workflow output and task results. Assertions that depend
on exact timing are fragile in CI environments.

---

## Related pages

- [Using the CLI](/docs/cli/using-the-cli): validate, test and run commands
- [How Zigflow Runs](/docs/concepts/how-zigflow-runs): execution model overview
- [Listen task](/docs/dsl/tasks/listen): waiting for external events and signals
- [Wait task](/docs/dsl/tasks/wait): durable timers
- [Signal-Driven Workflow example](/docs/examples/signal-driven): worked example
  using signals
- [Temporal prerequisites](/docs/concepts/temporal-prereqs): Temporal concepts
  used in testing

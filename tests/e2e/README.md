# End-to-end tests

These tests exercise Zigflow against a real Temporal dev server. They are
guarded by the `e2e` build tag and run via:

```sh
task e2e
```

`task e2e` starts the supporting services (Temporal and an HTTP mock) with
Docker Compose, exports their addresses, and runs `go test -tags=e2e
./tests/e2e/...`. This is also what GitHub Actions runs, so any example that
opts in is covered automatically in CI.

## Test sources

There are two sources of test cases, both executed by the same harness:

1. **Bespoke cases** under [`tests/`](./tests). Each registers a
   `utils.TestCase` describing a workflow and how to assert its behaviour. Use
   these when a test needs custom logic.
2. **Example-based cases**, auto-discovered from the repository's
   [`examples/`](../../examples) directory. These need no Go code.

## Example-based testing

An example opts into e2e testing by adding a `test.yaml` file next to its
`workflow.yaml`. Examples without a `test.yaml` are ignored.

The harness:

1. Discovers every `examples/**/test.yaml`.
2. Locates the `workflow.yaml` in the same directory.
3. Runs the workflow with the given input.
4. Asserts the result matches the expected output exactly.

### test.yaml

The schema is intentionally minimal:

```yaml
input: {}

expected:
  data:
    message: Hello from Ziggy
```

- `input` is passed to the workflow when it starts.
- `expected` is compared exactly against the workflow result.

Compose-backed examples, Temporal-direct execution, partial assertions and
advanced metadata are not supported yet.

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
4. Applies the assertions declared in the `test.yaml`.

### test.yaml

A `test.yaml` carries the workflow input and one or both assertion styles:

```yaml
input: {}

# Exact matching: the result must equal this value exactly.
expected:
  data:
    message: Hello from Ziggy
```

- `input` is passed to the workflow when it starts.
- `expected` is compared exactly against the workflow result. Use it when the
  output is fully deterministic.
- `assert` validates the shape and type of the result without requiring exact
  values. Use it for examples that produce variable data such as UUIDs,
  timestamps or generated IDs.

`expected` and `assert` may be used together in the same file. When both are
present the result must satisfy both.

### Structural assertions

An `assert` block mirrors the shape of the result. A node is either a type
assertion (a map with a `type` key) or a nested object whose keys are recursed
into. Assertions are partial, so keys present in the result but absent from the
block are ignored.

```yaml
assert:
  data:
    id:
      type: uuid
    createdAt:
      type: timestamp
    requestId:
      type: non-empty-string
```

The supported assertion types are:

| Type               | Passes when the value is                          |
| ------------------ | ------------------------------------------------- |
| `exists`           | present, whatever its type                        |
| `string`           | a string                                          |
| `non-empty-string` | a string with a length greater than zero          |
| `number`           | a number                                          |
| `boolean`          | a boolean                                         |
| `object`           | a JSON object                                     |
| `array`            | a JSON array                                      |
| `timestamp`        | a string parseable as an RFC 3339 / ISO 8601 time |
| `uuid`             | a string parseable as a UUID                      |

Examples that need a Docker Compose stack, a separate worker, external network
access or external interaction (signals, queries, updates) are out of scope and
do not carry a `test.yaml`.

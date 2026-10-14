# Concurrent test isolation

Fixtures for checking that concurrent `zigflow test` runs use separate
`zigflow-test-<uuid>` task queues without a file lock.

## Prerequisites

```sh
temporal server start-dev
```

From the repository root inside the Dev Container:

```sh
cd /workspaces/zigflow
```

## Run

Start the slow workflow in the background, then start the fast one immediately.
The fast run should complete while the slow run is still waiting.

```sh
(
  time go run . test examples/test-concurrency/slow-workflow.yaml \
    --input examples/test-concurrency/input.json \
    --timeout 3m
) &
slow_pid=$!

sleep 1

time go run . test examples/test-concurrency/fast-workflow.yaml \
  --input examples/test-concurrency/input.json \
  --timeout 3m

wait "$slow_pid"
```

The fast run should return its own output without waiting for the slow
workflow's 45-second timer. Both workflows use the same workflow type but
different definitions, so each result must match its own file. The Temporal
UI should show a different task queue for each execution.

## Validate fixtures only

```sh
go run . validate examples/test-concurrency/slow-workflow.yaml
go run . validate examples/test-concurrency/fast-workflow.yaml
```

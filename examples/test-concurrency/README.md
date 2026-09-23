# Concurrency lock manual check

Fixtures for checking that concurrent `zigflow test` runs serialise on the
`zigflow-test` task queue (file lock per Temporal address and namespace).

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
The fast run should not finish until the slow run has completed.

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

If locking works, the fast command's `time` output shows roughly the slow
workflow duration (about 45 seconds) plus its own run time, not a sub-second
total.

## Validate fixtures only

```sh
go run . validate examples/test-concurrency/slow-workflow.yaml
go run . validate examples/test-concurrency/fast-workflow.yaml
```

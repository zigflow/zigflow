---
sidebar_position: 5
---

# Parallel Tasks

Run two branches concurrently where the first to finish wins.

## What you will learn

- How to run workflow branches in parallel using `fork`
- How `compete: true` returns only the first branch to finish
- How to collect results from all branches using `compete: false`
The slower branch is cancelled.

## Workflow

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: competing-tasks
  version: 0.0.1
timeout:
  after:
    minutes: 1
do:
  - state:
      export:
        as:
          data: ${ . }
      set:
        id: ${ uuid }
  - race:
      fork:
        compete: true
        branches:
          - task1:
              do:
                - getUser:
                    call: http
                    with:
                      method: get
                      endpoint: >-
                        https://jsonplaceholder.typicode.com/users/2
                - wait:
                    wait:
                      seconds: 2
          - task2:
              do:
                - getUser:
                    call: http
                    with:
                      method: get
                      endpoint: >-
                        https://jsonplaceholder.typicode.com/users/1
                - wait:
                    wait:
                      seconds: 50
```

## Explanation

| Part | Purpose |
| --- | --- |
| `fork.compete: true` | First branch to finish cancels the others |
| `fork.branches` | Each branch runs as a child workflow |
| `state` task | Generates a UUID before the race starts |

`task1` finishes in roughly 2 seconds. `task2` would take 50
seconds. Because `compete: true` is set, `task2` is cancelled as
soon as `task1` completes.

**Cancelled branches are not rolled back.** Any HTTP calls or side
effects performed before cancellation are not undone. Only the
winner's output is returned.

## Non-competing fork

To run all branches and collect every result, use `compete: false`
(the default):

```yaml
- parallel:
    fork:
      compete: false
      branches:
        - branch1:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/users/1
        - branch2:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/users/2
```

The output is an object containing each branch's result, keyed by
branch name.

## How to run

1. Start a Temporal development server:

   ```sh
   temporal server start-dev
   ```

2. Start the worker:

   ```sh
   zigflow run -f workflow.yaml
   ```

3. Trigger the workflow:

   ```sh
   temporal workflow start \
     --type competing-tasks \
     --task-queue zigflow \
     --workflow-id race-1
   ```

4. View the result:

   ```sh
   temporal workflow show --workflow-id race-1
   ```

## Common mistakes

**Expecting the cancelled branch output to be available.**
With `compete: true` only the winner's output is returned. Cancelled
branch data is discarded.

---

## Related pages

- [Fork](/docs/dsl/tasks/fork): `fork` reference
- [Do](/docs/dsl/tasks/do): sequential execution within a branch
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs):
  child workflows and execution model

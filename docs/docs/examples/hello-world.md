---
sidebar_position: 2
---

# Hello World

A minimal workflow that sets a single value and returns it.

## What you will learn

- How to structure the `document` header required by every workflow
- How the `set` task stores values in workflow state
- How `output.as` shapes the workflow result

## Workflow

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: hello-world
  version: 0.0.1
do:
  - greet:
      set:
        message: Hello from Ziggy
      output:
        as:
          data: ${ . }
```

## Explanation

| Part | Purpose |
| --- | --- |
| `document.taskQueue` | Sets the Temporal task queue to `zigflow` |
| `document.workflowType` | Sets the Temporal workflow type to `hello-world` |
| `set` | Writes `message` into the workflow state |
| `output.as` | Shapes the return value of this task |

The `set` task runs as a
[Temporal side effect](https://docs.temporal.io/develop/go/side-effects),
so generated values such as `${ uuid }` are determinism-safe here.

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
     --type hello-world \
     --task-queue zigflow \
     --workflow-id hello-1
   ```

4. View the result:

   ```sh
   temporal workflow show --workflow-id hello-1
   ```

## Expected output

```json
{
  "data": {
    "message": "Hello from Ziggy"
  }
}
```

## Common mistakes

**"No workers are registered for this task queue."**
The `--task-queue` value must match `document.taskQueue` in the YAML
file.

**"Workflow type not found."**
The `--type` value must match `document.workflowType` in the YAML file.

---

## Related pages

- [Set](/docs/dsl/tasks/set): the task used here
- [Quickstart](/docs/getting-started/quickstart): guided walkthrough
- [HTTP Call](/docs/examples/http-call): next example

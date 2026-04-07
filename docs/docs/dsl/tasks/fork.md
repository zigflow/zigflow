# Fork

Executes multiple workflow branches concurrently.

## When to use this

Use Fork when multiple operations can run independently and
simultaneously. Set `compete: true` if you only need the result
of the fastest branch and want to cancel the rest.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| fork.branches | [`map[string, task]`](/docs/dsl/tasks/intro) | `no` | The tasks to perform concurrently. These will be run as [child workflows](https://docs.temporal.io/child-workflows). |
| fork.compete | `boolean` | `no` | Indicates whether or not the concurrent [`tasks`](/docs/dsl/tasks/intro) are racing against each other, with a single possible winner, which sets the composite task's output.<br />*If set to `false`, the task returns an array that includes the outputs from each branch, preserving the order in which the branches are declared.*<br />*If to `true`, the task returns only the output of the winning branch.*<br />*Defaults to `false`.* |

## Example

### Non-competing Fork

:::info
This is the default behaviour.
:::

This will return all the workflow.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - raiseAlarm:
      # A fork is a series of child workflows running in parallel
      output:
        # Add output to context without the `multiStep` key
        as:
          raiseAlarm: '${ $context + del(.multiStep) }'
      fork:
        # If not competing, all tasks will run to the finish - this is the default behaviour
        compete: false
        branches:
          # A single step is passed in by the Serverless Workflow task
          - callNurse:
              call: http
              with:
                method: get
                endpoint: https://jsonplaceholder.typicode.com/users/2
          # Multiple steps can be passed in by the Serverless Workflow do task
          - multiStep:
              do:
                - wait1:
                    wait:
                      seconds: 3
                - wait2:
                    wait:
                      seconds: 2
          # Another single step child workflow
          - callDoctor:
              call: http
              with:
                method: get
                endpoint: ${ "https://jsonplaceholder.typicode.com/users/" + ($input.userId | tostring) }

```

This will output an object similar to this, with the workflow data under the
`raiseAlarm` key:

```json
{
  "raiseAlarm": {
    "callDoctor": {
      // The workflow's data
    },
    "callNurse": {
      // The workflow's data
    }
  }
}
```

### Competing Fork

This will return the fastest returning workflow only. Simply change `compete: false`
to `compete: true`. The data will look similar to this:

```json
{
  "raiseAlarm": {
    // The workflow's data
  }
}
```

Unlike the non-competing version, the `raiseAlarm` object will *ONLY* contain
the data of the winning fork. All the other workflows will be cancelled and any
data generated will be discarded.

## Gotchas

**In competing mode, all non-winning branches are cancelled.** Any side effects
(HTTP calls, state changes) performed in cancelled branches are not rolled back.
Only the winner's output is returned.

**Each branch runs as a child workflow.** Errors in a branch propagate back to
the parent fork task and trigger the retry policy unless caught by a `try` task
wrapping the branch.

## Related pages

- [Do](/docs/dsl/tasks/do): sequential execution
- [For](/docs/dsl/tasks/for): iteration over collections
- [Try](/docs/dsl/tasks/try): error handling within a branch
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs):
  child workflows and execution model
- [Examples: parallel tasks](/docs/examples/parallel-tasks):
  competing and non-competing fork in action

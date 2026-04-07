# For

Iterates over a collection and executes a set of tasks for each item.

## When to use this

Use For when you need to run the same set of tasks for every item
in a collection. The collection can be an array, a map or an
integer count.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| for.each | `string` | `no` | The name of the variable used to store the current item being enumerated.<br />Defaults to `item`. |
| for.in | `string` | `yes` | A [runtime expression](/docs/dsl/tasks/intro#runtime-expressions) used to get the collection to enumerate. |
| for.at | `string` | `no` | The name of the variable used to store the index of the current item being enumerated.<br />Defaults to `index`. |
| while | `string` | `no` | A [runtime expression](/docs/dsl/tasks/intro#runtime-expressions) that represents the condition, if any, that must be met for the iteration to continue.<br />The result of each iteration is stored in `$data.<taskName>`, allowing the while expression to conditionally stop the loop. |
| do | [`map[string, task]`](/docs/dsl/tasks/intro) | `yes` | The [task(s)](/docs/dsl/tasks/intro) to perform for each item in the collection. These will be run as a [child workflow](https://docs.temporal.io/child-workflows) |

## Example

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow # Mapped to the task queue
  workflowType: for-loop # Workflow name
  version: 0.0.1
do:
  # Iterate over the map object
  - forTaskMap:
      export:
        as: '${ $context + { forTaskMap: . } }'
      for:
        in: ${ $input.map }
      do:
        - setData:
            export:
              as: ${ . }
            set:
              key: "${ \"hello: \" + $data.index }"
              value: ${ $data.item }
        - wait:
            output:
              as: ${ $context }
            wait:
              seconds: 2
  # Iterate over the data array
  - forTaskArray:
      export:
        as: '${ $context + { forTaskArray: . } }'
      for:
        each: item
        in: ${ $input.data }
        at: index
      # while: ${ $data.item.userId != 4 } # If this returns false, it will cut the iteration
      do:
        # Each iteration will run these tasks in order
        - setData:
            export:
              as: ${ . }
            set:
              userId: ${ $data.item.userId } # Get the userId for this iteration
              id: ${ $data.index } # Get the key
              processed: true
        - wait:
            output:
              as: ${ $context }
            wait:
              seconds: 1
  - forTaskNumber:
      output:
        as: '${ $context + { forTaskNumber: . } }'
      for:
        in: ${ 5 }
      do:
        - setData:
            export:
              as: ${ . }
            set:
              number: ${ $data.item }
        - wait:
            output:
              as: ${ $context }
            wait:
              seconds: 1
```

## Gotchas

**Each iteration runs as a child workflow.** A loop over a large collection
creates many child workflow executions. Temporal's history limits apply.

**The `while` condition is evaluated after each iteration.** It cannot prevent
the first iteration from running.

**Iteration results are stored under `$data.<taskName>`.** The `while`
expression can access this key to implement early termination.

## Related pages

- [Fork](/docs/dsl/tasks/fork): parallel execution of multiple branches
- [Do](/docs/dsl/tasks/do): sequential subtasks
- [Set](/docs/dsl/tasks/set): storing iteration data
- [Concepts: data and expressions](/docs/concepts/data-and-expressions):
  runtime expressions and `$data` variables
- [Examples: parallel tasks](/docs/examples/parallel-tasks):
  running branches concurrently

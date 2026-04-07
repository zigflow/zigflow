# Switch

Enables conditional branching within workflows. Based on a set of conditions
evaluated at runtime, the Switch task directs execution to a different task.

## When to use

Use Switch when you need to route execution to different tasks based on the
value of workflow state or input data.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| switch | [`case[]`](#switch-case) | `yes` | A name/value map of the cases to switch on |

### Switch case {#switch-case}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| when | `string` | `no` | A runtime expression that evaluates to a boolean. If the expression is truthy, this case matches.<br />*If not set, this case is the default and matches when no other case matches.*<br />*Only one default case is allowed.* |
| then | [`flowDirective`](/docs/dsl/tasks/intro#flow-directive) | `yes` | The flow directive to execute when the case matches. |

## How it works

Cases are evaluated in order. The first case whose `when` expression evaluates
to `true` wins. If no case matches, the default case (where `when` is absent)
is used.

After the matching case's `then` directive runs, execution continues from
the target task.

## Example

This workflow routes an order through different processing steps depending
on `$input.orderType`:

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: order-router
  version: 1.0.0
do:
  - routeOrder:
      switch:
        - electronic:
            when: ${ $input.orderType == "electronic" }
            then: processElectronicOrder
        - physical:
            when: ${ $input.orderType == "physical" }
            then: processPhysicalOrder
        - default:
            then: handleUnknownType

  - processElectronicOrder:
      do:
        - validatePayment:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/posts/1
        - fulfillOrder:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/posts/2

  - processPhysicalOrder:
      do:
        - checkInventory:
            call: http
            with:
              method: get
              endpoint: https://jsonplaceholder.typicode.com/posts/3

  - handleUnknownType:
      raise:
        error:
          type: https://serverlessworkflow.io/spec/1.0.0/errors/validation
          status: 400
          title: Unknown order type
          detail: ${ "Received order type: " + $input.orderType }
```

When triggered with `{ "orderType": "electronic" }`, the workflow runs
`processElectronicOrder` and skips the other branches.

## Using flow directives

The `then` property accepts any [flow directive](/docs/dsl/tasks/intro#flow-directive):

```yaml
- classify:
    switch:
      - urgent:
          when: ${ $input.priority == "high" }
          then: escalate       # Jump to a named task
      - empty:
          when: ${ $input.items | length == 0 }
          then: end            # End the workflow immediately
      - default:
          then: continue       # Proceed to the next task
```

## Gotchas

**Cases are evaluated in declaration order.** Place more specific conditions
before broader ones to avoid unintended matches.

**A missing default case is valid but risky.** If no case matches and there
is no default, execution falls through to the next task in the `do` list.
This is rarely intentional. Include a default case to make intent explicit.

**The `then` directive targets a task by name.** The named task must exist
in the same `do` list. Referencing a non-existent task fails at validation.

## Related tasks

- [Do](/docs/dsl/tasks/do): sequential subtasks, commonly used as branch targets
- [Raise](/docs/dsl/tasks/raise): raise an explicit error in a branch
- [For](/docs/dsl/tasks/for): iteration over collections
- [Concepts: data and expressions](/docs/concepts/data-and-expressions):
  runtime expression syntax

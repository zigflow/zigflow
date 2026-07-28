# Switch

Enables conditional branching within workflows. Based on a set of conditions
evaluated at runtime, the Switch task determines the next flow directive to execute.

## When to use

Use Switch when you need to route execution to different tasks based on the
value of workflow state or input data.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| switch | [`case[]`](#switch-case) | `yes` | A name/value map of the cases to switch on |

### Switch case {/*#switch-case*/}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| when | `string` | `no` | A runtime expression that evaluates to a boolean. If the expression is truthy, this case matches.<br />*If not set, this case is the default and matches when no other case matches.*<br />*Only one default case is allowed.* |
| then | [`flowDirective`](/docs/dsl/tasks/intro#flow-directive) | `yes` | The flow directive to execute when the case matches. |

## How it works

Cases are evaluated in order. The first case whose `when` expression evaluates
to `true` wins. If no case matches, the default case (where `when` is absent)
is used.

When a case matches, its `then` flow directive determines what happens next:
execution may continue to the next task, redirect to a named target, exit the
current scope or end the workflow.

### Named redirects

A named `then` on a `switch` **invokes** the named target. The target is
resolved from the workflow's document-wide redirect namespace, so a `switch`
nested inside a `for`, `try` or `fork` can redirect to a target declared in
another scope of the same document. The target runs inline within the current
workflow (it is not a Temporal child workflow), returns its result to the
`switch`, and the `switch` task's own `output` and `export` directives are then
applied to that result.

This is distinct from a task-level `then: <name>` on an ordinary task, which is
same-scope forward navigation: it skips ahead to a named sibling in the current
task list. See [flow directives](/docs/dsl/tasks/intro#flow-directive).

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

When triggered with `{ "orderType": "electronic" }`, the switch selects
`processElectronicOrder` as the next branch to execute.

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

**Named `then` directives redirect to a named target.** The target is resolved
from the document-wide redirect namespace when the redirect runs. Referencing a
target that does not exist fails at runtime with `redirect target not found`,
not at validation time.

**`end` terminates the workflow.** Use `exit` when you only want to leave the
current scope, such as a nested `do` or loop.

## Related tasks

- [Do](/docs/dsl/tasks/do): sequential subtasks, commonly used as branch targets
- [Raise](/docs/dsl/tasks/raise): raise an explicit error in a branch
- [For](/docs/dsl/tasks/for): iteration over collections
- [Concepts: data and expressions](/docs/concepts/data-and-expressions):
  runtime expression syntax

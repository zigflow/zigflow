# Wait

Allows workflows to pause or delay their execution for a specified period of time.
This converts to a Temporal [Durable Timer](https://docs.temporal.io/workflow-execution/timers-delays).

## When to use this

Use Wait to introduce a durable delay into your workflow. The
timer survives worker restarts. Typical uses include cooldown
periods, scheduled actions and pauses between retry attempts.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| wait | [`duration`](/docs/dsl/intro#duration) | `yes` | The amount of time to wait. |

## Example

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - wait:
      wait:
        seconds: 5
```

## Gotchas

**The timer is durable.** A wait of hours or days survives worker restarts.
Temporal holds the timer state. This is intended behaviour.

**There is no maximum duration.** Very long timers are supported by Temporal
but increase workflow history length.

## Related pages

- [For](/docs/dsl/tasks/for): using wait inside iteration loops
- [Listen](/docs/dsl/tasks/listen): waiting for external events instead of
  a fixed duration
- [Concepts: temporal prerequisites](/docs/concepts/temporal-prereqs):
  durable timers explained

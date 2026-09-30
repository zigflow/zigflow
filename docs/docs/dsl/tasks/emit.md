# Emit

Publishes data from a running workflow to a
[Temporal Workflow Stream](https://github.com/temporalio/sdk-go/tree/main/contrib/workflowstreams).
External clients subscribe to the workflow's stream to receive each item as
it is emitted.

## When to use this

Use emit to stream progress or intermediate results back to a caller while the
workflow is still running, rather than waiting for the final output.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| emit.event.with.type | `string` | `yes` | The event type. Must be `io.temporal.streams.<topic>`. |
| emit.event.with.data | `any` | `no` | The payload to publish to the topic. Any JSON value is accepted. [Runtime expressions](/docs/dsl/tasks/intro#runtime-expressions) are evaluated, including inside objects and arrays. If omitted, the item is published with a `null` payload. |

The topic is everything after the exact `io.temporal.streams.` prefix, so
`io.temporal.streams.order.status` publishes to the `order.status` topic.

## Example

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - emitProgress:
      emit:
        event:
          with:
            type: io.temporal.streams.progress
            data:
              step: 1
              message: Started
```

A client reads the stream with the `workflowstreams` package from the Temporal
Go SDK:

```go
stream := workflowstreams.NewClient(c, workflowID, workflowstreams.Options{})

for item, err := range stream.Subscribe(ctx, workflowstreams.SubscribeOptions{
    Topics: []string{"progress"},
}) {
    // Handle item.Topic and item.Data
}
```

The subscription ends when the workflow completes.

## Gotchas

**Only Temporal Workflow Streams are supported.** Any event type that does not
start with `io.temporal.streams.` is rejected when the workflow is loaded. The
prefix is case sensitive and the topic must not be empty.

**Expressions in `data` must be deterministic.** Non-deterministic functions
such as `uuid` are rejected when the workflow is validated. Generate the value
in a [Set](/docs/dsl/tasks/set) task first, then reference it from `data`, for
example `${ $data.id }`.

## Related pages

- [Workflow Streams example](https://github.com/zigflow/zigflow/tree/main/examples/workflow-streams)

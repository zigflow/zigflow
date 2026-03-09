---
sidebar_label: Debugging Workflows
---

# Debugging Workflows

## What you will learn

- How to enable CloudEvents to observe workflow execution in real time
- How to configure file and HTTP CloudEvent clients
- How to interpret event types and event structure
- Common failure scenarios and how to diagnose them

Zigflow emits [CloudEvents](https://cloudevents.io) at key points during
workflow execution to help you understand what's happening inside your
workflows. This is particularly useful during development and debugging.

## Overview

CloudEvents provide real-time visibility into:
- When workflows start and complete
- When individual tasks begin execution
- Task retry attempts
- Task cancellations
- Task failures and faults
- Task completion with output data

## Configuration

To enable CloudEvents emission, provide a configuration file using either:

- **CLI flag**: `--cloudevents-config /path/to/config.yaml`
- **Environment variable**: `CLOUDEVENTS_CONFIG=/path/to/config.yaml`

Example:

```bash
# Using CLI flag
zigflow run --cloudevents-config ./cloudevents.yaml

# Using environment variable
export CLOUDEVENTS_CONFIG=./cloudevents.yaml
zigflow run
```

## Configuration File Format

The CloudEvents configuration file defines one or more clients that receive
events. Each client specifies a protocol (either `http` or `file`) and a target
destination.

```yaml
clients:
  - name: http-output
    protocol: http
    target: "{{ .env.HTTP_TARGET }}"
    disabled: false  # Optional, defaults to false
    options:
      timeout: 5s    # Optional, defaults to 5s
      method: POST   # Optional, defaults to POST

  - name: file-output
    protocol: file
    target: "{{ .env.FILE_TARGET }}"
```

### Configuration Options

#### Client Fields

- **`name`** (required): Unique identifier for this client
- **`protocol`** (required): Currently, either `file` or `http`
- **`target`** (required): Destination for events (URL for http, directory path
  for file)
- **`disabled`** (optional): Set to `true` to temporarily disable this client
  (defaults to `false`)
- **`options`** (optional): Protocol-specific options

#### HTTP Options

- **`timeout`**: Request timeout duration (defaults to `5s`)
- **`method`**: HTTP method to use (defaults to `POST`)

#### Template Variables

The `target` field supports Go template syntax with access to environment variables:

```yaml
target: "{{ .env.HTTP_TARGET }}"  # References $HTTP_TARGET envvar
```

## Event Types

Zigflow emits the following CloudEvent types:

### Workflow Events

- **`dev.zigflow.workflow.started`**: Emitted when a workflow begins execution
  - Contains workflow input data and metadata
- **`dev.zigflow.workflow.completed`**: Emitted when a workflow completes successfully
  - Contains workflow output data and final context

### Task Events

- **`dev.zigflow.task.started`**: Emitted when a task begins execution
  - Contains task name, input data, and current context
- **`dev.zigflow.task.retried`**: Emitted when a task is retried after a failure
  - Contains retry attempt number and error information
- **`dev.zigflow.task.cancelled`**: Emitted when a task is cancelled
  - Contains cancellation reason
- **`dev.zigflow.task.faulted`**: Emitted when a task fails
  - Contains error details and context at time of failure
- **`dev.zigflow.task.completed`**: Emitted when a task completes successfully
  - Contains task output and updated context

### Iteration Events

- **`dev.zigflow.iteration.completed`**: Emitted when a task has iterated over
  some data
  - Contains data relevant to the iteration

## Event Structure

All events follow the
[CloudEvents v1.0 specification](https://github.com/cloudevents/spec/blob/v1.0/spec.md)
and include:

- **`specversion`**: Always `1.0`
- **`type`**: Event type (e.g., `dev.zigflow.task.started`)
- **`source`**: Event source in format `zigflow.dev/<namespace>/<workflow-name>`
- **`id`**: Unique event identifier (UUID)
- **`time`**: Event timestamp (ISO 8601)
- **`datacontenttype`**: Always `application/json`
- **`data`**: Event payload with workflow/task information

## Protocols

:::info
Please raise a [feature request](https://github.com/zigflow/zigflow/issues/new?template=feature_request.md)
for additional protocols.
:::

### File Protocol

The file protocol writes CloudEvents to YAML files on disk, organized by
workflow execution ID.

#### Behavior

- Events are appended to files in the target directory
- Each workflow execution creates a separate file: `<workflowID>.yaml`
- Events are written in YAML format, making them human-readable
- Files persist between workflow runs for post-execution analysis

#### File Configuration

```yaml
clients:
  - name: file-logger
    protocol: file
    target: "./tmp/events"
```

#### Example Output

After running a workflow, you'll find files like:

```text
./tmp/events/
├── 550e8400-e29b-41d4-a716-446655440000.yaml
└── 550e8400-e29b-41d4-a716-446655440001.yaml
```

Each file contains a sequence of CloudEvents in YAML format:

```yaml
---
specversion: "1.0"
type: dev.zigflow.workflow.started
source: zigflow.dev/zigflow/example
id: 550e8400-e29b-41d4-a716-446655440000
time: "2026-02-09T10:30:00Z"
datacontenttype: application/json
data:
  workflowName: example
  input:
    userId: 42
---
specversion: "1.0"
type: dev.zigflow.task.started
source: zigflow.dev/zigflow/example
id: 550e8400-e29b-41d4-a716-446655440001
time: "2026-02-09T10:30:01Z"
datacontenttype: application/json
data:
  taskName: getUser
  context:
    userId: 42
```

### HTTP Protocol

The HTTP protocol sends CloudEvents to an HTTP endpoint via POST requests
(configurable).

#### Example HTTP Receiver (Go)

```go
package main

import (
    "context"
    "log"

    cloudevents "github.com/cloudevents/sdk-go/v2"
)

func main() {
    p, err := cloudevents.NewHTTP()
    if err != nil {
        log.Fatal(err)
    }

    c, err := cloudevents.NewClient(p)
    if err != nil {
        log.Fatal(err)
    }

    log.Fatal(c.StartReceiver(context.Background(), receive))
}

func receive(ctx context.Context, event cloudevents.Event) {
    log.Printf("Received event %s of type %s from %s",
        event.ID(),
        event.Type(),
        event.Source(),
    )

    var data map[string]any
    if err := event.DataAs(&data); err == nil {
        log.Printf("Data: %+v", data)
    }
}
```

#### Example Configuration

```yaml
clients:
  - name: debug-server
    protocol: http
    target: "http://localhost:8080"
    options:
      timeout: 10s
      method: POST
```

## Complete Example

A complete example demonstrating CloudEvents debugging is available in the
repository at
[`examples/cloudevents`](https://github.com/zigflow/zigflow/tree/main/examples/cloudevents).

The example includes:
- A sample workflow with multiple tasks
- CloudEvents configuration for both HTTP and file protocols
- An HTTP receiver implementation
- Docker Compose setup for easy testing

### Running the Example

```bash
cd examples/cloudevents

# Start the workflow worker with CloudEvents enabled
docker compose up workflow

# In another terminal, trigger the workflow
docker compose up trigger

# Watch HTTP events
docker compose logs -f http
```

View file events in the `tmp/events` directory in the root of the project.

## Best Practices

### Development

- Use the **file protocol** for local development to capture all events for
  later analysis
- Use the **http protocol** to integrate with existing observability tools

### Production

- Consider the performance impact of event emission on high-throughput workflows
- Use `disabled: true` to selectively disable clients without removing configuration
- Set appropriate timeouts for HTTP clients to prevent workflow delays
- Monitor undelivered events via Prometheus metrics

### Debugging Workflow Issues

1. **Enable file-based CloudEvents** to capture a complete execution trace
2. **Search for `task.faulted` events** to identify failures
3. **Check `task.retried` events** to understand retry behavior
4. **Compare `task.started` and `task.completed` data** to see how context evolves
5. **Review `workflow.completed` output** to verify final results

## Observability

Zigflow exposes Prometheus metrics for CloudEvents:

- `zigflow_events_emitted_total`: Total events emitted per client and type
- `zigflow_events_undelivered_total`: Events that failed to deliver
- `zigflow_events_emit_duration_seconds`: Time taken to emit events

These metrics help you monitor the health and performance of your event
emission pipeline.

## Limitations

- CloudEvents are emitted on a best-effort basis
- Failed event deliveries do not fail the workflow
- Events are emitted from the workflow execution context and must remain deterministic
- Event payloads are subject to Temporal's payload size limits

---

## Common failure scenarios

### Worker exits immediately after starting

The most common cause is a validation failure. Zigflow validates the workflow
before starting the worker. Run `zigflow validate` to see the error:

```text
❌ Validation failed for workflow.yaml

1 validation error(s):

1. document.name: is required
```

Fix the reported field and restart the worker.

### Non-Determinism Error during workflow replay

Temporal replays workflows from their event history. If a workflow produces
different results on replay. For example, if a UUID or timestamp is
generated outside a `set` task, Temporal raises a Non-Determinism Error.

Check your logs for lines containing `NonDeterministicWorkflowError`. The fix
is to move any generated value into a `set` task, which wraps the generation
in a Temporal side-effect.

See [Data and expressions](/docs/concepts/data-and-expressions) for guidance.

### HTTP activity returns an error

HTTP calls (via `call: http`) fail the task when the server returns a non-2xx
status. The error is visible in the workflow history and in task-faulted
CloudEvents.

To recover without failing the whole workflow, wrap the call in a
[`try` task](/docs/dsl/tasks/try).

### Listen task times out

A `listen` task with `type: signal` or `type: update` will time out if no
matching event arrives within `metadata.timeout` (default 60 seconds). The
task is then marked as failed.

Increase `metadata.timeout` if the signal may take longer to arrive.

### Activity exceeds its schedule-to-close timeout

If an activity (HTTP call, container run, etc.) does not complete within its
configured timeout, Temporal marks it as timed out and schedules a retry
according to the retry policy. Check the workflow history in the Temporal UI
for `ScheduleToClose` timeout events.

Adjust `activityOptions` in the workflow or task metadata to increase the
timeout.

---

## See Also

- [CloudEvents Specification](https://cloudevents.io)
- [CloudEvents SDK for Go](https://github.com/cloudevents/sdk-go)
- [Temporal Observability](https://docs.temporal.io/observability)

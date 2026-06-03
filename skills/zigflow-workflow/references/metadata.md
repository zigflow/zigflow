# Document Metadata Reference

The `document.metadata` object is the extension point for
Zigflow-specific configuration that does not exist in the
Serverless Workflow specification.

## Activity Options

Controls Temporal activity timeouts and retries for the entire
workflow (can be overridden per task via task-level `metadata`).

```yaml
document:
  metadata:
    activityOptions:
      startToCloseTimeout:
        minutes: 1
      heartbeatTimeout:
        seconds: 30
      retryPolicy:
        maximumAttempts: 3
```

### Available fields

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `startToCloseTimeout` | duration | 15s | Max time for a single activity execution |
| `heartbeatTimeout` | duration | none | Timeout between heartbeats |
| `retryPolicy.maximumAttempts` | int | 5 | Max retry attempts |

---

## Schedule Metadata

Required when using the `schedule` top-level key.

```yaml
document:
  metadata:
    scheduleWorkflowName: my-workflow  # required: which workflow to trigger
    scheduleId: my-schedule-id         # optional: custom schedule ID
    scheduleInput:                     # optional: input for the triggered workflow
      - key: value
```

---

## Continue-As-New Override

For testing or tuning the history length threshold that triggers Temporal continue-as-new:

```yaml
document:
  metadata:
    canMaxHistoryLength: 100
```

---

## Search Attributes

Set at the task level (not document level) via task `metadata`:

```yaml
- myTask:
    metadata:
      searchAttributes:
        customField:
          type: int
          value: 42
    set:
      data: hello
```

---

## Heartbeat

Configure activity heartbeat at the task level:

```yaml
- longTask:
    metadata:
      heartbeat:
        seconds: 10
    call: http
    with:
      method: get
      endpoint: https://slow-api.example.com
```

---

## Signal/Query Timeout

For `listen` tasks, set an await timeout:

```yaml
- awaitSignal:
    metadata:
      timeout: 60s
    listen:
      to:
        one:
          with:
            id: my-signal
            type: signal
```

---

## Tags and Display

For the example catalog and documentation generation:

```yaml
document:
  metadata:
    tags:
      - http
      - activity
    display: false  # hide from example listings
```

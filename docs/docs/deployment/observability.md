---
sidebar_position: 5
---

# Observability

## What you will learn

- How to use the health endpoint for liveness and readiness probes
- How to scrape Prometheus metrics from a running worker
- How to emit CloudEvents during workflow execution
- How to control log verbosity

## Health checks

Zigflow exposes an HTTP health endpoint while the worker is running:

```text
GET http://localhost:3000/health
```

Returns `200 OK` when the worker is healthy. Use this for liveness and
readiness probes in any orchestration system.

Change the address with `--health-listen-address` (default `0.0.0.0:3000`).

### Kubernetes

The Helm chart configures liveness and readiness probes automatically. No
additional configuration is needed for standard deployments.

### Docker Compose

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl -f http://localhost:3000/health || exit 1"]
  interval: 10s
  timeout: 5s
  retries: 3
```

---

## Prometheus metrics

Zigflow exposes Prometheus metrics at:

```text
http://localhost:9090/metrics
```

Change the address with `--metrics-listen-address` (default `0.0.0.0:9090`).

Use `--metrics-prefix` to add a prefix to all metric names.

### CloudEvents metrics

| Metric | Labels | Description |
| --- | --- | --- |
| `zigflow_events_emitted_total` | `client`, `type` | Total events emitted per client and event type |
| `zigflow_events_undelivered_total` | `client`, `type` | Events that failed to deliver |
| `zigflow_events_emit_duration_seconds` | `client`, `type` | Emit duration histogram |

These metrics are only populated when a CloudEvents configuration is active.

### Scraping in Kubernetes

When using the Helm chart, add a Prometheus scrape annotation to the pod:

```yaml title="values.yaml"
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

---

## CloudEvents

:::tip
For full configuration options, event structure, file output examples
and debugging guidance, see [Debugging workflows](/docs/dsl/debugging).
:::

Zigflow can emit [CloudEvents v1.0](https://cloudevents.io) at key points
during workflow execution. This is the primary mechanism for real-time
observability into running workflows.

Enable it by providing a configuration file:

```sh
zigflow run -f workflow.yaml --cloudevents-config ./cloudevents.yaml
```

### Configuration file

```yaml title="cloudevents.yaml"
clients:
  - name: file-logger
    protocol: file
    target: ./tmp/events

  - name: http-sink
    protocol: http
    target: "{{ .env.HTTP_EVENTS_URL }}"
    options:
      timeout: 5s
      method: POST
```

The `target` field supports Go template syntax with access to environment
variables via `.env`.

### Supported protocols

| Protocol | Target format | Notes |
| --- | --- | --- |
| `file` | Directory path | Events written as YAML, one file per workflow execution |
| `http` | HTTP URL | Events sent as POST requests |

### Event types

| Event type | Emitted when |
| --- | --- |
| `dev.zigflow.workflow.started` | A workflow execution begins |
| `dev.zigflow.workflow.completed` | A workflow execution completes successfully |
| `dev.zigflow.task.started` | A task begins |
| `dev.zigflow.task.retried` | A task is retried after failure |
| `dev.zigflow.task.cancelled` | A task is cancelled |
| `dev.zigflow.task.faulted` | A task fails |
| `dev.zigflow.task.completed` | A task completes successfully |
| `dev.zigflow.iteration.completed` | A task iteration completes |

### Important notes

- CloudEvents are emitted on a best-effort basis.
- A failed event delivery does not fail the workflow.
- Event emission must remain within Temporal's determinism constraints.
- For high-throughput workflows, set appropriate HTTP timeouts to avoid
  workflow delays.

---

## Logging

Control log verbosity with `--log-level`:

```sh
zigflow run -f workflow.yaml --log-level debug
```

Valid values: `trace`, `debug`, `info`, `warn`, `error`. The code default
is `info`. If the `LOG_LEVEL` environment variable is set, that value
takes precedence.

Logs are structured JSON, written to stderr.

---

## Related pages

- [Deploying overview](/docs/deployment/intro) — runtime ports and configuration
- [Docker](/docs/deployment/docker) — Docker and Compose configuration
- [Kubernetes](/docs/deployment/kubernetes) — Helm chart deployment
- [Debugging workflows](/docs/dsl/debugging) — CloudEvents in detail

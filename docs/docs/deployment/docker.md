---
sidebar_position: 2
---

# Docker

Zigflow publishes a Docker image to the GitHub Container Registry on every
release.

## What you will learn

- How to run Zigflow using the official Docker image
- How to connect a containerised worker to a Temporal server
- How to configure the worker using environment variables
- How to set up Zigflow in a Docker Compose stack

## Image

```text
ghcr.io/zigflow/zigflow
```

Tags:

- `latest`: most recent release
- `0.1.0` (or any version): specific release

The binary is set as the container entrypoint. Pass `run` and any flags as
the command.

---

## Running the binary

To run Zigflow without Docker, download the binary for your
platform from the
[releases page](https://github.com/zigflow/zigflow/releases)
and run it directly:

```sh
zigflow run -f workflow.yaml
```

Pass any flag or set the equivalent environment variable. See
[Deploying overview](/docs/deployment/intro#connecting-to-temporal) for the full
connection flag reference.

---

## Basic usage

Mount your workflow file into the container and pass its path with `-f`:

```sh
docker run --rm \
  -v /path/to/workflow.yaml:/app/workflow.yaml \
  ghcr.io/zigflow/zigflow \
  run -f /app/workflow.yaml \
  --temporal-address host.docker.internal:7233
```

Use `host.docker.internal` to reach a Temporal server running on your host
machine. On Linux, you may need to use `172.17.0.1` or `--network host`
instead.

---

## Docker Compose

A typical Compose setup runs Temporal and Zigflow together:

```yaml title="compose.yaml"
services:
  temporal:
    image: temporalio/temporal
    command: server start-dev --ip 0.0.0.0
    ports:
      - "8233:8233"
      - "7233:7233"
    healthcheck:
      test: ["CMD-SHELL", "temporal operator cluster health"]
      interval: 10s
      timeout: 10s
      retries: 3

  worker:
    image: ghcr.io/zigflow/zigflow
    command:
      - run
      - -f
      - /app/workflow.yaml
    environment:
      TEMPORAL_ADDRESS: temporal:7233
      TEMPORAL_NAMESPACE: default
      LOG_LEVEL: info
    volumes:
      - ./workflow.yaml:/app/workflow.yaml
    depends_on:
      temporal:
        condition: service_healthy
```

---

## Environment variables

The Zigflow Docker image reads the same environment variables as the CLI
flags. Key variables:

| Variable | CLI flag | Default |
| --- | --- | --- |
| `TEMPORAL_ADDRESS` | `--temporal-address` | `localhost:7233` |
| `TEMPORAL_NAMESPACE` | `--temporal-namespace` | `default` |
| `TEMPORAL_API_KEY` | `--temporal-api-key` | (none) |
| `TEMPORAL_TLS` | `--temporal-tls` | `false` |
| `LOG_LEVEL` | `--log-level` | `info` |
| `CLOUDEVENTS_CONFIG` | `--cloudevents-config` | (none) |
| `WORKFLOW_FILE` | `-f` | (none) |
| `DISABLE_TELEMETRY` | `--disable-telemetry` | (false) |

Workflow environment variables (accessed via `$env` in expressions) can be
passed with the `ZIGGY_` prefix by default:

```yaml
environment:
  ZIGGY_API_BASE_URL: https://api.example.com
```

Inside the workflow:

```yaml
endpoint: ${ $env.API_BASE_URL }
```

---

## Ports

The container exposes two ports:

| Port | Purpose |
| --- | --- |
| `3000` | Health check (`/health`) |
| `9090` | Prometheus metrics |

Map them in Compose if needed:

```yaml
ports:
  - "3000:3000"
  - "9090:9090"
```

---

## Connecting to Temporal Cloud

For Temporal Cloud, enable TLS and provide authentication:

### API key

```yaml
environment:
  TEMPORAL_ADDRESS: your-namespace.tmprl.cloud:7233
  TEMPORAL_NAMESPACE: your-namespace
  TEMPORAL_TLS: "true"
  TEMPORAL_API_KEY: your-api-key
```

### mTLS

```yaml
environment:
  TEMPORAL_ADDRESS: your-namespace.tmprl.cloud:7233
  TEMPORAL_NAMESPACE: your-namespace
  TEMPORAL_TLS: "true"
  TEMPORAL_TLS_CLIENT_CERT_PATH: /path/to/cert
  TEMPORAL_TLS_CLIENT_KEY_PATH: /path/to/key
```

---

## Common mistakes

**The worker cannot reach the Temporal server.**
Check the `TEMPORAL_ADDRESS` value. In Docker Compose, use the service name
(`temporal:7233`). On a local host, use `host.docker.internal:7233`.

**The workflow file is not found.**
Verify the volume mount path matches the path passed to `-f`.

---

## Building a dedicated image

For production use, consider building an image with the workflow definition
baked in rather than mounting it at runtime. See
[Dedicated image](/docs/deployment/dedicated-image).

---

## Related pages

- [Deploying overview](/docs/deployment/intro): connection flags and telemetry
- [Dedicated image](/docs/deployment/dedicated-image): workflow in the image
- [Kubernetes](/docs/deployment/kubernetes): Helm chart deployment
- [Observability](/docs/deployment/observability): health and metrics

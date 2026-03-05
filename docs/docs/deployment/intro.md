---
sidebar_position: 1
---

# Deploying Zigflow

## What you will learn

- What Zigflow deploys and what infrastructure it requires
- Which deployment methods are available and when to use each
- How to connect a Zigflow worker to a Temporal server
- How to pass configuration via CLI flags or environment variables

## What Zigflow deploys

Zigflow is a single Go binary. When you run `zigflow run`, it starts a
**Temporal worker** that:

- Connects to a Temporal server
- Registers the compiled workflow
- Polls the task queue for executions

There is no separate API server, no database and no persistent storage.
All workflow state is held by Temporal.

Your infrastructure requirements are:

1. A running Temporal server (Cloud or self-hosted)
2. A Zigflow worker process for each workflow definition

---

## Deployment options

| Method | When to use |
| --- | --- |
| [Binary](/docs/deployment/docker#running-the-binary) | Direct binary, VMs, simple scripts |
| [Docker](/docs/deployment/docker) | Containerised environments, Docker Compose |
| [Kubernetes / Helm](/docs/deployment/kubernetes) | Production Kubernetes clusters |

For Docker and Kubernetes deployments, you can also
[build a dedicated image](/docs/deployment/dedicated-image) with the workflow
baked in at build time rather than mounted at runtime.

---

## Assumptions

**Provided by Zigflow:**

- A compiled binary for Linux, macOS and Windows via the
  [releases page](https://github.com/mrsimonemms/zigflow/releases)
- A Docker image published on every release to
  `ghcr.io/mrsimonemms/zigflow`
- A Helm chart published as an OCI artifact to
  `oci://ghcr.io/mrsimonemms/charts/zigflow`

**You must provide:**

- A running Temporal server, either
  [Temporal Cloud](https://temporal.io/cloud) or self-hosted
- Your workflow definition files

**Not included:**

- Container registry, ingress or other cluster infrastructure
- A Temporal server. Zigflow is a worker, not a server.

---

## Runtime ports

A running Zigflow worker exposes two ports:

| Port | Default | Purpose |
| --- | --- | --- |
| Health | `3000` | HTTP `/health`: liveness and readiness |
| Metrics | `9090` | Prometheus metrics |

Override with `--health-listen-address` and `--metrics-listen-address`.

---

## Connecting to Temporal

All deployment methods share the same connection flags:

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--temporal-address` | `TEMPORAL_ADDRESS` | `localhost:7233` | Temporal server address |
| `--temporal-namespace` | `TEMPORAL_NAMESPACE` | `default` | Temporal namespace |
| `--temporal-tls` | `TEMPORAL_TLS` | `false` | Enable TLS. Required when connecting to Temporal Cloud |
| `--temporal-api-key` | `TEMPORAL_API_KEY` | (none) | API key for Temporal Cloud |
| `--tls-client-cert-path` | `TEMPORAL_TLS_CLIENT_CERT_PATH` | (none) | Path to mTLS client certificate |
| `--tls-client-key-path` | `TEMPORAL_TLS_CLIENT_KEY_PATH` | (none) | Path to mTLS client key |

For [Temporal Cloud](https://temporal.io/cloud), enable TLS and provide authentication:

### API key

:::info
API key requires TLS to be enabled. Set `--temporal-tls` in addition to providing
API key.
:::

```sh
zigflow run \
  -f workflow.yaml \
  --temporal-address your-namespace.tmprl.cloud:7233 \
  --temporal-namespace your-namespace \
  --temporal-tls \
  --temporal-api-key your-api-key
```

### mTLS

:::info
mTLS requires TLS to be enabled. Set `--temporal-tls` in addition to providing the
client certificate and key.
:::

```sh
zigflow run \
  -f workflow.yaml \
  --temporal-address your-namespace.tmprl.cloud:7233 \
  --temporal-namespace your-namespace \
  --temporal-tls \
  --tls-client-cert-path /path/to/cert \
  --tls-client-key-path /path/to/key
```

---

## Environment variables

Zigflow reads its own configuration from environment variables that mirror
the CLI flag names. The naming convention is: flag `--flag-name` becomes
env var `FLAG_NAME` (uppercase, hyphens replaced by underscores).

Workflow definitions can also read environment variables via `$env`. By
default, environment variables prefixed with `ZIGGY_` are passed to the
workflow. Change the prefix with `--env-prefix`.

Example: the environment variable `ZIGGY_API_BASE_URL` is accessible in
expressions as `${ $env.API_BASE_URL }`.

---

## Telemetry

Telemetry helps the maintainers understand whether Zigflow is being used in
real production environments. No personal data is collected.

When a worker starts, Zigflow sends:

- an anonymous installation ID (generated locally on first run, or derived from
  the container hostname)
- the Zigflow version
- basic runtime information (OS, architecture, container detection)

When workflows are executed, Zigflow sends a periodic heartbeat (once per minute)
containing:

- the total number of workflow runs since the worker started
- the worker uptime in seconds

Heartbeats are only sent when the run count changes. Idle workers do not
emit repeated telemetry.

Zigflow does **not** collect:

- workflow definitions
- workflow inputs or outputs
- execution IDs
- task names
- hostnames
- environment variable values
- organisation identifiers

Telemetry exists solely to understand real-world adoption and usage.

**Opting out** is straightforward:

```sh
# Environment variable
DISABLE_TELEMETRY=true

# CLI flag
--disable-telemetry
```

---

## Next steps

- [Docker](/docs/deployment/docker): running Zigflow in a container
- [Kubernetes](/docs/deployment/kubernetes): deploying with the official Helm chart
- [Dedicated image](/docs/deployment/dedicated-image): workflow at build time
- [Observability](/docs/deployment/observability): health checks, metrics and CloudEvents

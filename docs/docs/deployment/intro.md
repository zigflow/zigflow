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

Zigflow is a single Go binary. When you run `zigflow run`, it starts one or
more **Temporal workers** that:

- Connect to a Temporal server
- Register one or more workflow definitions
- Poll task queues for executions

A single Zigflow process can load multiple workflow definitions from separate
files or a directory. Workflows that share a task queue (defined by
`document.taskQueue`) run on the same worker. Each distinct task queue gets
its own worker.

There is no separate API server, no database and no persistent storage.
All workflow state is held by Temporal.

Your infrastructure requirements are:

1. A running Temporal server (Cloud or self-hosted)
2. One or more Zigflow worker processes

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
  [releases page](https://github.com/zigflow/zigflow/releases)
- A Docker image published on every release to
  `ghcr.io/zigflow/zigflow`
- A Helm chart published as an OCI artifact to
  `oci://ghcr.io/zigflow/charts/zigflow`

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

| Port | Default | Endpoints |
| --- | --- | --- |
| Health | `3000` | `/livez` (liveness), `/readyz` (readiness), `/health` (alias for `/readyz`) |
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
| `--temporal-server-name` | `TEMPORAL_SERVER_NAME` | (none) | Override the TLS server name for SNI and certificate validation |
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

### Private connectivity (AWS PrivateLink, VPC endpoints)

When connecting to Temporal Cloud through a VPC endpoint or AWS PrivateLink, the
dial address (for example `vpce-*.amazonaws.com:7233`) does not match the
hostname on the Temporal Cloud certificate (for example `*.tmprl.cloud`). TLS
validation will fail unless you tell Zigflow which hostname to validate against.

Use `--temporal-server-name` to set the expected certificate hostname independently
of the dial address:

```sh
zigflow run \
  -f workflow.yaml \
  --temporal-address vpce-0123456789abcdef0.vpce-svc-0123456789abcdef0.us-east-1.vpce.amazonaws.com:7233 \
  --temporal-namespace your-namespace \
  --temporal-tls \
  --temporal-server-name your-namespace.tmprl.cloud \
  --temporal-api-key your-api-key
```

`--temporal-server-name` only takes effect when `--temporal-tls` is also set.
When omitted, the server name is derived from the dial address as normal.

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
real production environments. No personal or identifiable user data is collected.

When a worker starts, Zigflow sends:

- an anonymous installation ID (generated locally on first run, or derived from
  the container hostname)
- the Zigflow version
- basic runtime information (OS, architecture, container detection)
- approximate server country (2-letter code, derived once at startup)

The country value is derived once at startup and is not tied to any identity. No
IP addresses are stored.

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
- IP addresses or precise location data

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

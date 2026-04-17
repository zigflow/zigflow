---
sidebar_position: 3
---

# Kubernetes

Zigflow provides an official Helm chart for deploying to Kubernetes.

## What you will learn

- How to install the Zigflow Helm chart from the OCI registry
- How to deliver a workflow definition via inline YAML or Kubernetes Secret
- How to connect the worker to Temporal and configure environment variables
- How to scale workers and configure horizontal pod autoscaling
- How to deploy using Temporal Worker Controller for versioned rollouts
- How to configure worker versioning and deployment options

## Helm chart

The chart is published to the GitHub Container Registry as an OCI artifact.

### Install

Replace `${ZIGFLOW_VERSION}` with a
[published version](https://github.com/zigflow/zigflow/pkgs/container/charts%2Fzigflow):

```sh
helm install zigflow oci://ghcr.io/zigflow/charts/zigflow@${ZIGFLOW_VERSION}
```

---

## Minimal configuration

The chart requires at minimum a Temporal server address and a workflow
definition. The simplest configuration uses an inline workflow:

```yaml title="values.yaml"
config:
  temporal-address: temporal:7233

workflow:
  useInline: true
  inline:
    document:
      dsl: 1.0.0
      taskQueue: zigflow
      workflowType: hello-world
      version: 1.0.0
    do:
      - greet:
          set:
            message: Hello from Ziggy
          output:
            as:
              data: ${ . }
```

Install with custom values:

```sh
helm install zigflow oci://ghcr.io/zigflow/charts/zigflow@${ZIGFLOW_VERSION} \
  -f values.yaml
```

---

## Workflow delivery options

The chart supports three ways to provide the workflow file:

### Option 1 - Inline YAML (default)

Set `workflow.useInline: true` and provide the workflow under `workflow.inline`.
The chart renders the workflow into a ConfigMap.

```yaml title="values.yaml"
workflow:
  useInline: true
  inline:
    document:
      dsl: 1.0.0
      taskQueue: zigflow
      workflowType: my-workflow
      version: 1.0.0
    do:
      - ...
```

See [Minimal configuration](#minimal-configuration) for a complete example.

### Option 2 - Kubernetes Secret

Set `workflow.useInline: false` and create a Secret that contains the workflow:

```sh
kubectl create secret generic workflow \
  --from-file=workflow.yaml=./workflow.yaml
```

The chart mounts the Secret at `workflow.file` (default `/data/workflow.yaml`).

```yaml title="values.yaml"
workflow:
  useInline: false
  secret: workflow
  file: /data/workflow.yaml
```

### Option 3 - Dedicated image

If you have built an image with the workflow baked in (see
[Dedicated image](/docs/deployment/dedicated-image)), set `workflow.enabled: false`
to disable workflow injection entirely:

```yaml title="values.yaml"
# Add your image name and tag
image:
  repository: your-registry/your-image
  tag: your-tag

# Optionally, add in private registry credentials
imagePullSecrets:
  - name: my-registry-secret

workflow:
  enabled: false
```

The worker reads the workflow from the path already present inside the
image. No Secret or ConfigMap is created.

---

## Connecting to Temporal

Pass Temporal connection settings through the `config` map, which accepts
any CLI flag name:

```yaml title="values.yaml"
config:
  temporal-address: temporal:7233
  temporal-namespace: default
  log-level: info
```

For Temporal Cloud, use environment variables via the `envvars` list to
keep credentials in a Secret:

### API key

```yaml title="values.yaml"
config:
  temporal-address: your-namespace.tmprl.cloud:7233
  temporal-tls: true

envvars:
  - name: TEMPORAL_NAMESPACE
    valueFrom:
      secretKeyRef:
        name: temporal-config
        key: namespace
  - name: TEMPORAL_API_KEY
    valueFrom:
      secretKeyRef:
        name: temporal-config
        key: api-key
```

### mTLS

```yaml title="values.yaml"
config:
  temporal-address: your-namespace.tmprl.cloud:7233
  temporal-tls: true

envvars:
  - name: TEMPORAL_NAMESPACE
    valueFrom:
      secretKeyRef:
        name: temporal-config
        key: namespace
  - name: TEMPORAL_TLS_CLIENT_CERT_PATH
    value: /certs/cert
  - name: TEMPORAL_TLS_CLIENT_KEY_PATH
    value: /certs/key

volumes:
  - name: temporal-mtls
    secret:
      secretName: temporal-mtls

volumeMounts:
  - name: temporal-mtls
    mountPath: /certs
    readOnly: true
```

---

## Environment variables for workflows

Workflow `$env` variables are passed with the `ZIGGY_` prefix by default.
Add them to the `envvars` list:

```yaml title="values.yaml"
envvars:
  - name: ZIGGY_API_BASE_URL
    value: https://api.example.com
```

Inside the workflow:

```yaml
endpoint: ${ $env.API_BASE_URL }
```

---

## Health and readiness probes

The chart configures liveness and readiness probes automatically:

- Liveness probe: `GET /livez` on port `3000`
- Readiness probe: `GET /readyz` on port `3000`

No additional configuration is required for standard deployments. The `/health`
endpoint remains available as a backwards-compatible alias for `/readyz`.

---

## Replicas and scaling

:::tip
When using Temporal Worker Controller, autoscaling uses a `WorkerResourceTemplate`
instead of a `HorizontalPodAutoscaler`. See [Temporal Worker Controller](#temporal-worker-controller).
:::

Workers are stateless. You can run multiple replicas of the same worker
against the same Temporal task queue. Temporal distributes executions across
available workers.

```yaml title="values.yaml"
replicaCount: 3
```

Horizontal Pod Autoscaling is available but disabled by default:

```yaml title="values.yaml"
autoscaling:
  enabled: true
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
```

---

## Image configuration

```yaml title="values.yaml"
image:
  repository: ghcr.io/zigflow/zigflow
  pullPolicy: IfNotPresent
  tag: "0.1.0"  # Defaults to chart version if not set
```

---

## Temporal Worker Controller

:::info
Temporal Worker Controller is an advanced deployment mode. Standard Deployment
is the default and is appropriate for most use cases.
:::

[Temporal Worker Controller (TWC)](https://github.com/temporalio/temporal-worker-controller)
is a Kubernetes operator that manages worker lifecycle, rollouts and versioning
natively within the Temporal control plane. When `controller.enabled: true`, the
Helm chart renders TWC custom resources instead of a standard Kubernetes `Deployment`.

Use TWC when you need:

- Controlled, traffic-ramped rollouts for new worker versions
- Temporal-native versioning of worker deployments
- Automatic sunset of outdated worker versions

For standard production deployments, the default `Deployment` mode is sufficient.

### Prerequisites

Zigflow does not install Temporal Worker Controller, its CRDs or cert-manager.
You must install and manage these yourself before enabling TWC in the chart:

- Temporal Worker Controller and its CRDs
- [cert-manager](https://cert-manager.io/) (required by TWC)

Zigflow only renders the CRDs that TWC consumes. It does not install or manage TWC.

### Enabling TWC

Set `controller.enabled: true`:

```yaml title="values.yaml"
controller:
  enabled: true
  connection:
    hostPort: temporal.temporal.svc.cluster.local:7233
  workerOptions:
    temporalNamespace: default
```

When enabled, the chart renders:

- `TemporalConnection`: connection configuration consumed by TWC
- `TemporalWorkerDeployment`: worker definition with rollout and sunset settings

No standard Kubernetes `Deployment` is created.

### Connection and authentication

The `TemporalConnection` resource configures how TWC connects to Temporal.
This is separate from the `config` map used for Zigflow's own connection settings.

**No authentication (cluster-local Temporal):**

```yaml title="values.yaml"
controller:
  connection:
    hostPort: temporal.temporal.svc.cluster.local:7233
```

**mTLS:**

```yaml title="values.yaml"
controller:
  connection:
    hostPort: temporal.temporal.svc.cluster.local:7233
    mutualTLSSecretRef:
      name: temporal-mtls
```

**API key:**

```yaml title="values.yaml"
controller:
  connection:
    hostPort: your-namespace.tmprl.cloud:7233
    apiKeySecretRef:
      name: temporal-api-key
      key: api-key
```

`mutualTLSSecretRef` and `apiKeySecretRef` cannot both be set. Helm rendering
fails with a validation error if both are provided.

### Rollout strategy

Configure how TWC ramps traffic to new worker versions:

```yaml title="values.yaml"
controller:
  rollout:
    strategy: Progressive
    steps:
      - rampPercentage: 10
        pauseDuration: 5m
      - rampPercentage: 50
        pauseDuration: 10m
```

TWC applies each step in sequence. The example above ramps to 10% of traffic,
pauses for 5 minutes, ramps to 50%, pauses for 10 minutes, then promotes fully.

### Sunset behaviour

Old worker versions are retired according to the sunset configuration:

```yaml title="values.yaml"
controller:
  sunset:
    scaledownDelay: 1h
    deleteDelay: 24h
```

`scaledownDelay` controls how long to wait before scaling down an old version
once a new version is fully ramped. `deleteDelay` controls how long to keep
the old resource before deleting it.

### Autoscaling

When `autoscaling.enabled: true` and `controller.enabled: true`, the chart
renders a `WorkerResourceTemplate` instead of a `HorizontalPodAutoscaler`. TWC
uses this template to scale each worker version independently.

```yaml title="values.yaml"
autoscaling:
  enabled: true
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80

controller:
  enabled: true
```

In standard mode (controller disabled), the same `autoscaling` values render a
`HorizontalPodAutoscaler` targeting the `Deployment` directly.

### Worker versioning

When `controller.enabled: true`, the chart sets `ENABLE_VERSIONING=true` on the
worker container automatically. This activates Temporal worker deployment versioning.

TWC injects the following environment variables at runtime:

| Environment variable | Purpose | CLI flag equivalent |
| --- | --- | --- |
| `TEMPORAL_WORKER_BUILD_ID` | Build ID for this worker version | `--temporal-worker-build-id` |
| `TEMPORAL_DEPLOYMENT_NAME` | Deployment name for this worker version | `--temporal-deployment-name` |

Environment variables take precedence over CLI flag values. These values are
required when versioning is enabled.

You can also control versioning independently of TWC using the following flags
when running outside Kubernetes:

| Flag | Default | Description |
| --- | --- | --- |
| `--enable-versioning` | `false` | Enable Temporal worker deployment versioning |
| `--default-versioning-type` | `autoupgrade` | Default versioning type: `unspecified`, `pinned` or `autoupgrade` |
| `--temporal-worker-build-id` | (none) | Build ID for this worker |
| `--temporal-deployment-name` | (none) | Deployment name for this worker |

See [zigflow run](/docs/cli/commands/zigflow_run) for the full CLI reference.

---

## Full values reference

See the [Helm chart README](https://github.com/zigflow/zigflow/tree/main/charts/zigflow)
for the complete values reference.

---

## Smoke tests

The Helm chart includes basic smoke tests using Helm test hooks. These
confirm that the Zigflow worker pod starts successfully and is reachable.

Run the tests with:

```sh
helm test zigflow
```

If the tests fail, check the pod logs for more details:

```sh
kubectl logs -l app.kubernetes.io/instance=zigflow
```

The test definitions can be found under `templates/tests` in the chart.

---

## Common mistakes

**The pod starts but the workflow does not execute.**
Check that the task queue name (`document.taskQueue`) matches what your
client uses. Also confirm the worker is connected to the correct Temporal
namespace.

**ImagePullBackOff for the chart image.**
Verify the version tag exists in
[ghcr.io/zigflow/charts/zigflow](https://github.com/zigflow/zigflow/pkgs/container/charts%2Fzigflow).

**Workflow not updating after a values change.**
The chart renders the inline workflow into a ConfigMap. After a `helm
upgrade`, the pod must be restarted to pick up the new workflow.

**TWC resources not applied because prerequisites are missing.**
If Temporal Worker Controller and its CRDs are not installed in the cluster,
applying the chart with `controller.enabled: true` will fail. Install TWC and
cert-manager before enabling TWC in the chart.

**Both `mutualTLSSecretRef` and `apiKeySecretRef` set.**
Helm rendering fails if both `controller.connection.mutualTLSSecretRef.name` and
`controller.connection.apiKeySecretRef.name` are non-empty. Use one or the other.

---

## Related pages

- [Deploying overview](/docs/deployment/intro): connection flags and telemetry
- [Docker](/docs/deployment/docker): Docker and Docker Compose
- [Dedicated image](/docs/deployment/dedicated-image): workflow in the image
- [Observability](/docs/deployment/observability): health, metrics and CloudEvents

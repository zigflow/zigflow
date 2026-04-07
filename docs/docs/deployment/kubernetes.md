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

## Helm chart

The chart is published to the GitHub Container Registry as an OCI artifact.

### Install

Replace `${ZIGFLOW_VERSION}` with a
[published version](https://github.com/zigflow/zigflow/pkgs/container/charts%2Fzigflow):

```sh
helm install my-workflow oci://ghcr.io/zigflow/charts/zigflow@${ZIGFLOW_VERSION}
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
helm install my-workflow oci://ghcr.io/zigflow/charts/zigflow@${ZIGFLOW_VERSION} \
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

## Full values reference

See the [Helm chart README](https://github.com/zigflow/zigflow/tree/main/charts/zigflow)
for the complete values reference.

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

---

## Related pages

- [Deploying overview](/docs/deployment/intro): connection flags and telemetry
- [Docker](/docs/deployment/docker): Docker and Docker Compose
- [Dedicated image](/docs/deployment/dedicated-image): workflow in the image
- [Observability](/docs/deployment/observability): health, metrics and CloudEvents

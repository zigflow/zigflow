---
sidebar_position: 4
---

# Dedicated image

Building a dedicated image is a build-time pattern that works with both
[Docker](/docs/deployment/docker) and [Kubernetes](/docs/deployment/kubernetes)
deployments. Rather than mounting or injecting the workflow file at runtime,
you embed it directly into the container image.

The resulting image is self-contained and requires no volume mounts, no
ConfigMaps and no inline workflow configuration at runtime.

## What you will learn

- Why this approach works with the official Zigflow base image
- How to write a Dockerfile that embeds your workflow
- When this build pattern is useful
- The security posture of the Zigflow container image
- How to configure Kubernetes security contexts for the worker
- How to handle writable paths when using a read-only root filesystem
- How to verify the image before deploying

---

## How it works

The official Zigflow image sets one environment variable by default:

```text
WORKFLOW_FILE=/app/workflow.yaml
```

When Zigflow starts, it loads the file at the path specified by `WORKFLOW_FILE`.
Because this is already set in the base image, a container that copies a workflow
file to `/app/workflow.yaml` will start the worker immediately. No `-f` flag is
required and no override is needed.

**Single-file mode (default):** copy one workflow file to `/app/workflow.yaml`.
Zigflow loads it via `WORKFLOW_FILE`.

**Multi-file mode:** to load multiple workflow files from a directory, set
`WORKFLOW_DIRECTORY` in your Dockerfile. If you do not want to load the default
single file, also set `WORKFLOW_FILE=`:

```dockerfile title="Dockerfile"
FROM ghcr.io/zigflow/zigflow
ENV WORKFLOW_FILE=
ENV WORKFLOW_DIRECTORY=/app/workflows
COPY ./workflows /app/workflows
```

Each distinct `document.taskQueue` in those files gets its own Temporal worker
and task queue.

You can also combine both: set `WORKFLOW_FILE` and `WORKFLOW_DIRECTORY` together.
All discovered files are merged and deduplicated before startup.

:::warning
When using directory mode, the directory should contain workflow definitions
only. Non-workflow YAML/JSON files will be treated as workflows and may cause
startup errors.
:::

If a configured file does not exist, Zigflow will fail at startup. There is no
implicit fallback or silent skip.

---

## Dockerfile

```dockerfile title="Dockerfile"
FROM ghcr.io/zigflow/zigflow
COPY ./workflow.yaml /app/workflow.yaml
```

Build it:

```sh
docker build -t your-registry/your-image:your-tag .
```

Nothing else is required in the Dockerfile. The base image already defines the
entrypoint and the `WORKFLOW_FILE` path.

---

## Running the image

The workflow file is already present inside the image. Deploy it as you would
any other Zigflow container. No `-f` flag and no volume mount are required.

- [Docker](/docs/deployment/docker): run with `docker run` or Docker Compose
- [Kubernetes](/docs/deployment/kubernetes#option-3---dedicated-image): use
  Option 3 in the Helm chart workflow delivery options

---

## Production use

This approach is well suited to production environments for the following
reasons:

**Immutable deployments.** The workflow definition is part of the image. Once
built, the image cannot be modified without producing a new build. This
eliminates drift between the running container and a separately managed file.

**Versioning tied to the image tag.** Rolling back a workflow version is the same
operation as rolling back the image tag. There is no separate artefact to track.

**No runtime file mounts.** Volume mounts and ConfigMap projections introduce
a dependency between the running container and external configuration. Baking
the workflow into the image removes that dependency.

**Simpler CI/CD.** The build pipeline produces a single artefact. Deployment
consists of updating the image reference. There is nothing to synchronise.

---

## Security posture

The official `ghcr.io/zigflow/zigflow` image is built on
[Chainguard's Wolfi](https://github.com/chainguard-dev/wolfi-os), a minimal
Linux distribution with a reduced package surface and a minimal base image
without a general-purpose shell.

The image runs as a dedicated non-root user (`zigflow`, UID 1000). It contains
only the Zigflow binary, Node.js and Python for script task support, and the CA
certificate bundle.

---

## Script and shell task execution

:::warning
Script and shell tasks execute code directly on the Zigflow worker process.
:::

Any workflow that uses `run` tasks with a script or shell interpreter runs
that code inside the worker container, with access to the same environment,
network and mounted secrets as the worker itself.

Apply least-privilege principles:

- Do not mount credentials or secrets that the workflow does not require.
- Restrict network egress at the pod or network-policy level if workflows
  should not make arbitrary outbound connections.
- Treat the worker with the same trust boundaries you would apply to any
  code execution environment.

---

## Writable paths

By default, Zigflow writes only to `/tmp` during script task execution.

When `readOnlyRootFilesystem: true` is set, `/tmp` must be provided as a
writable volume. Mount an `emptyDir` at `/tmp`:

```yaml
volumes:
  - name: tmp
    emptyDir:
      medium: Memory
      sizeLimit: 32Mi

# inside the container spec:
volumeMounts:
  - mountPath: /tmp
    name: tmp
```

The Helm chart configures this volume automatically. If you are writing your
own manifests, add the mount shown above.

---

## Kubernetes security context

The Helm chart ships with the following security context defaults. If you are
deploying with the chart, these are applied automatically. If you are writing
your own manifests, apply these settings directly.

```yaml
podSecurityContext:
  runAsNonRoot: true
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

A minimal Kubernetes `Deployment` applying these contexts alongside the
`/tmp` volume looks like this:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-workflow
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: zigflow
          image: your-registry/your-image:your-tag
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            capabilities:
              drop:
                - ALL
            seccompProfile:
              type: RuntimeDefault
          volumeMounts:
            - mountPath: /tmp
              name: tmp
      volumes:
        - name: tmp
          emptyDir:
            medium: Memory
            sizeLimit: 32Mi
```

Resource limits should be configured to constrain CPU and memory usage for script
and shell tasks.

---

## Verifying the image

:::info
Set the `VERSION` environment variable to the Zigflow version
you want to use. This requires v0.9.0 or later.
:::

### Vulnerability scanning

Scan the image for known vulnerabilities before deploying:

```sh
trivy image \
  --severity HIGH,CRITICAL \
  --ignore-unfixed \
  ghcr.io/zigflow/zigflow:$VERSION
```

Zigflow's CI pipeline scans each published image at build time. HIGH and CRITICAL
findings are treated as build failures. Images are only published when that check
passes.

The `--ignore-unfixed` flag filters out vulnerabilities that do not yet have a
published fix, reducing noise from upstream issues.

### Helm chart config scanning

To check your Helm chart configuration for security misconfigurations:

```sh
trivy config \
  --severity HIGH,CRITICAL \
  ./charts/zigflow
```

### Signature verification (optional)

Images published from the main branch are signed with
[Cosign](https://github.com/sigstore/cosign) using keyless signing via
GitHub Actions OIDC. A CycloneDX SBOM is generated with Syft and attached
to the image reference.

To verify the signature:

```sh
cosign verify ghcr.io/zigflow/zigflow:$VERSION \
  --certificate-identity-regexp="https://github.com/zigflow/zigflow" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

To inspect the attached SBOM:

```sh
cosign download sbom ghcr.io/zigflow/zigflow:$VERSION
```

Signature verification is optional but recommended when deploying in
environments with strict supply chain requirements.

---

## Related pages

- [Deploying overview](/docs/deployment/intro): connection flags and telemetry
- [Docker](/docs/deployment/docker): runtime file mounts and Docker Compose
- [Kubernetes](/docs/deployment/kubernetes): Helm chart deployment

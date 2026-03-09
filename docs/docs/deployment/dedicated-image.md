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

---

## How it works

The official Zigflow image sets the following environment variable:

```text
WORKFLOW_FILE=/app/workflow.yaml
```

When Zigflow starts, it reads the workflow file from the path specified by
`WORKFLOW_FILE`. Because this variable is already set in the base image, a
container that provides a file at `/app/workflow.yaml` will start the worker
immediately. No `-f` flag is required and no `WORKFLOW_FILE` override is needed.

Copying your workflow file into the image at that path is sufficient.

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

**Versioning tied to the image tag.** Rolling back a workflow version is the
same operation as rolling back an image tag. There is no separate artefact to
track.

**No runtime file mounts.** Volume mounts and ConfigMap projections introduce
a dependency between the running container and external configuration. Baking
the workflow into the image removes that dependency.

**Simpler CI/CD.** The build pipeline produces a single artefact. Deployment
consists of updating the image reference. There is nothing to synchronise.

---

## Related pages

- [Deploying overview](/docs/deployment/intro): connection flags and telemetry
- [Docker](/docs/deployment/docker): runtime file mounts and Docker Compose
- [Kubernetes](/docs/deployment/kubernetes): Helm chart deployment

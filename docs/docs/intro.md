---
sidebar_position: 1
---
# Introduction

Zigflow runs workflows defined in YAML on [Temporal](https://temporal.io).
You write a workflow definition file; Zigflow compiles it, validates it and
registers it as a Temporal worker.

**Zigflow is for teams that want to:**

- Define workflow logic in YAML rather than in SDK code
- Separate orchestration structure from application implementation
- Reduce boilerplate when building Temporal workers

**Zigflow is not for:**

- Teams who prefer writing workflow code directly in a Temporal SDK
- Users seeking an official Temporal product. Zigflow is an independent project.

---

## Quick Start

### Install Zigflow

1. Find the binary for your computer from the [releases](https://github.com/zigflow/zigflow/releases)
   page
2. Make it executable `chmod +x ./path/to/binary`

## Start a Temporal Server (optional)

Zigflow works with all Temporal server types. [Cloud](https://temporal.io/cloud)
is best for high-performance, production workflows and [Self-Hosted](https://docs.temporal.io/self-hosted-guide)
is great for smaller workflows and development/testing.

For ease, this example uses the development server bundled with the
[Temporal CLI](https://docs.temporal.io/cli)

```sh
temporal server start-dev
```

### Create a Workflow

This workflow sets a single value and returns it as output.

```yaml title="workflow.yaml"
document:
  dsl: 1.0.0
  namespace: zigflow
  name: simple-workflow
  version: 1.0.0
do:
  - set:
      output:
        as:
          data: ${ . }
      set:
        message: Hello from Ziggy
```

:::tip
The DSL schema follows the Serverless Workflow specification
:::

### Run

Start the worker with a reference to the workflow file:

```sh
zigflow run -f workflow.yaml
```

### Trigger the Workflow

Temporal supports [multiple languages through their SDKs](https://docs.temporal.io/encyclopedia/temporal-sdks).
If you want to trigger this through your application, refer to these docs to create
your script.

To run through the UI:
- Go to your [Temporal UI](http://localhost:8233)
- Select "Start Workflow"
- Enter these parameters:
  - **Workflow ID**: generate a random UUID
  - **Task Queue**: enter `zigflow`
  - **Workflow Type**: enter `simple-workflow`
- Click "Start Workflow" and then go to the running workflow

You should see the workflow result:

```json
{
  "data": {
    "message": "Hello from Ziggy"
  }
}
```

Your first workflow is running. See the [Quickstart](/docs/getting-started/quickstart)
for a full walkthrough with validation and troubleshooting.

If Zigflow is valuable to you, consider
[supporting its development](/docs/support).

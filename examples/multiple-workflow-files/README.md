# Multiple Workflow Files

Run multiple workflow definitions from separate YAML files in a single Zigflow worker

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
go run .
```

This example runs multiple workflow definitions from separate YAML files
within a single Zigflow process.

In this case:

* each YAML file defines a separate workflow
* all workflows are loaded at startup
* workflows are grouped by task queue and registered on shared workers
* each workflow can be executed independently

This is useful when:

* you have many small or low-volume workflows
* you want to reduce deployment and operational overhead
* you prefer to organise workflows across multiple files rather than one large
  definition

> This differs from the [Multiple Workflows](../multiple-workflows/) example,
> where multiple workflows are defined in a single YAML file. Here, workflows
> are split across multiple files but run together in one Zigflow instance.

This will trigger one of the workflows with some input data and print
everything to the console.

## Diagram

**Task Queue**: `zigflow`
**Workflow Type**: `workflow1`

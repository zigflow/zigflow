# Hello World

Hello world with Zigflow

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
go run .
```

This will trigger the workflow and print everything to the console.

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    subgraph wf_catalog["catalog"]
        direction TB
        wf_catalog__start([Start])
        wf_catalog__end([End])
        catalog_workflow["RUN (workflow)"]
        wf_catalog__start --> catalog_workflow
        catalog_workflow --> wf_catalog__end
    end
```
<!-- ZIGFLOW_GRAPH_END -->

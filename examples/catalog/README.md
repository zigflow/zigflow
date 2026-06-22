# Catalogue

Load an external reference from an external catalogue

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
    catalog__start([Start])
    catalog__end([End])
    catalog_exec_catalog["RUN (exec-catalog)"]
    catalog__start --> catalog_exec_catalog
    catalog_exec_catalog --> catalog__end
```
<!-- ZIGFLOW_GRAPH_END -->

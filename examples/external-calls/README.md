# External Calls Workflow

An example of how to make external calls with Zigflow

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

Generate the protobuf definition

```sh
task -d ../../ generate-grpc
```

Now run the application

```sh
docker compose up starter
```

This will trigger the workflow with some input data and print everything to the
console.

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    external_calls__start([Start])
    external_calls__end([End])
    external_calls_fork["FORK (fork)"]
    external_calls_fork__join((" "))
    subgraph fork_external_calls_mcp_http["mcp-http"]
        direction TB
        external_calls_mcp_http__start([ ])
        external_calls_mcp_http__end([ ])
        external_calls_mcp_http__start --> external_calls_mcp_http__end
    end
    external_calls_fork --> external_calls_mcp_http__start
    external_calls_mcp_http__end --> external_calls_fork__join
    external_calls__start --> external_calls_fork
    external_calls_fork__join --> external_calls__end
```
<!-- ZIGFLOW_GRAPH_END -->

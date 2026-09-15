# Hello World External Storage

Hello world with Zigflow, but with the data stored externally

This example demonstrates the Claim Check pattern. Both the worker and the
trigger set a payload size threshold of 1, so every payload is offloaded to a
local S3-compatible store and only a reference passes through Temporal. For
the configuration reference, see
[External storage](https://zigflow.dev/docs/deployment/external-storage).

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

In one terminal, run:

```sh
docker compose up workflow
```

In another terminal, run:

```sh
docker compose up trigger
```

This will trigger the workflow and print everything to the console. When you
look in the [Temporal UI](http://localhost:8080), the history holds references
rather than values. Because the threshold is set to 1 on both the worker and
the trigger, every payload goes to the local S3 store instead of being sent to
the Temporal server.

The S3 container has no volume, so its contents are discarded when you run
`docker compose down`. That is a convenience of this throwaway stack rather
than anything the external storage mechanism does: offloaded objects have no
managed lifecycle, so a real deployment needs its own
[lifecycle policy](https://zigflow.dev/docs/deployment/external-storage#operational-considerations).

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    hello_world__start([Start])
    hello_world__end([End])
    hello_world_set["SET (set)"]
    hello_world__start --> hello_world_set
    hello_world_set --> hello_world__end
```
<!-- ZIGFLOW_GRAPH_END -->

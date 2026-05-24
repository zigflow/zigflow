# Hello World Encrypted Remote

Hello world with Zigflow, but remotely encrypted

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

In one terminal, run:

```sh
docker compose up worker
```

In another terminal, run:

```sh
docker compose up trigger
```

This will trigger the workflow and print everything to the console. When you look
in the [Temporal UI](http://localhost:8080), all the data will be stored in a
local Redis store meaning your data is never sent to the Temporal server.

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

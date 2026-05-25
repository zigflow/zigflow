# Wait

Pause a workflow on a Temporal durable timer using the Zigflow `wait` extension:
an absolute `until` moment or an expression-aware duration resolved from input.

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
go run .
```

This triggers the workflow with a `sendAt` set 10 seconds in the future and a
2-second `cancelGraceSeconds`, then prints the resulting payload.

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    wait__start([Start])
    wait__end([End])
    wait_prewarmDelay["WAIT (prewarmDelay)"]
    wait__start --> wait_prewarmDelay
    wait_cancelGrace["cancelGrace"]
    wait_prewarmDelay --> wait_cancelGrace
    wait_waitForSendTime["waitForSendTime"]
    wait_cancelGrace --> wait_waitForSendTime
    wait_recordSent["SET (recordSent)"]
    wait_waitForSendTime --> wait_recordSent
    wait_recordSent --> wait__end
```
<!-- ZIGFLOW_GRAPH_END -->

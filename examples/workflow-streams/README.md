# Workflow Streams

Stream data back from a workflow

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
    workflow_streams__start([Start])
    workflow_streams__end([End])
    workflow_streams_preWait["WAIT (preWait)"]
    workflow_streams__start --> workflow_streams_preWait
    workflow_streams_emitEvent["emitEvent"]
    workflow_streams_preWait --> workflow_streams_emitEvent
    workflow_streams_wait["WAIT (wait)"]
    workflow_streams_emitEvent --> workflow_streams_wait
    workflow_streams_emitEvent_2["emitEvent"]
    workflow_streams_wait --> workflow_streams_emitEvent_2
    workflow_streams_postWait["WAIT (postWait)"]
    workflow_streams_emitEvent_2 --> workflow_streams_postWait
    workflow_streams_postWait --> workflow_streams__end
```
<!-- ZIGFLOW_GRAPH_END -->

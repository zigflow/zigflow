# For Loops

An example of how to use the for loop task

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
go run .
```

This will trigger the workflow with some input data and print everything to the
console.

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    for_loop__start([Start])
    for_loop__end([End])
    for_loop_forTaskMap[["FOR (forTaskMap)"]]
    subgraph body_forTaskMap["forTaskMap (loop body)"]
        direction TB
        for_loop_forTaskMap_body__start([ ])
        for_loop_forTaskMap_body_setData["SET (setData)"]
        for_loop_forTaskMap_body__start --> for_loop_forTaskMap_body_setData
        for_loop_forTaskMap_body_wait["WAIT (wait)"]
        for_loop_forTaskMap_body_setData --> for_loop_forTaskMap_body_wait
    end
    for_loop_forTaskMap --> for_loop_forTaskMap_body__start
    for_loop_forTaskMap_body_wait -->|"next iteration"| for_loop_forTaskMap
    for_loop__start --> for_loop_forTaskMap
    for_loop_forTaskArray[["FOR (forTaskArray)"]]
    subgraph body_forTaskArray["forTaskArray (loop body)"]
        direction TB
        for_loop_forTaskArray_body__start([ ])
        for_loop_forTaskArray_body_setData["SET (setData)"]
        for_loop_forTaskArray_body__start --> for_loop_forTaskArray_body_setData
        for_loop_forTaskArray_body_wait["WAIT (wait)"]
        for_loop_forTaskArray_body_setData --> for_loop_forTaskArray_body_wait
    end
    for_loop_forTaskArray --> for_loop_forTaskArray_body__start
    for_loop_forTaskArray_body_wait -->|"next iteration"| for_loop_forTaskArray
    for_loop_forTaskMap --> for_loop_forTaskArray
    for_loop_forTaskNumber[["FOR (forTaskNumber)"]]
    subgraph body_forTaskNumber["forTaskNumber (loop body)"]
        direction TB
        for_loop_forTaskNumber_body__start([ ])
        for_loop_forTaskNumber_body_setData["SET (setData)"]
        for_loop_forTaskNumber_body__start --> for_loop_forTaskNumber_body_setData
        for_loop_forTaskNumber_body_wait["WAIT (wait)"]
        for_loop_forTaskNumber_body_setData --> for_loop_forTaskNumber_body_wait
    end
    for_loop_forTaskNumber --> for_loop_forTaskNumber_body__start
    for_loop_forTaskNumber_body_wait -->|"next iteration"| for_loop_forTaskNumber
    for_loop_forTaskArray --> for_loop_forTaskNumber
    for_loop_forTaskStateCarryOver[["FOR (forTaskStateCarryOver)"]]
    subgraph body_forTaskStateCarryOver["forTaskStateCarryOver (loop body)"]
        direction TB
        for_loop_forTaskStateCarryOver_body__start([ ])
        for_loop_forTaskStateCarryOver_body_incrementPage["SET (incrementPage)"]
        for_loop_forTaskStateCarryOver_body__start --> for_loop_forTaskStateCarryOver_body_incrementPage
    end
    for_loop_forTaskStateCarryOver --> for_loop_forTaskStateCarryOver_body__start
    for_loop_forTaskStateCarryOver_body_incrementPage -->|"next iteration"| for_loop_forTaskStateCarryOver
    for_loop_forTaskNumber --> for_loop_forTaskStateCarryOver
    for_loop_forTaskStateCarryOver --> for_loop__end
```
<!-- ZIGFLOW_GRAPH_END -->

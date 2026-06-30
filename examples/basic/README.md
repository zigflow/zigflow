# Basic Workflow

An example of how to use the Open Workflow Specification (formerly
Serverless Workflow) to define Temporal Workflows

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
    basic__start([Start])
    basic__end([End])
    basic_baseData["SET (baseData)"]
    basic__start --> basic_baseData
    basic_wait["WAIT (wait)"]
    basic_baseData --> basic_wait
    basic_getUser["CALL_HTTP (getUser)"]
    basic_wait --> basic_getUser
    basic_raiseAlarm["FORK (raiseAlarm)"]
    basic_raiseAlarm__join((" "))
    subgraph fork_basic_callNurse["callNurse"]
        direction TB
        basic_callNurse__start([ ])
        basic_callNurse__end([ ])
        basic_callNurse__start --> basic_callNurse__end
    end
    basic_raiseAlarm --> basic_callNurse__start
    basic_callNurse__end --> basic_raiseAlarm__join
    subgraph fork_basic_multiStep["multiStep"]
        direction TB
        basic_multiStep__start([ ])
        basic_multiStep__end([ ])
        basic_multiStep_wait1["WAIT (wait1)"]
        basic_multiStep__start --> basic_multiStep_wait1
        basic_multiStep_wait2["WAIT (wait2)"]
        basic_multiStep_wait1 --> basic_multiStep_wait2
        basic_multiStep_wait2 --> basic_multiStep__end
    end
    basic_raiseAlarm --> basic_multiStep__start
    basic_multiStep__end --> basic_raiseAlarm__join
    subgraph fork_basic_callDoctor["callDoctor"]
        direction TB
        basic_callDoctor__start([ ])
        basic_callDoctor__end([ ])
        basic_callDoctor__start --> basic_callDoctor__end
    end
    basic_raiseAlarm --> basic_callDoctor__start
    basic_callDoctor__end --> basic_raiseAlarm__join
    basic_getUser --> basic_raiseAlarm
    basic_raiseAlarm__join --> basic__end
```
<!-- ZIGFLOW_GRAPH_END -->

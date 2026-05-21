# Switch

Perform a switch statement

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
    subgraph wf_switch["switch"]
        direction TB
        wf_switch__start([Start])
        wf_switch__end([End])
        switch_wait["WAIT (wait)"]
        wf_switch__start --> switch_wait
        switch_switcher{"SWITCH (switcher)"}
        switch_wait --> switch_switcher
        switch_flowSwitcher{"SWITCH (flowSwitcher)"}
        switch_flowSwitcher -->|"${ $input.flow == 'exit' }"| wf_switch__end
        switch_flowSwitcher -->|"${ $input.flow == 'end' }"| wf_switch__end
        switch_switcher --> switch_flowSwitcher
        switch_wait_2["WAIT (wait)"]
        switch_flowSwitcher --> switch_wait_2
        switch_wait_2 --> wf_switch__end
    end
    subgraph wf_processElectronicOrder["processElectronicOrder"]
        direction TB
        wf_processElectronicOrder__start([Start])
        wf_processElectronicOrder__end([End])
        processElectronicOrder_validatePayment["CALL_HTTP (validatePayment)"]
        wf_processElectronicOrder__start --> processElectronicOrder_validatePayment
        processElectronicOrder_fulfillOrder["CALL_HTTP (fulfillOrder)"]
        processElectronicOrder_validatePayment --> processElectronicOrder_fulfillOrder
        processElectronicOrder_fulfillOrder --> wf_processElectronicOrder__end
    end
    subgraph wf_processPhysicalOrder["processPhysicalOrder"]
        direction TB
        wf_processPhysicalOrder__start([Start])
        wf_processPhysicalOrder__end([End])
        processPhysicalOrder_checkInventory["CALL_HTTP (checkInventory)"]
        wf_processPhysicalOrder__start --> processPhysicalOrder_checkInventory
        processPhysicalOrder_packItems["CALL_HTTP (packItems)"]
        processPhysicalOrder_checkInventory --> processPhysicalOrder_packItems
        processPhysicalOrder_scheduleShipping["CALL_HTTP (scheduleShipping)"]
        processPhysicalOrder_packItems --> processPhysicalOrder_scheduleShipping
        processPhysicalOrder_scheduleShipping --> wf_processPhysicalOrder__end
    end
    subgraph wf_handleUnknownOrderType["handleUnknownOrderType"]
        direction TB
        wf_handleUnknownOrderType__start([Start])
        wf_handleUnknownOrderType__end([End])
        handleUnknownOrderType_logWarning["CALL_HTTP (logWarning)"]
        wf_handleUnknownOrderType__start --> handleUnknownOrderType_logWarning
        handleUnknownOrderType_notifyAdmin["CALL_HTTP (notifyAdmin)"]
        handleUnknownOrderType_logWarning --> handleUnknownOrderType_notifyAdmin
        handleUnknownOrderType_notifyAdmin --> wf_handleUnknownOrderType__end
    end
    switch_switcher -->|"${ $input.orderType == 'electronic' }"| wf_processElectronicOrder__start
    switch_switcher -->|"${ $input.orderType == 'physical' }"| wf_processPhysicalOrder__start
    switch_switcher -->|"default"| wf_handleUnknownOrderType__start
```
<!-- ZIGFLOW_GRAPH_END -->

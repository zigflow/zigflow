# Money Transfer Demo

Temporal's world-famous [Money Transfer Demo](https://github.com/temporal-sa/money-transfer-demo),
in Open Workflow Specification (formerly Serverless Workflow) form.

<!-- toc -->

* [Getting started](#getting-started)
* [Diagram](#diagram)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
docker compose up workflow
```

Now open the [Money Transfer UI](http://localhost:7070) and the [Temporal UI](http://localhost:8080)

## Diagram

<!-- ZIGFLOW_GRAPH_START -->
```mermaid
flowchart TD
    subgraph wf_AccountTransferWorkflow["AccountTransferWorkflow"]
        direction TB
        wf_AccountTransferWorkflow__start([Start])
        wf_AccountTransferWorkflow__end([End])
        AccountTransferWorkflow_queryState["LISTEN (queryState)"]
        wf_AccountTransferWorkflow__start --> AccountTransferWorkflow_queryState
        AccountTransferWorkflow_setup["SET (setup)"]
        AccountTransferWorkflow_queryState --> AccountTransferWorkflow_setup
        AccountTransferWorkflow_validate["CALL_HTTP (validate)"]
        AccountTransferWorkflow_setup --> AccountTransferWorkflow_validate
        AccountTransferWorkflow_updateState["SET (updateState)"]
        AccountTransferWorkflow_validate --> AccountTransferWorkflow_updateState
        AccountTransferWorkflow_sleep["WAIT (sleep)"]
        AccountTransferWorkflow_updateState --> AccountTransferWorkflow_sleep
        AccountTransferWorkflow_withdraw["CALL_HTTP (withdraw)"]
        AccountTransferWorkflow_sleep --> AccountTransferWorkflow_withdraw
        AccountTransferWorkflow_updateState_2["SET (updateState)"]
        AccountTransferWorkflow_withdraw --> AccountTransferWorkflow_updateState_2
        AccountTransferWorkflow_sleep_2["WAIT (sleep)"]
        AccountTransferWorkflow_updateState_2 --> AccountTransferWorkflow_sleep_2
        AccountTransferWorkflow_deposit["CALL_HTTP (deposit)"]
        AccountTransferWorkflow_sleep_2 --> AccountTransferWorkflow_deposit
        AccountTransferWorkflow_updateState_3["SET (updateState)"]
        AccountTransferWorkflow_deposit --> AccountTransferWorkflow_updateState_3
        AccountTransferWorkflow_sleep_3["WAIT (sleep)"]
        AccountTransferWorkflow_updateState_3 --> AccountTransferWorkflow_sleep_3
        AccountTransferWorkflow_sendNotification["CALL_HTTP (sendNotification)"]
        AccountTransferWorkflow_sleep_3 --> AccountTransferWorkflow_sendNotification
        AccountTransferWorkflow_updateState_4["SET (updateState)"]
        AccountTransferWorkflow_sendNotification --> AccountTransferWorkflow_updateState_4
        AccountTransferWorkflow_sleep_4["WAIT (sleep)"]
        AccountTransferWorkflow_updateState_4 --> AccountTransferWorkflow_sleep_4
        AccountTransferWorkflow_sleep_4 --> wf_AccountTransferWorkflow__end
    end
    subgraph wf_AccountTransferWorkflowAdvancedVisibility["AccountTransferWorkflowAdvancedVisibility"]
        direction TB
        wf_AccountTransferWorkflowAdvancedVisibility__start([Start])
        wf_AccountTransferWorkflowAdvancedVisibility__end([End])
        AccountTransferWorkflowAdvancedVisibility_queryState["LISTEN (queryState)"]
        wf_AccountTransferWorkflowAdvancedVisibility__start --> AccountTransferWorkflowAdvancedVisibility_queryState
        AccountTransferWorkflowAdvancedVisibility_setup["SET (setup)"]
        AccountTransferWorkflowAdvancedVisibility_queryState --> AccountTransferWorkflowAdvancedVisibility_setup
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAdvancedVisibility_setup --> AccountTransferWorkflowAdvancedVisibility_advancedVisibility
        AccountTransferWorkflowAdvancedVisibility_validate["CALL_HTTP (validate)"]
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility --> AccountTransferWorkflowAdvancedVisibility_validate
        AccountTransferWorkflowAdvancedVisibility_updateState["SET (updateState)"]
        AccountTransferWorkflowAdvancedVisibility_validate --> AccountTransferWorkflowAdvancedVisibility_updateState
        AccountTransferWorkflowAdvancedVisibility_requireApprovalStatus["SET (requireApprovalStatus) [?]"]
        AccountTransferWorkflowAdvancedVisibility_updateState --> AccountTransferWorkflowAdvancedVisibility_requireApprovalStatus
        AccountTransferWorkflowAdvancedVisibility_requireApprovalListener["LISTEN (requireApprovalListener) [?]"]
        AccountTransferWorkflowAdvancedVisibility_requireApprovalStatus --> AccountTransferWorkflowAdvancedVisibility_requireApprovalListener
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility_2["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAdvancedVisibility_requireApprovalListener --> AccountTransferWorkflowAdvancedVisibility_advancedVisibility_2
        AccountTransferWorkflowAdvancedVisibility_withdraw["CALL_HTTP (withdraw)"]
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility_2 --> AccountTransferWorkflowAdvancedVisibility_withdraw
        AccountTransferWorkflowAdvancedVisibility_bug["RAISE (bug) [?]"]
        AccountTransferWorkflowAdvancedVisibility_withdraw --> AccountTransferWorkflowAdvancedVisibility_bug
        AccountTransferWorkflowAdvancedVisibility_updateState_2["SET (updateState)"]
        AccountTransferWorkflowAdvancedVisibility_bug --> AccountTransferWorkflowAdvancedVisibility_updateState_2
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility_3["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAdvancedVisibility_updateState_2 --> AccountTransferWorkflowAdvancedVisibility_advancedVisibility_3
        AccountTransferWorkflowAdvancedVisibility_deposit["CALL_HTTP (deposit)"]
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility_3 --> AccountTransferWorkflowAdvancedVisibility_deposit
        AccountTransferWorkflowAdvancedVisibility_updateState_3["SET (updateState)"]
        AccountTransferWorkflowAdvancedVisibility_deposit --> AccountTransferWorkflowAdvancedVisibility_updateState_3
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility_4["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAdvancedVisibility_updateState_3 --> AccountTransferWorkflowAdvancedVisibility_advancedVisibility_4
        AccountTransferWorkflowAdvancedVisibility_sendNotification["CALL_HTTP (sendNotification)"]
        AccountTransferWorkflowAdvancedVisibility_advancedVisibility_4 --> AccountTransferWorkflowAdvancedVisibility_sendNotification
        AccountTransferWorkflowAdvancedVisibility_updateState_4["SET (updateState)"]
        AccountTransferWorkflowAdvancedVisibility_sendNotification --> AccountTransferWorkflowAdvancedVisibility_updateState_4
        AccountTransferWorkflowAdvancedVisibility_sleep["WAIT (sleep)"]
        AccountTransferWorkflowAdvancedVisibility_updateState_4 --> AccountTransferWorkflowAdvancedVisibility_sleep
        AccountTransferWorkflowAdvancedVisibility_sleep --> wf_AccountTransferWorkflowAdvancedVisibility__end
    end
    subgraph wf_AccountTransferWorkflowHumanInLoop["AccountTransferWorkflowHumanInLoop"]
        direction TB
        wf_AccountTransferWorkflowHumanInLoop__start([Start])
        wf_AccountTransferWorkflowHumanInLoop__end([End])
        AccountTransferWorkflowHumanInLoop_queryState["LISTEN (queryState)"]
        wf_AccountTransferWorkflowHumanInLoop__start --> AccountTransferWorkflowHumanInLoop_queryState
        AccountTransferWorkflowHumanInLoop_setup["SET (setup)"]
        AccountTransferWorkflowHumanInLoop_queryState --> AccountTransferWorkflowHumanInLoop_setup
        AccountTransferWorkflowHumanInLoop_advancedVisibility["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowHumanInLoop_setup --> AccountTransferWorkflowHumanInLoop_advancedVisibility
        AccountTransferWorkflowHumanInLoop_validate["CALL_HTTP (validate)"]
        AccountTransferWorkflowHumanInLoop_advancedVisibility --> AccountTransferWorkflowHumanInLoop_validate
        AccountTransferWorkflowHumanInLoop_updateState["SET (updateState)"]
        AccountTransferWorkflowHumanInLoop_validate --> AccountTransferWorkflowHumanInLoop_updateState
        AccountTransferWorkflowHumanInLoop_requireApprovalStatus["SET (requireApprovalStatus) [?]"]
        AccountTransferWorkflowHumanInLoop_updateState --> AccountTransferWorkflowHumanInLoop_requireApprovalStatus
        AccountTransferWorkflowHumanInLoop_requireApprovalListener["LISTEN (requireApprovalListener) [?]"]
        AccountTransferWorkflowHumanInLoop_requireApprovalStatus --> AccountTransferWorkflowHumanInLoop_requireApprovalListener
        AccountTransferWorkflowHumanInLoop_advancedVisibility_2["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowHumanInLoop_requireApprovalListener --> AccountTransferWorkflowHumanInLoop_advancedVisibility_2
        AccountTransferWorkflowHumanInLoop_withdraw["CALL_HTTP (withdraw)"]
        AccountTransferWorkflowHumanInLoop_advancedVisibility_2 --> AccountTransferWorkflowHumanInLoop_withdraw
        AccountTransferWorkflowHumanInLoop_bug["RAISE (bug) [?]"]
        AccountTransferWorkflowHumanInLoop_withdraw --> AccountTransferWorkflowHumanInLoop_bug
        AccountTransferWorkflowHumanInLoop_updateState_2["SET (updateState)"]
        AccountTransferWorkflowHumanInLoop_bug --> AccountTransferWorkflowHumanInLoop_updateState_2
        AccountTransferWorkflowHumanInLoop_advancedVisibility_3["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowHumanInLoop_updateState_2 --> AccountTransferWorkflowHumanInLoop_advancedVisibility_3
        AccountTransferWorkflowHumanInLoop_deposit["CALL_HTTP (deposit)"]
        AccountTransferWorkflowHumanInLoop_advancedVisibility_3 --> AccountTransferWorkflowHumanInLoop_deposit
        AccountTransferWorkflowHumanInLoop_updateState_3["SET (updateState)"]
        AccountTransferWorkflowHumanInLoop_deposit --> AccountTransferWorkflowHumanInLoop_updateState_3
        AccountTransferWorkflowHumanInLoop_advancedVisibility_4["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowHumanInLoop_updateState_3 --> AccountTransferWorkflowHumanInLoop_advancedVisibility_4
        AccountTransferWorkflowHumanInLoop_sendNotification["CALL_HTTP (sendNotification)"]
        AccountTransferWorkflowHumanInLoop_advancedVisibility_4 --> AccountTransferWorkflowHumanInLoop_sendNotification
        AccountTransferWorkflowHumanInLoop_updateState_4["SET (updateState)"]
        AccountTransferWorkflowHumanInLoop_sendNotification --> AccountTransferWorkflowHumanInLoop_updateState_4
        AccountTransferWorkflowHumanInLoop_sleep["WAIT (sleep)"]
        AccountTransferWorkflowHumanInLoop_updateState_4 --> AccountTransferWorkflowHumanInLoop_sleep
        AccountTransferWorkflowHumanInLoop_sleep --> wf_AccountTransferWorkflowHumanInLoop__end
    end
    subgraph wf_AccountTransferWorkflowAPIDowntime["AccountTransferWorkflowAPIDowntime"]
        direction TB
        wf_AccountTransferWorkflowAPIDowntime__start([Start])
        wf_AccountTransferWorkflowAPIDowntime__end([End])
        AccountTransferWorkflowAPIDowntime_queryState["LISTEN (queryState)"]
        wf_AccountTransferWorkflowAPIDowntime__start --> AccountTransferWorkflowAPIDowntime_queryState
        AccountTransferWorkflowAPIDowntime_setup["SET (setup)"]
        AccountTransferWorkflowAPIDowntime_queryState --> AccountTransferWorkflowAPIDowntime_setup
        AccountTransferWorkflowAPIDowntime_advancedVisibility["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAPIDowntime_setup --> AccountTransferWorkflowAPIDowntime_advancedVisibility
        AccountTransferWorkflowAPIDowntime_validate["CALL_HTTP (validate)"]
        AccountTransferWorkflowAPIDowntime_advancedVisibility --> AccountTransferWorkflowAPIDowntime_validate
        AccountTransferWorkflowAPIDowntime_updateState["SET (updateState)"]
        AccountTransferWorkflowAPIDowntime_validate --> AccountTransferWorkflowAPIDowntime_updateState
        AccountTransferWorkflowAPIDowntime_requireApprovalStatus["SET (requireApprovalStatus) [?]"]
        AccountTransferWorkflowAPIDowntime_updateState --> AccountTransferWorkflowAPIDowntime_requireApprovalStatus
        AccountTransferWorkflowAPIDowntime_requireApprovalListener["LISTEN (requireApprovalListener) [?]"]
        AccountTransferWorkflowAPIDowntime_requireApprovalStatus --> AccountTransferWorkflowAPIDowntime_requireApprovalListener
        AccountTransferWorkflowAPIDowntime_advancedVisibility_2["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAPIDowntime_requireApprovalListener --> AccountTransferWorkflowAPIDowntime_advancedVisibility_2
        AccountTransferWorkflowAPIDowntime_withdraw["CALL_HTTP (withdraw)"]
        AccountTransferWorkflowAPIDowntime_advancedVisibility_2 --> AccountTransferWorkflowAPIDowntime_withdraw
        AccountTransferWorkflowAPIDowntime_bug["RAISE (bug) [?]"]
        AccountTransferWorkflowAPIDowntime_withdraw --> AccountTransferWorkflowAPIDowntime_bug
        AccountTransferWorkflowAPIDowntime_updateState_2["SET (updateState)"]
        AccountTransferWorkflowAPIDowntime_bug --> AccountTransferWorkflowAPIDowntime_updateState_2
        AccountTransferWorkflowAPIDowntime_advancedVisibility_3["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAPIDowntime_updateState_2 --> AccountTransferWorkflowAPIDowntime_advancedVisibility_3
        AccountTransferWorkflowAPIDowntime_deposit["CALL_HTTP (deposit)"]
        AccountTransferWorkflowAPIDowntime_advancedVisibility_3 --> AccountTransferWorkflowAPIDowntime_deposit
        AccountTransferWorkflowAPIDowntime_updateState_3["SET (updateState)"]
        AccountTransferWorkflowAPIDowntime_deposit --> AccountTransferWorkflowAPIDowntime_updateState_3
        AccountTransferWorkflowAPIDowntime_advancedVisibility_4["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowAPIDowntime_updateState_3 --> AccountTransferWorkflowAPIDowntime_advancedVisibility_4
        AccountTransferWorkflowAPIDowntime_sendNotification["CALL_HTTP (sendNotification)"]
        AccountTransferWorkflowAPIDowntime_advancedVisibility_4 --> AccountTransferWorkflowAPIDowntime_sendNotification
        AccountTransferWorkflowAPIDowntime_updateState_4["SET (updateState)"]
        AccountTransferWorkflowAPIDowntime_sendNotification --> AccountTransferWorkflowAPIDowntime_updateState_4
        AccountTransferWorkflowAPIDowntime_sleep["WAIT (sleep)"]
        AccountTransferWorkflowAPIDowntime_updateState_4 --> AccountTransferWorkflowAPIDowntime_sleep
        AccountTransferWorkflowAPIDowntime_sleep --> wf_AccountTransferWorkflowAPIDowntime__end
    end
    subgraph wf_AccountTransferWorkflowRecoverableFailure["AccountTransferWorkflowRecoverableFailure"]
        direction TB
        wf_AccountTransferWorkflowRecoverableFailure__start([Start])
        wf_AccountTransferWorkflowRecoverableFailure__end([End])
        AccountTransferWorkflowRecoverableFailure_queryState["LISTEN (queryState)"]
        wf_AccountTransferWorkflowRecoverableFailure__start --> AccountTransferWorkflowRecoverableFailure_queryState
        AccountTransferWorkflowRecoverableFailure_setup["SET (setup)"]
        AccountTransferWorkflowRecoverableFailure_queryState --> AccountTransferWorkflowRecoverableFailure_setup
        AccountTransferWorkflowRecoverableFailure_advancedVisibility["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowRecoverableFailure_setup --> AccountTransferWorkflowRecoverableFailure_advancedVisibility
        AccountTransferWorkflowRecoverableFailure_validate["CALL_HTTP (validate)"]
        AccountTransferWorkflowRecoverableFailure_advancedVisibility --> AccountTransferWorkflowRecoverableFailure_validate
        AccountTransferWorkflowRecoverableFailure_updateState["SET (updateState)"]
        AccountTransferWorkflowRecoverableFailure_validate --> AccountTransferWorkflowRecoverableFailure_updateState
        AccountTransferWorkflowRecoverableFailure_requireApprovalStatus["SET (requireApprovalStatus) [?]"]
        AccountTransferWorkflowRecoverableFailure_updateState --> AccountTransferWorkflowRecoverableFailure_requireApprovalStatus
        AccountTransferWorkflowRecoverableFailure_requireApprovalListener["LISTEN (requireApprovalListener) [?]"]
        AccountTransferWorkflowRecoverableFailure_requireApprovalStatus --> AccountTransferWorkflowRecoverableFailure_requireApprovalListener
        AccountTransferWorkflowRecoverableFailure_advancedVisibility_2["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowRecoverableFailure_requireApprovalListener --> AccountTransferWorkflowRecoverableFailure_advancedVisibility_2
        AccountTransferWorkflowRecoverableFailure_withdraw["CALL_HTTP (withdraw)"]
        AccountTransferWorkflowRecoverableFailure_advancedVisibility_2 --> AccountTransferWorkflowRecoverableFailure_withdraw
        AccountTransferWorkflowRecoverableFailure_bug["RAISE (bug) [?]"]
        AccountTransferWorkflowRecoverableFailure_withdraw --> AccountTransferWorkflowRecoverableFailure_bug
        AccountTransferWorkflowRecoverableFailure_updateState_2["SET (updateState)"]
        AccountTransferWorkflowRecoverableFailure_bug --> AccountTransferWorkflowRecoverableFailure_updateState_2
        AccountTransferWorkflowRecoverableFailure_advancedVisibility_3["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowRecoverableFailure_updateState_2 --> AccountTransferWorkflowRecoverableFailure_advancedVisibility_3
        AccountTransferWorkflowRecoverableFailure_deposit["CALL_HTTP (deposit)"]
        AccountTransferWorkflowRecoverableFailure_advancedVisibility_3 --> AccountTransferWorkflowRecoverableFailure_deposit
        AccountTransferWorkflowRecoverableFailure_updateState_3["SET (updateState)"]
        AccountTransferWorkflowRecoverableFailure_deposit --> AccountTransferWorkflowRecoverableFailure_updateState_3
        AccountTransferWorkflowRecoverableFailure_advancedVisibility_4["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowRecoverableFailure_updateState_3 --> AccountTransferWorkflowRecoverableFailure_advancedVisibility_4
        AccountTransferWorkflowRecoverableFailure_sendNotification["CALL_HTTP (sendNotification)"]
        AccountTransferWorkflowRecoverableFailure_advancedVisibility_4 --> AccountTransferWorkflowRecoverableFailure_sendNotification
        AccountTransferWorkflowRecoverableFailure_updateState_4["SET (updateState)"]
        AccountTransferWorkflowRecoverableFailure_sendNotification --> AccountTransferWorkflowRecoverableFailure_updateState_4
        AccountTransferWorkflowRecoverableFailure_sleep["WAIT (sleep)"]
        AccountTransferWorkflowRecoverableFailure_updateState_4 --> AccountTransferWorkflowRecoverableFailure_sleep
        AccountTransferWorkflowRecoverableFailure_sleep --> wf_AccountTransferWorkflowRecoverableFailure__end
    end
    subgraph wf_AccountTransferWorkflowInvalidAccount["AccountTransferWorkflowInvalidAccount"]
        direction TB
        wf_AccountTransferWorkflowInvalidAccount__start([Start])
        wf_AccountTransferWorkflowInvalidAccount__end([End])
        AccountTransferWorkflowInvalidAccount_queryState["LISTEN (queryState)"]
        wf_AccountTransferWorkflowInvalidAccount__start --> AccountTransferWorkflowInvalidAccount_queryState
        AccountTransferWorkflowInvalidAccount_setup["SET (setup)"]
        AccountTransferWorkflowInvalidAccount_queryState --> AccountTransferWorkflowInvalidAccount_setup
        AccountTransferWorkflowInvalidAccount_advancedVisibility["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowInvalidAccount_setup --> AccountTransferWorkflowInvalidAccount_advancedVisibility
        AccountTransferWorkflowInvalidAccount_validate["CALL_HTTP (validate)"]
        AccountTransferWorkflowInvalidAccount_advancedVisibility --> AccountTransferWorkflowInvalidAccount_validate
        AccountTransferWorkflowInvalidAccount_updateState["SET (updateState)"]
        AccountTransferWorkflowInvalidAccount_validate --> AccountTransferWorkflowInvalidAccount_updateState
        AccountTransferWorkflowInvalidAccount_requireApprovalStatus["SET (requireApprovalStatus) [?]"]
        AccountTransferWorkflowInvalidAccount_updateState --> AccountTransferWorkflowInvalidAccount_requireApprovalStatus
        AccountTransferWorkflowInvalidAccount_requireApprovalListener["LISTEN (requireApprovalListener) [?]"]
        AccountTransferWorkflowInvalidAccount_requireApprovalStatus --> AccountTransferWorkflowInvalidAccount_requireApprovalListener
        AccountTransferWorkflowInvalidAccount_advancedVisibility_2["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowInvalidAccount_requireApprovalListener --> AccountTransferWorkflowInvalidAccount_advancedVisibility_2
        AccountTransferWorkflowInvalidAccount_withdraw["CALL_HTTP (withdraw)"]
        AccountTransferWorkflowInvalidAccount_advancedVisibility_2 --> AccountTransferWorkflowInvalidAccount_withdraw
        AccountTransferWorkflowInvalidAccount_bug["RAISE (bug) [?]"]
        AccountTransferWorkflowInvalidAccount_withdraw --> AccountTransferWorkflowInvalidAccount_bug
        AccountTransferWorkflowInvalidAccount_updateState_2["SET (updateState)"]
        AccountTransferWorkflowInvalidAccount_bug --> AccountTransferWorkflowInvalidAccount_updateState_2
        AccountTransferWorkflowInvalidAccount_advancedVisibility_3["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowInvalidAccount_updateState_2 --> AccountTransferWorkflowInvalidAccount_advancedVisibility_3
        AccountTransferWorkflowInvalidAccount_deposit["CALL_HTTP (deposit)"]
        AccountTransferWorkflowInvalidAccount_advancedVisibility_3 --> AccountTransferWorkflowInvalidAccount_deposit
        AccountTransferWorkflowInvalidAccount_updateState_3["SET (updateState)"]
        AccountTransferWorkflowInvalidAccount_deposit --> AccountTransferWorkflowInvalidAccount_updateState_3
        AccountTransferWorkflowInvalidAccount_advancedVisibility_4["SET (advancedVisibility) [?]"]
        AccountTransferWorkflowInvalidAccount_updateState_3 --> AccountTransferWorkflowInvalidAccount_advancedVisibility_4
        AccountTransferWorkflowInvalidAccount_sendNotification["CALL_HTTP (sendNotification)"]
        AccountTransferWorkflowInvalidAccount_advancedVisibility_4 --> AccountTransferWorkflowInvalidAccount_sendNotification
        AccountTransferWorkflowInvalidAccount_updateState_4["SET (updateState)"]
        AccountTransferWorkflowInvalidAccount_sendNotification --> AccountTransferWorkflowInvalidAccount_updateState_4
        AccountTransferWorkflowInvalidAccount_sleep["WAIT (sleep)"]
        AccountTransferWorkflowInvalidAccount_updateState_4 --> AccountTransferWorkflowInvalidAccount_sleep
        AccountTransferWorkflowInvalidAccount_sleep --> wf_AccountTransferWorkflowInvalidAccount__end
    end
```
<!-- ZIGFLOW_GRAPH_END -->

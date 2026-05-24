# Examples

A collection of examples

<!-- toc -->

* [Applications](#applications)
* [Running](#running)
  * [Running the worker](#running-the-worker)
  * [Starting the workflow](#starting-the-workflow)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Applications

<!-- apps-start -->

| Name | Description |
| --- | --- |
| [Activity Call](./activity-call) | Invoke an external Temporal activity from Zigflow |
| [Agentic Workflow](./agentic-workflow) | A bounded plan/act/observe loop driven by an AI planner and a tool-backed lookup activity. |
| [Authorise Change Request](./authorise-change-request) | Authorise and implement or reject a change request |
| [Basic](./basic) | An example of how to use Serverless Workflow to define Temporal Workflows |
| [Catch Error](./catch-error) | Catch an error |
| [Child Workflows](./child-workflows) | Define multiple workflows and call a child workflow from a parent |
| [Debugging](./cloudevents) | An example of how to use CloudEvents for debugging workflows |
| [Competing Concurrent Tasks](./competing-concurrent-tasks) | Have two tasks competing and the first to finish wins |
| [Conditional Tasks](./conditionally-execute) | Execute tasks conditionally |
| [External Calls](./external-calls) | An example of how to use Zigflow to make external gRPC and HTTP calls |
| [For](./for-loop) | How to use the for loop task |
| [Heartbeat](./heartbeat) | Set [activity heartbeat](https://docs.temporal.io/encyclopedia/detecting-activity-failures#activity-heartbeat). Useful on long-running activities. |
| [Hello World](./hello-world) | Hello world with Zigflow |
| [Hello World AES Encrypted](./hello-world-encrypted-aes) | Hello world with Zigflow, but [encrypted](https://github.com/mrsimonemms/temporal-codec-server) |
| [Hello World Encrypted External Storage](./hello-world-encrypted-external-storage) | Hello world with Zigflow, but the data stored in an external service |
| [Hello World Encrypted Remote](./hello-world-encrypted-remote) | Hello world with Zigflow, but remotely encrypted |
| [Money Transfer Demo](./money-transfer) | Temporal's world-famous Money Transfer Demo, in Zigflow form |
| [Multiple Workflow Files](./multiple-workflow-files) | Run multiple workflow definitions from separate YAML files in a single Zigflow worker |
| [Multiple Workflows](./multiple-workflows) | Define and run multiple workflows within a single YAML file |
| [Priority and Fairness](./priority-and-fairness) | Using Temporal's Priority and Fairness features with Zigflow |
| [Python](./python) | The basic example, but in Python |
| [Query Listeners](./query) | Listen for Temporal query events |
| [Throw an error](./raise) | Throw an error from a Temporal workflow |
| [Run Task](./run-task) | How to execute code in a container, NodeJS, Python and Shell |
| [Run Container Task Kubernetes](./run-task-kubernetes) | Run container tasks in a Kubernetes environment |
| [Scheduling](./schedule) | Schedule the tasks to be triggered automatically |
| [Custom Search Attributes](./search-attributes) | How to add custom search attribute data into your Temporal workflows |
| [Signal Listeners](./signal) | Listen for Temporal signal events |
| [Switching](./switch) | Perform a switch statement |
| [Try/Catch](./try-catch) | An example of how to catch an erroring workflow |
| [TypeScript](./typescript) | The basic example, but in TypeScript |
| [Update Listeners](./update) | Listen for Temporal update events |
| [Wait](./wait) | Pause a workflow on a Temporal durable timer with until and expression durations. |

<!-- apps-end -->

## Running

> These commands should be run from the root directory

The `NAME` variable should be set to the example you wish to run (eg, `basic`)

### Running the worker

```sh
task worker NAME=<example>
```

### Starting the workflow

```sh
task start NAME=<example>
```

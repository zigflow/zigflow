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
| [Authorise Change Request](./authorise-change-request) | Authorise and implement or reject a change request |
| [Basic](./basic) | An example of how to use Serverless Workflow to define Temporal Workflows |
| [Child Workflows](./child-workflows) | Define multiple workflows and call a child workflow from a parent |
| [Debugging](./cloudevents) | An example of how to use CloudEvents for debugging workflows |
| [Competing Concurrent Tasks](./competing-concurrent-tasks) | Have two tasks competing and the first to finish wins |
| [Conditional Tasks](./conditionally-execute) | Execute tasks conditionally |
| [External Calls](./external-calls) | An example of how to use Zigflow to make external gRPC and HTTP calls |
| [For](./for-loop) | How to use the for loop task |
| [Heartbeat](./heartbeat) | Set [activity heartbeat](https://docs.temporal.io/encyclopedia/detecting-activity-failures#activity-heartbeat). Useful on long-running activities. |
| [Hello World](./hello-world) | Hello world with Zigflow |
| [Hello World AES Encrypted](./hello-world-encrypted-aes) | Hello world with Zigflow, but [encrypted](https://github.com/mrsimonemms/temporal-codec-server) |
| [Hello World Encrypted Remote](./hello-world-encrypted-remote) | Hello world with Zigflow, but remotely encrypted |
| [Money Transfer Demo](./money-transfer) | Temporal's world-famous Money Transfer Demo, in Zigflow form |
| [Multiple Workflows](./multiple-workflows) | Configure multiple Temporal workflows from a single Zigflow definition |
| [Python](./python) | The basic example, but in Python |
| [Query Listeners](./query) | Listen for Temporal query events |
| [Throw an error](./raise) | Throw an error from a Temporal workflow |
| [Run Task](./run-task) | How to execute code in NodeJS, Python and Shell |
| [Scheduling](./schedule) | Schedule the tasks to be triggered automatically |
| [Custom Search Attributes](./search-attributes) | How to add custom search attribute data into your Temporal workflows |
| [Signal Listeners](./signal) | Listen for Temporal signal events |
| [Switching](./switch) | Perform a switch statement |
| [Try/Catch](./try-catch) | An example of how to catch an erroring workflow |
| [TypeScript](./typescript) | The basic example, but in TypeScript |
| [Update Listeners](./update) | Listen for Temporal update events |

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

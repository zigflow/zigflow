---
sidebar_position: 1
sidebar_label: The DSL
---
# Understanding the DSL

## What you will learn

- How Zigflow's DSL relates to the CNCF Serverless Workflow specification
- How a workflow definition is structured
- Available task types and their properties
- How runtime expressions, output shaping and metadata work

Zigflow is built upon the [CNCF's Serverless Workflow](https://serverlessworkflow.io)
project. This provides a solid foundation of a comprehensive and vendor-neutral
framework. The [specification](https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md)
is well documented and acts as the inspiration behind this project.

Zigflow supports Serverless Workflow `v1.0.0` and above.

By design, some aspects of the specification will not be implemented, may diverge
from the Serverless Workflow specification, or will implement additional aspects.
These will be documented.

## Workflow

A [workflow](#workflow) serves as a blueprint outlining the series of [tasks](/docs/dsl/tasks/intro)
required to execute a specific business operation. It details the sequence in
which [tasks](/docs/dsl/tasks/intro) must be completed, guiding users through
the process from start to finish, and helps streamline operations, ensure
consistency, and
optimise efficiency within an organisation.

### Properties {/*#workflow-properties*/}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| document | [`document`](#document) | `yes` | Documents the defined workflow. |
| do | [`map[string, task]`](/docs/dsl/tasks/intro) | `yes` | The [task(s)](/docs/dsl/tasks/intro) that must be performed by the [workflow](#workflow). |
| input | [`input`](#input) | `no` | Configures the workflow's input. |
| timeout | [`timeout`](#timeout) | `no` | The configuration of the workflow's activity [Start-To-Close timeout](https://docs.temporal.io/encyclopedia/detecting-activity-failures#start-to-close-timeout). Defaults to 15 seconds |
| schedule | [`schedule`](#schedule) | `no` | Configures the workflow's schedule, if any. |

## Document

Documents the workflow definition.

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| dsl | `string` | `yes` | The version of the DSL used to define the workflow. |
| taskQueue | `string` | `yes` | The Temporal [Task Queue](https://docs.temporal.io/task-queue) this workflow runs on. Workers register on this queue; clients must target the same name. |
| workflowType | `string` | `yes` | The Temporal workflow type name. This will be ignored if multiple [`do`](/docs/dsl/tasks/do) are set and the workflow names will be taken from the step name. |
| version | `string` | `yes` | The workflow's [semantic version](https://semver.org/) |
| title | `string` | `no` | The workflow's title. |
| summary | `string` | `no` | The workflow's Markdown summary. |
| tags | `map[string, string]` | `no` | A key/value mapping of the workflow's tags, if any. |
| metadata | `map` | `no` | Additional information about the workflow. |

## Input

Documents the structure - and optionally configures the transformation of - workflow/task
input data.

It's crucial for authors to document the schema of input data whenever feasible.
This documentation empowers consuming applications to provide contextual auto-suggestions
when handling runtime expressions.

When set, runtimes must validate raw input data against the defined schema before
applying transformations, unless defined otherwise.

### Properties {/*#input-properties*/}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| schema | [`schema`](#schema) | `no` | The [`schema`](#schema) used to describe and validate raw input data.<br />*Even though the schema is not required, it is strongly encouraged to document it, whenever feasible. The input will be validated against this schema, returning an error if the given input does not match.* |

### Examples {/*#input-examples*/}

```yaml
schema:
  format: json
  document:
    type: object
    properties:
      order:
        type: object
        required:
          - pet
        properties:
          pet:
            type: object
            required:
              - id
            properties:
              id:
                type: string
```

## Output

Documents the structure - and optionally configures the transformations of -
workflow/task output data.

It's crucial for authors to document the schema of output data whenever feasible.
This documentation empowers consuming applications to provide contextual auto-suggestions
when handling runtime expressions.

When set, runtimes must validate output data against the defined schema after
applying transformations, unless defined otherwise.

### Properties {/*#output-properties*/}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| schema | [`schema`](#schema) | `no` | The [`schema`](#schema) used to describe and validate raw input data.<br />*Even though the schema is not required, it is strongly encouraged to document it, whenever feasible.* |
| as | `string`<br />`object` | `no` | A [runtime expression](/docs/dsl/tasks/intro#runtime-expressions), if any, used to filter and/or mutate the workflow/task output. |

### Examples {/*#output-examples*/}

```yaml
output:
  schema:
    format: json
    document:
      type: object
      properties:
        petId:
          type: string
      required:
        - petId
  as:
    petId: ${ .pet.id }
```

## Export

Certain task needs to set the workflow context to save the task output for later
usage. Users set the content of the context through a runtime expression. The
result of the expression is the new value of the context. The expression is
evaluated against the transformed task output.

Optionally, the context might have an associated schema which is validated against
the result of the expression.

### Properties {/*#export-properties*/}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| schema | [`schema`](#schema) | `no` | The [`schema`](#schema) used to describe and validate context.<br />*Included to handle the non frequent case in which the context has a known format.* |
| as | `string`<br />`object` | `no` | A runtime expression, if any, used to export the output data to the context. |

### Examples {/*#export-examples*/}

Merge the task output into the current context.

```yaml
export:
  as: ${ $context + . }
```

Merge the task output into the context under the `task` key.

```yaml
export:
  as: '${ $context + { task: . } }'
```

Replace the context with the task output.

```yaml
export:
  as: ${ . }
```

## Schema

Describes a data schema.

### Properties {/*#schema-properties*/}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| format | `string` | `yes` | The schema format.<br />*Supported values are:*<br />*- `json`, which indicates the [JSONSchema](https://json-schema.org/) format.* |
| document | `object` | `yes` | The inline schema document. |

### Examples {/*#schema-examples*/}

```yaml
format: json
document:
  type: object
  properties:
    id:
      type: string
    firstName:
      type: string
    lastName:
      type: string
  required:
    - id
    - firstName
    - lastName
```

This expects the output to look something like this:

```json
{
  "id": "99a4ab14-29aa-4e1b-8ca8-e6610524b546",
  "firstName": "Ziggy",
  "lastName": "Stardust"
}
```

## Timeout

:::warning
The top-level `timeout` field is deprecated. Use `metadata.activityOptions.startToCloseTimeout`
instead. See [`metadata.activityOptions`](./metadata/activity-options#types-activity-options)
for details.
:::

Defines a workflow or task timeout.

### Properties {/*#timeout-properties*/}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| after | [`duration`](#duration) | `yes` | The duration after which the workflow or task times out. |

### Examples {/*#timeout-examples*/}

```yaml
document:
  dsl: 1.0.0
  taskQueue: default
  workflowType: timeout-example
  version: 0.1.0
do:
  - waitAMinute:
      wait:
        seconds: 60
timeout:
  after:
    seconds: 30
```

## Duration

Defines a time duration.

### Properties {/*#duration-properties*/}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| Days | `integer` | `no` | Number of days, if any. |
| Hours | `integer` | `no` | Number of hours, if any. |
| Minutes | `integer` | `no` | Number of minutes, if any. |
| Seconds | `integer` | `no` | Number of seconds, if any. |
| Milliseconds | `integer` | `no` | Number of milliseconds, if any. |

### Examples {/*#duration-examples*/}

*Example of a duration of 2 hours, 15 minutes and 30 seconds:*

```yaml
hours: 2
minutes: 15
seconds: 30
```

## Schedule

Configures the schedule of a workflow.

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| every | [`duration`](#duration) | `no` | Specifies the duration of the interval at which the workflow should be executed. Unlike `after`, this option will run the workflow regardless of whether the previous run is still in progress.<br />*Required when no other property has been set.* |
| cron | `string` | `no` | Specifies the schedule using a CRON expression, e.g., `0 0 * * *` for daily at midnight.<br />*Required when no other property has been set.* |

### Metadata

Additional options can be configured by setting `metadata` in the [Document](#document).
This configures a [Temporal schedule](https://docs.temporal.io/schedule).

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| scheduleWorkflowName | `string` | `yes` | Set the workflow name to trigger - this will either be the document.workflowType or the Do task |
| scheduleId | `string` | `no` | Set the schedule ID. If not set, this defaults to `zigflow_<document.workflowType>` |
| scheduleInput | `any[]` | `no` | Set the input |

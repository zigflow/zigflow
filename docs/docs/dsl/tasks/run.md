# Run

Executes external commands, scripts or containers as workflow activities.

## When to use this

Use Run when your workflow must execute code that cannot be
expressed in the YAML DSL: a Docker container, a shell command,
a JavaScript or Python script or another Zigflow workflow.

## Properties

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| run.container | [`container`](#container) | `no` | The definition of the container to run.<br />*Required if `script`, `shell` and `workflow` have not been set.* |
| run.script | [`script`](#script) | `no` | The definition of the script to run.<br />*Required if `container`, `shell` and `workflow` have not been set.* |
| run.shell | [`shell`](#shell) | `no` | The definition of the shell command to run.<br />*Required if `container`, `script` and `workflow` have not been set.* |
| run.workflow | [`workflow`](#workflow) | `no` | The definition of the workflow to run.<br />*Required if `container`, `script` and `shell` have not been set.* |
| await | `boolean` | `no` | Determines whether or not the process to run should be awaited for.<br />*When set to `false`, the task cannot wait for the process to complete and thus cannot output the process's result.* Only available for workflows.<br />*Defaults to `true`.* |

## Container

:::info
Currently, this only supports Docker containers run via the `docker` binary on
your local machine. Additional container runtimes are planned - please upvote
[#181](https://github.com/zigflow/zigflow/issues/181) to influence prioritsation.
:::

Enables the execution of external processes encapsulated within a containerised
environment, allowing workflows to interact with and execute complex operations
using containerised applications, scripts, or commands.

### Properties {#container-properties}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| image | `string` | `yes` | The name of the container image to run |
| name | `string` | `no` | A [runtime expression](/docs/dsl/tasks/intro#runtime-expressions), if any, used to give specific name to the container. Uses a UUID if not set. |
| command | `string` | `no` | The command, if any, to execute on the container |
| volumes | `map` | `no` | The container's volume mappings, if any |
| environment | `map` | `no` | A key/value mapping of the environment variables, if any, to use when running the configured process |
| arguments | `string[]` | `no` | A list of the arguments, if any, passed as argv to the command or default container CMD |
| lifetime | [`containerLifetime`](#container-lifetime) | `no` | An object used to configure the container's lifetime |

### Example {#container-example}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - container:
      run:
        container:
          image: alpine
          arguments:
            - env
          environment:
            hello: world
```

### Container Lifetime

Configures the lifetime of a container.

#### Properties {#container-lifetime-properties}

| Property | Type | Required | Description |
| --- | :---: | :---: | --- |
| cleanup | `string` | `yes` | The cleanup policy to use.<br />*Supported values are:<br />- `always`: the container is deleted immediately after execution.<br />- `never`: the runtime should never delete the container.*<br />*Defaults to `always`.* |

## Script

Enables the execution of custom scripts or code within a workflow, empowering
workflows to perform specialised logic, data processing, or integration tasks by
executing user-defined scripts written in various programming languages.

### Properties {#script-properties}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| language | `string` | `yes` | The language of the script to run.<br />*Supported values are: [`js` and `python`](#supported-languages).* |
| code | `string` | `yes` | The script's code. |
| arguments | `string[]` | `no` | A list of the arguments, if any, to the script as argv |
| environment | `map` | `no` | A key/value mapping of the environment variables, if any, to use when running the configured script process |

#### Supported languages

:::warning
The [Docker image](https://github.com/zigflow/zigflow/blob/main/Dockerfile)
installs latest versions of `nodejs` and `python3`. For specific versions of
these languages, build your own image.
:::

This is a list of available languages and the command that is called.

| Language | Binary Target |
| :--- | :--- |
| `js` | `node` |
| `python` | `python` |

### Example {#script-example}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - nodejs:
      export:
        as: '${ $context + { nodejs: . } }'
      run:
        script:
          language: js
          code: |
            const http = require('http');

            console.log(`${process.argv[2]} from ${process.env.NAME}`);

            console.log(http.STATUS_CODES);
          arguments:
            - Hello
          environment:
            NAME: js
  - python:
      output:
        as: '${ $context + { python: . } }'
      run:
        script:
          language: python
          code: |
            import os
            import sys

            def main():
                arg = sys.argv[1] if len(sys.argv) > 1 else ""

                name = os.getenv("NAME", "")

                print(f"{arg} from {name}")

            if __name__ == "__main__":
                main()
          arguments:
            - Hello
          environment:
            NAME: python
```

## Shell

Enables the execution of shell commands within a workflow, enabling workflows to
interact with the underlying operating system and perform system-level operations,
such as file manipulation, environment configuration, or system administration
tasks.

### Properties {#shell-properties}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| command | `string` | `yes` | The shell command to run |
| arguments | `string[]` | `no` | A list of the arguments, if any, to the shell command as argv |
| environment | `map` | `no` | A key/value mapping of the environment variables, if any, to use when running the configured process |

### Examples {#shell-examples}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - runShell:
      output:
        as: '${ $context + { shell: . } }'
      run:
        shell:
          command: ls
          arguments:
            - -la
            - /
```

## Workflow

Enables the invocation and execution of [child workflows](https://docs.temporal.io/child-workflows)
from a parent workflow, facilitating modularization, reusability, and abstraction
of complex logic or business processes by encapsulating them into standalone
workflow units.

### Properties {#workflow-properties}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| name | `string` | `yes` | The name of the workflow to run |
| namespace | `string` | `yes` | This is not used and only exists to maintain compatability with the Serverless Workflow schema |
| version | `string` | `yes` | This is not used and only exists to maintain compatability with the Serverless Workflow schema |
| input | `any` | `no` | The data, if any, to pass as input to the workflow to execute. The value should be validated against the target workflow's input schema, if specified |

### Example {#workflow-example}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
timeout:
  after:
    minutes: 1
do:
  - parentWorkflow:
      do:
        - wait:
            wait:
              seconds: 5
        - callChildWorkflow1:
            run:
              workflow:
                name: child-workflow1
                namespace: default
                version: 0.0.0
        - wait:
            wait:
              seconds: 5
        - callChildWorkflow2:
            run:
              workflow:
                name: child-workflow2
                namespace: default
                version: 0.0.0
  - child-workflow1:
      do:
        - wait:
            wait:
              seconds: 10
  - child-workflow2:
      do:
        - wait:
            wait:
              seconds: 3
```

## Gotchas

**Container execution requires the `docker` binary.** The Zigflow worker
process must have access to Docker on the host machine. The container runtime
is Docker only at this time; other runtimes are not yet supported.

**Script execution requires the language runtime in the worker image.** The
official Docker image includes Node.js and Python. For other languages or
specific versions, build a custom image.

**`namespace` and `version` in `run.workflow` are not used.** These fields
exist for Serverless Workflow specification compatibility only. The target
workflow is looked up by `name` on the same task queue.

## Related pages

- [Do](/docs/dsl/tasks/do): sequential subtask composition
- [Fork](/docs/dsl/tasks/fork): parallel child execution
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs): worker and
  activity model

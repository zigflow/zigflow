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
| code | `string` | `no` | The inline script code. Required if `source` is not set. |
| source | [`externalResource`](#external-source) | `no` | An external resource from which the script is fetched at execution time. Required if `code` is not set. |
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

### External source {#external-source}

When `source` is used instead of `code`, Zigflow fetches the script from an
external resource before execution. The resource is identified by its `endpoint`.

#### Properties {#external-source-properties}

| Name | Type | Required | Description |
| --- | :---: | :---: | --- |
| endpoint | `string` or runtime expression | `yes` | The URL of the external script. |

#### Supported schemes

| Scheme | Example |
| --- | --- |
| `https://` | `https://example.com/script.js` |
| `http://` | `http://internal-host/script.py` |
| `file://` | `file:///scripts/run.js` |

Any other scheme (for example `ftp://`) will cause the task to fail.

#### Runtime expressions

The `endpoint` value can be a [runtime expression](/docs/concepts/data-and-expressions),
evaluated using the workflow state at execution time. The expression is resolved
before the script is fetched and must produce a valid URL with a supported scheme.

```yaml
run:
  script:
    language: js
    source:
      endpoint: ${ $env.SCRIPT_URL }
```

Expressions have access to `$env`, `$input`, `$context` and `$data`.

:::warning
The `endpoint` field also accepts an object form with a `uri` property:

```yaml
source:
  endpoint:
    uri: https://example.com/script.js
```

Runtime expressions are **not** supported within the `uri` field of this object
form. Use a top-level expression (`endpoint: ${ ... }`) instead.
:::

#### Size limit

Scripts fetched over HTTP or HTTPS are limited to **10 MiB**. Requests that
exceed this limit will fail, whether the size is declared in a `Content-Length`
header or detected during streaming.

### Examples {#script-examples}

#### Inline script {#script-example-inline}

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

#### External file script {#script-example-file}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - runScript:
      run:
        script:
          language: js
          source:
            endpoint: file:///scripts/run.js
```

#### External HTTP script {#script-example-http}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - runScript:
      run:
        script:
          language: python
          source:
            endpoint: https://example.com/scripts/run.py
```

#### Expression-based endpoint {#script-example-expression}

The endpoint URL can be resolved from workflow state at execution time.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
do:
  - runScript:
      run:
        script:
          language: js
          source:
            endpoint: ${ $env.SCRIPT_URL }
```

The worker must have `SCRIPT_URL` set in its environment. The expression is
evaluated before the script is fetched.

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
| type | `string` | `yes` | The workflow type to run |
| input | `any` | `no` | The data, if any, to pass as input to the workflow to execute. The value should be validated against the target workflow's input schema, if specified |

### Example {#workflow-example}

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: example
  version: 0.0.1
  metadata:
    activityOptions:
      startToCloseTimeout:
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
                type: child-workflow1
        - wait:
            wait:
              seconds: 5
        - callChildWorkflow2:
            run:
              workflow:
                type: child-workflow2
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

**External scripts are limited to 10 MiB.** Scripts fetched over HTTP or HTTPS
will fail if the response body exceeds this limit. File sources are not subject
to this limit.

**Only `http`, `https` and `file` schemes are supported for external scripts.**
Using any other scheme (for example `ftp://`) will cause the task to fail with
an unsupported scheme error.

**`namespace` and `version` in `run.workflow` are not used.** These fields
exist for Serverless Workflow specification compatibility only. The target
workflow is looked up by `name` on the same task queue.

## Related pages

- [Do](/docs/dsl/tasks/do): sequential subtask composition
- [Fork](/docs/dsl/tasks/fork): parallel child execution
- [Concepts: how Zigflow runs](/docs/concepts/how-zigflow-runs): worker and
  activity model

# Run Script Workflow

An example of how to use Serverless Workflow to run scripts and shell commands

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
    run_task__start([Start])
    run_task__end([End])
    run_task_container["RUN (container)"]
    run_task__start --> run_task_container
    run_task_nodejs_file["RUN (nodejs_file)"]
    run_task_container --> run_task_nodejs_file
    run_task_nodejs["RUN (nodejs)"]
    run_task_nodejs_file --> run_task_nodejs
    run_task_python_file["RUN (python_file)"]
    run_task_nodejs --> run_task_python_file
    run_task_python["RUN (python)"]
    run_task_python_file --> run_task_python
    run_task_shell_file["RUN (shell_file)"]
    run_task_python --> run_task_shell_file
    run_task_shell["RUN (shell)"]
    run_task_shell_file --> run_task_shell
    run_task_shell --> run_task__end
```
<!-- ZIGFLOW_GRAPH_END -->

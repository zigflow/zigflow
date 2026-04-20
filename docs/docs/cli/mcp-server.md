---
sidebar_position: 2
---

# MCP Server

## What you will learn

- What the Zigflow MCP server is and what it is for
- How MCP-compatible clients use it for AI-assisted workflow development
- What tools the server exposes and what each one does
- How to develop and refine workflows iteratively using the server

---

Zigflow provides a [Model Context Protocol](https://modelcontextprotocol.io) (MCP)
server for AI-assisted workflow development. It exposes Zigflow's core
functionality as structured tools that MCP-compatible clients can call, including
access to workflow examples, the DSL schema and workflow validation. No
filesystem access or running Temporal server is required.

---

## Running the server

```sh
zigflow mcp
```

The server communicates over stdin/stdout using the MCP protocol. It is designed
to be launched and managed by an MCP-compatible client, not run interactively in
a terminal.

---

## Available tools

### list_examples

Returns the list of bundled workflow examples with name, title, description and
tags. Use this before calling `get_example` to see what patterns are available.

### get_example

Returns the YAML content and metadata for a named example. The `name` must match
an identifier returned by `list_examples`.

### get_schema

Returns the Zigflow DSL JSON Schema for the current version. Accepts `"json"` or
`"yaml"` as the output format. Use this to understand valid workflow structure
before generating or validating a definition.

### validate_workflow

Validates a workflow YAML string and returns structured errors. The `yaml` field
must contain the full workflow definition as a string.

Errors include a `stage` field that identifies where in the validation pipeline
the failure occurred:

| Stage | Meaning |
| --- | --- |
| `input` | The YAML input is missing or empty |
| `parse` | The YAML could not be parsed |
| `schema` | The workflow fails JSON Schema validation |
| `load` | The workflow cannot be loaded into the model |
| `struct` | The workflow fails structural validation |

A successful response includes `"valid": true` and no errors.

---

## Design

**Stateless.** Each tool call is independent. The server holds no state between
calls.

**No filesystem access.** The server does not access the host filesystem. All
tool operations are based on input provided in the request or data embedded in
the binary. This makes behaviour predictable and consistent across environments.

**Structured errors.** Validation failures include a `stage` and `message` field,
identifying exactly where and why a workflow is invalid.

**Embedded examples.** Bundled examples are compiled into the binary. No external
files are needed to list or fetch them.

---

## Usage pattern

A typical AI-assisted authoring session:

1. Call `list_examples` to browse available patterns
2. Call `get_example` to inspect a relevant example
3. Call `get_schema` to understand the DSL structure
4. Generate or modify a workflow YAML
5. Call `validate_workflow` to check it
6. Correct errors based on the `stage` and `message` fields
7. Repeat from step 5 until valid

Steps 4 to 7 are typically cycled several times. Each validation response gives
targeted feedback on what to refine next in the workflow.

Starting from a known example produces more accurate results than generating from
scratch. Zigflow's DSL is a deliberate subset of the Serverless Workflow
specification. The schema and bundled examples define what is actually supported.

:::note
AI-generated workflows are a starting point. `validate_workflow` confirms
structural validity against the DSL schema; it does not verify that the workflow
logic is correct for your use case. Review generated workflows before using them
in production.
:::

---

## Common mistakes

**Running it directly in a terminal.**
The server communicates over stdin/stdout using the MCP protocol. It will not
print anything useful when run interactively. Connect to it through an
MCP-compatible client.

**Passing a file path to `validate_workflow`.**
The tool accepts a YAML string, not a file path. Read the file contents first and
pass the YAML as a string.

---

## Related pages

- [DSL reference](/docs/dsl/intro): full schema for workflow definitions
- [Schema](/docs/dsl/schema): the JSON Schema for workflow files
- [Examples](/docs/examples): bundled workflow patterns
- [Using the CLI](/docs/cli/using-the-cli): command-line validation and
  workflow execution

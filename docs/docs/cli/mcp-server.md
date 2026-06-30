---
sidebar_position: 2
---

# MCP Server

:::info[Public MCP server]
Connect your AI tool to the hosted Zigflow MCP server:

```text
https://mcp.zigflow.dev
```

Hosted, public and read-only. No install and no API key required.
:::

Zigflow runs a [Model Context Protocol](https://modelcontextprotocol.io) (MCP)
server that gives AI development tools structured, read-only access to Zigflow.
It exposes a small set of tools so an assistant can work from real examples and
the exact DSL schema for the current Zigflow version, then validate the YAML it
produces.

## Why use it

- Scaffold a workflow from a real, supported example instead of from scratch
- Validate workflow YAML against the current DSL schema before you commit
- Keep your assistant aligned with the schema for your Zigflow version
- Stay safe: the server is read-only, never connects to Temporal and never reads
  your files

## Connecting a client

The hosted server uses the MCP
[Streamable HTTP](https://modelcontextprotocol.io) transport. Any MCP client that
supports remote (HTTP) servers can connect by using the endpoint:

```text
https://mcp.zigflow.dev
```

Add it as a remote MCP server in your client. No credentials are required.

:::note
Each MCP client manages servers differently, and those formats change often.
Zigflow documents the endpoint and the tools it provides. For how to add a remote
MCP server in a specific client, see the official
[MCP documentation](https://modelcontextprotocol.io/docs/getting-started/intro).
:::

## Available tools

The server exposes five read-only tools.

### list_examples

Lists the bundled workflow examples with their name, title, description and tags.
Takes no parameters. Use this before calling `get_example` to discover the
patterns that are available.

### get_example

Returns a named example, including its YAML content and metadata (name, title,
description and tags). The `name` field must match an identifier returned by
`list_examples`. If the name is unknown, the error message lists the available
names.

### get_schema

Returns the Zigflow DSL JSON Schema for the current version. The `output` field
accepts `"json"` (the default) or `"yaml"`. Use this to understand valid workflow
structure before generating or validating a definition.

Set the optional `def` field to return a single schema definition from `$defs`,
for example `{ "def": "taskList" }`. The name must match a `$defs` key exactly.
Unknown definitions return a tool error.

### get_task_docs

Returns authoritative documentation for a single task type. The `task_type`
field must be one of the supported task types: `call`, `do`, `for`, `fork`,
`listen`, `raise`, `run`, `set`, `switch`, `try` or `wait`. An unknown task type
returns a tool error that lists the supported types.

The response aggregates several sources so a client does not need to scrape the
documentation site:

| Field | Description |
| --- | --- |
| `description` | The task summary from the JSON Schema definition |
| `subTypes` | The task variants where the schema defines them, for example `call` returns `activity`, `grpc` and `http` |
| `schema` | The task's JSON Schema definition, the authoritative source for its properties and required fields |
| `documentation` | The full Markdown reference page for the task |
| `relatedLinks` | Canonical documentation URLs for the task |
| `examples` | Bundled, validated example workflows that use the task |

Use this to learn how a specific task type works before authoring YAML. The
schema and reference page are served from the same sources as the
[DSL reference](/docs/dsl/tasks/intro), so they stay in step with the engine.

### validate_workflow

Validates a workflow YAML string and returns structured errors. The `yaml` field
must contain the full workflow definition as a string, not a file path.

Each error includes a `stage` field that identifies where in the validation
pipeline the failure occurred:

| Stage | Meaning |
| --- | --- |
| `input` | The YAML input is missing or empty |
| `parse` | The YAML could not be parsed |
| `schema` | The workflow fails JSON Schema validation |
| `load` | The workflow cannot be loaded into the model |
| `struct` | The workflow fails structural validation |

Errors also include a `message`. Errors from the `schema` and `struct` stages
additionally include a `path` that pinpoints the failing field. `struct` stage
errors also include `rule` and `param` fields describing the failing rule. A
successful response includes `"valid": true` and no errors.

Recognised validation errors carry two further fields:

| Field | Meaning |
| --- | --- |
| `code` | A stable identifier for the class of error, such as `ERR_INVALID_TASK_QUEUE` |
| `documentation` | The documentation URL derived from `code` |

The `code` is additive metadata. The `message` is never rewritten to embed it.
The `documentation` URL is derived from the `code`, so the two always agree. The
URL is built by lowercasing the code, dropping the `ERR_` prefix and replacing
underscores with hyphens, so `ERR_INVALID_TASK_QUEUE` becomes
`https://zigflow.dev/errors/invalid-task-queue`.

Errors without a recognised `code` omit both the `code` and `documentation`
fields. For example, an invalid `taskQueue` returns:

```json
{
  "stage": "schema",
  "path": "$.document.taskQueue",
  "code": "ERR_INVALID_TASK_QUEUE",
  "message": "pattern: \"Not A Valid Queue\" does not match regular expression \"^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$\"",
  "documentation": "https://zigflow.dev/errors/invalid-task-queue"
}
```

## Typical workflow

A typical AI-assisted authoring session:

1. Call `list_examples` to browse available patterns
2. Call `get_example` to inspect a relevant example
3. Call `get_schema` to understand the DSL structure
4. Call `get_task_docs` to learn a specific task type in depth
5. Generate or modify a workflow YAML
6. Call `validate_workflow` to check it
7. Correct errors based on the `stage` and `message` fields
8. Repeat from step 6 until valid

Starting from a known example produces more accurate results than generating from
scratch. Zigflow's DSL is a deliberate subset of the Open Workflow Specification
(formerly Serverless Workflow). The schema and bundled examples define
what is actually supported.

:::note
AI-generated workflows are a starting point. `validate_workflow` confirms
structural validity against the DSL schema. It does not verify that the workflow
logic is correct for your use case. Review generated workflows before using them
in production.
:::

## Security

The server is intentionally limited in what it can do, which keeps it safe to
expose:

- **No filesystem access.** All tool operations use input from the request or
  data embedded in the binary. The server does not read or write host files.
- **No Temporal access.** The server does not connect to Temporal and cannot
  start, query or affect workflow executions.
- **Read-only operations.** Every tool returns information or validates input. No
  tool mutates state.
- **Stateless requests.** Each request is independent. The HTTP transport holds
  no per-session state, and request bodies are size-limited to bound memory use.

The hosted server adds no authentication of its own. If you self-host and expose
it publicly, apply authentication and access control at the reverse proxy or
ingress layer.

## Self-hosting

You can run the same server yourself, either over HTTP or over stdio.

### HTTP

```sh
zigflow mcp --transport http
```

The server listens on `0.0.0.0:8080` by default. Use `--address` to change the
listen address:

```sh
zigflow mcp --transport http --address 0.0.0.0:9000
```

Because the HTTP transport is stateless, it runs well behind a reverse proxy,
ingress or service such as Cloudflare, and scales horizontally without session
affinity.

### stdio

```sh
zigflow mcp
```

The stdio transport communicates over stdin/stdout and is the usual choice for a
client that launches and manages the server locally as a subprocess. It is not
meant to be run interactively in a terminal.

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--transport` | `stdio` | Transport to use: `stdio` or `http`. |
| `--address` | `0.0.0.0:8080` | Address to listen on. HTTP transport only. |
| `--website-url` | `https://mcp.zigflow.dev` | Website URL advertised by the server. |

An unknown `--transport` value fails the command rather than falling back to a
default.

## Common mistakes

**Passing a file path to `validate_workflow`.**
The tool accepts a YAML string, not a file path. Read the file contents first and
pass the YAML as a string.

**Running the stdio transport directly in a terminal.**
With the default stdio transport, the server communicates over stdin/stdout using
the MCP protocol. It will not print anything useful when run interactively.
Connect to it through an MCP client, or use the HTTP transport for a
network-facing server.

## Related pages

- [DSL reference](/docs/dsl/intro): full schema for workflow definitions
- [Schema](/docs/dsl/schema): the JSON Schema for workflow files
- [Examples](/docs/examples): bundled workflow patterns
- [Using the CLI](/docs/cli/using-the-cli): command-line validation and
  workflow execution

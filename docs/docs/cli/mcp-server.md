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

The server exposes four read-only tools.

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

Errors also include a `message`. Errors from the `struct` stage additionally
include `path`, `rule` and `param` fields that pinpoint the failing field and
rule. A successful response includes `"valid": true` and no errors.

## Typical workflow

A typical AI-assisted authoring session:

1. Call `list_examples` to browse available patterns
2. Call `get_example` to inspect a relevant example
3. Call `get_schema` to understand the DSL structure
4. Generate or modify a workflow YAML
5. Call `validate_workflow` to check it
6. Correct errors based on the `stage` and `message` fields
7. Repeat from step 5 until valid

Starting from a known example produces more accurate results than generating from
scratch. Zigflow's DSL is a deliberate subset of the Serverless Workflow
specification. The schema and bundled examples define what is actually supported.

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

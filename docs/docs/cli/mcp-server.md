---
sidebar_position: 2
---

# MCP Server

## What you will learn

- What the Zigflow MCP server is and what it is for
- How MCP-compatible clients use it for AI-assisted workflow development
- The difference between the stdio and HTTP transports, and when to use each
- How to start the server in each mode
- What tools the server exposes and what each one does
- How to connect from MCP Inspector and configure an MCP client
- Common deployment patterns and the server's security boundaries

---

Zigflow provides a [Model Context Protocol](https://modelcontextprotocol.io) (MCP)
server for AI-assisted workflow development. It exposes Zigflow's core
functionality as structured tools that MCP-compatible clients can call, including
access to workflow examples, the DSL schema and workflow validation.

The server is deliberately lightweight and read-only. It does not connect to
Temporal and it does not access the host filesystem. This makes it safe to run
locally or to expose as a shared, public-facing service.

---

## Transports

The server supports two transports. Both expose the same four tools and the same
behaviour. They differ only in how a client connects.

| Transport | Use when |
| --- | --- |
| `stdio` | An MCP client launches and manages the server locally as a subprocess. This is the default. |
| `http` | You want a long-running server that multiple clients connect to over the network, for example a shared or public deployment. |

The `stdio` transport communicates over stdin/stdout and is the usual choice for
locally launched MCP clients such as editor integrations and desktop assistants.

The `http` transport uses the MCP
[Streamable HTTP](https://modelcontextprotocol.io) protocol via the official
`NewStreamableHTTPHandler`. It runs in stateless mode, so each request is
self-contained and the server holds no per-session state between requests. This
makes it suitable for shared deployments and for running behind a reverse proxy,
ingress or service such as Cloudflare.

---

## Running the server

### stdio (default)

```sh
zigflow mcp
```

The server communicates over stdin/stdout using the MCP protocol. It is designed
to be launched and managed by an MCP-compatible client, not run interactively in
a terminal.

### HTTP

```sh
zigflow mcp --transport http
```

The server listens on `0.0.0.0:8080` by default. Use `--address` to change the
listen address:

```sh
zigflow mcp --transport http --address 0.0.0.0:9000
```

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--transport` | `stdio` | Transport to use: `stdio` or `http`. |
| `--address` | `0.0.0.0:8080` | Address to listen on. HTTP transport only. |
| `--website-url` | `https://mcp.zigflow.dev` | Website URL advertised by the server. |

An unknown `--transport` value fails the command rather than falling back to a
default.

---

## Available tools

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

---

## Connecting from MCP Inspector

[MCP Inspector](https://github.com/modelcontextprotocol/inspector) is a useful
way to explore the tools interactively.

For the stdio transport, point the Inspector at the command to launch:

```sh
npx @modelcontextprotocol/inspector zigflow mcp
```

For the HTTP transport, start the server in one terminal:

```sh
zigflow mcp --transport http
```

Then connect the Inspector to its URL in another terminal:

```sh
npx @modelcontextprotocol/inspector http://localhost:8080
```

---

## Configuring an MCP client

For locally launched clients, configure the stdio transport by giving the client
the command to run:

```json
{
  "mcpServers": {
    "zigflow": {
      "command": "zigflow",
      "args": ["mcp"]
    }
  }
}
```

For a server running over HTTP, point the client at its URL. The exact field
names depend on the client, but the transport is Streamable HTTP:

```json
{
  "mcpServers": {
    "zigflow": {
      "type": "streamable-http",
      "url": "http://localhost:8080"
    }
  }
}
```

---

## Deployment patterns

**Local stdio.** An MCP client launches `zigflow mcp` as a subprocess on the same
machine. This is the simplest setup and needs no network configuration.

**Local HTTP.** Run `zigflow mcp --transport http` and connect local clients to
`http://localhost:8080`. Useful when several clients on one machine share a single
server, or for local testing of an HTTP deployment.

**Public HTTP behind a reverse proxy.** Run the HTTP transport behind a reverse
proxy, ingress or service such as Cloudflare, which terminates TLS and forwards
requests to the server. Because the server is stateless, it scales horizontally
without session affinity.

---

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

The server adds no authentication of its own. When exposing it publicly, apply
authentication and access control at the reverse proxy or ingress layer.

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

**Running the stdio transport directly in a terminal.**
With the default stdio transport, the server communicates over stdin/stdout using
the MCP protocol. It will not print anything useful when run interactively.
Connect to it through an MCP-compatible client, or use the HTTP transport for a
network-facing server.

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

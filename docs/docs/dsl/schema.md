---
sidebar_label: Schema
---

# Schema

## What you will learn

- What the Zigflow JSON Schema is and what it covers
- How to reference the hosted schema in your editor

Zigflow provides a JSON Schema for its workflow DSL. The schema describes the
fields, types and constraints that Zigflow enforces, and can be used by editors,
linters and code generators to validate workflow files before you run them.

:::warning
The Zigflow schema reflects only the features that Zigflow supports. It is a
deliberate subset of the CNCF Serverless Workflow specification. Workflow
constructs not supported by Zigflow are absent from the schema. If your workflow
uses an unsupported construct, validation against this schema will fail.
:::

## Schema URLs

The latest schema is available at:

- `https://zigflow.dev/schema.yaml`
- `https://zigflow.dev/schema.json`

For production use, you should pin to a specific version.. Versioned schemas do
not change after release:

- `https://zigflow.dev/schema/<version>/workflow.yaml`
- `https://zigflow.dev/schema/<version>/workflow.json`

Versioned schemas are available from version `0.10.0` onwards.

## Editor setup

Most editors that support JSON Schema can be configured to apply the Zigflow
schema to your workflow files. Once configured, your editor will provide
validation, autocomplete and inline error messages.

### VS Code example

```json title=".vscode/settings.json"
{
  "yaml.schemas": {
    "https://zigflow.dev/schema.yaml": [
      "workflows/**/*.yaml",
      "workflows/**/*.yml",
      "workflow.yaml",
      "workflow.yml"
    ]
  }
}
```

This requires the [Red Hat YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml).
For full configuration options, see the
[extension documentation](https://github.com/redhat-developer/vscode-yaml#language-server-settings).

Other editors with JSON Schema support can be configured similarly. See
[JetBrains YAML schema mappings](https://www.jetbrains.com/help/idea/json.html#ws_json_using_schemas)
for JetBrains IDEs.

## Related pages

- [The DSL](/docs/dsl/intro)
- [Validating workflows](/docs/cli/commands/zigflow_validate)

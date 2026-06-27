# Invalid Workflow (validation demo)

This example is intentionally invalid. It exists to demonstrate the stable
validation error codes and documentation links that Zigflow attaches to
validation failures. Do not use it as a template for a real workflow.

## Why it is invalid

A `document.taskQueue` must be an RFC 1123 DNS label: letters, digits and
hyphens only, starting and ending with a letter or digit. The value in
[workflow.yaml](./workflow.yaml) contains spaces:

```yaml
taskQueue: Not A Valid Queue
```

Every other field is valid, so the workflow fails on this single rule.

## Expected validation error

Validation rejects the document with the `ERR_INVALID_TASK_QUEUE` error code
and a documentation link derived from that code.

## How to run it

```bash
zigflow validate examples/invalid/workflow.yaml --output-json
```

The command exits with a non-zero status and prints the failure as JSON. The
exact wording of `message` may change, but `code` and `documentation` are
stable:

```json
{
  "valid": false,
  "file": "examples/invalid/workflow.yaml",
  "errors": [
    {
      "key": "schema_validation",
      "code": "ERR_INVALID_TASK_QUEUE",
      "message": "pattern: \"Not A Valid Queue\" does not match regular expression \"^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$\"",
      "path": "$.document.taskQueue",
      "documentation": "https://zigflow.dev/errors/invalid-task-queue"
    }
  ]
}
```

Following `documentation` resolves to the relevant section of the
[Common Mistakes](https://zigflow.dev/docs/concepts/common-mistakes) guide.

# Common Mistakes Quick Reference

When you see these patterns, fix them immediately.

## Wrong task types

| Written | Fix |
| --- | --- |
| `http:` | `call: http` with `with: { method, endpoint }` |
| `parallel:` | `fork: { compete: false, branches: [...] }` |
| `function:` | `call: activity` with `with: { name, arguments, taskQueue }` |
| `delay:` | `wait: { seconds: N }` |
| `emit:` | Not supported, remove |

## Wrong field names

| Written | Fix |
| --- | --- |
| `url:` | `endpoint:` |
| `document.name` | `document.title` |
| `document.description` | `document.summary` |
| `document.author` | Put in `document.metadata` |

## Wrong expression syntax

| Written | Fix |
| --- | --- |
| `{{ $input.x }}` | `${ $input.x }` |
| `${{ $input.x }}` | `${ $input.x }` |
| `$.input.x` | `${ $input.x }` |
| `$workflow.id` | `${ $data.workflow.workflow_execution_id }` |
| `$now` | `${ timestamp }` inside a `set` task |
| `$steps.foo` | `${ $data.foo }` |
| `$vars.x` | `${ $context.x }` |

## Wrong duration format

| Written | Fix |
| --- | --- |
| `wait: PT1M` | `wait: { minutes: 1 }` |
| `wait: { minute: 1 }` | `wait: { minutes: 1 }` |
| `wait: { second: 5 }` | `wait: { seconds: 5 }` |
| `wait: 60` | `wait: { seconds: 60 }` |

## Determinism violations

| Pattern | Fix |
| --- | --- |
| `uuid` in a `call` body | Move `${ uuid }` to a preceding `set` task, reference via `$data` |
| `timestamp` in `wait.until` | Generate in `set`, then `wait: { until: ${ $data.myTimestamp } }` |
| `timestamp_iso8601` anywhere except `set` | Move to `set` |

## Data flow confusion

| Mistake | Explanation |
| --- | --- |
| Using `$output` 3 tasks later | `$output` only holds the MOST RECENT task result. Use `export` to persist to `$context` |
| `export.as: ${ newVal }` loses old context | `export` REPLACES `$context`. Merge: `${ $context + { newVal: . } }` |
| Expecting task input to auto-receive previous output | Chain explicitly via `$output` or `$data.<taskName>` |

## Structure mistakes

| Mistake | Fix |
| --- | --- |
| `do` as a map (YAML object) | `do` must be a list: `do: [ - name: { task } ]` |
| Multiple task keys in one step | Each step has exactly ONE task key |
| `taskQueue: my_queue.v2` | RFC 1123: `my-queue-v2` (no underscores, dots) |
| `workflowType: My Workflow` | RFC 1123: `my-workflow` (no spaces, lowercase) |

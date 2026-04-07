---
sidebar_position: -1
---
# What is Metadata?

## What you will learn

- What metadata is and where it can be applied
- The difference between document-level and task-level metadata
- Which metadata options are available

Metadata is used to provide extra configuration for workflows and tasks.

## Where can I use it?

There are two places where you can use metadata, the [Document](#document)
and the [Task](#task).

Check each metadata for the **location** subheading to see where you can use each
piece of metadata

### Document

The main `document` object has a `metadata` property, which allows you to configure
metadata which is treated as global to the workflow.

```yaml
document:
  dsl: 1.0.0
  taskQueue: zigflow
  workflowType: simple-workflow
  version: 1.0.0
  metadata: {}
```

### Task

Each `task` object has a `metadata` property, which allows you to configure
metadata on a per-task basic.

```yaml
do:
  - example:
      metadata: {}
      set:
        hello: world
```

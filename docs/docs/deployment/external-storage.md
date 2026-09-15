---
title: External storage
sidebar_position: 6
description: "How Zigflow offloads workflow payloads to an external store such as S3 using the Claim Check pattern, including the payload size threshold, the full S3 configuration and what is required to keep all payload values out of Temporal."
---

## What you will learn

- What external storage is and why you might want it
- How the payload size threshold decides what is offloaded
- What is required to keep all payload values out of Temporal
- How to configure the S3 backend
- What a client needs in order to read offloaded payloads
- The operational and security considerations that come with it

## What external storage is

By default, every workflow input, output and activity result is sent to the
Temporal server as a payload and persisted in the workflow history. External
storage changes where that data lives.

External storage is Temporal's implementation of the **Claim Check** pattern,
which Zigflow exposes through its own configuration. Instead of putting the
value in the message, the value is written to a store you control and only a
reference, the claim check, travels through Temporal. Whatever needs the data
redeems that reference to fetch it.

When external storage is configured to offload every relevant payload, the
payload values themselves do not need to be sent to or stored by the Temporal
server. Temporal holds the references and your bucket holds the data.

That is not the default. Out of the box Zigflow only offloads payloads that
meet the configured payload size threshold, so payloads below it remain inline
in Temporal. Payloads submitted by a client that is not configured for the
same external storage also reach Temporal inline. See
[keeping all payload values outside Temporal](#keeping-all-payload-values-outside-temporal)
for the two settings that close both gaps.

Two common reasons to use it:

- **Data residency and confidentiality.** Offloaded payload values stay inside
  your own infrastructure. This matters when the workflow handles data that
  must not leave your account, or when you use Temporal Cloud and would rather
  the provider never held the values at all.
- **Large payloads.** Temporal enforces limits on payload and history size.
  Offloading the large values keeps the history small while still passing the
  data between tasks.

:::info
External storage is **opt-in**. When `--external-storage` is not set, Zigflow
behaves exactly as before and every payload is sent to Temporal inline.
:::

:::warning
External payload storage is an experimental feature of the underlying Temporal
Go SDK. Treat it as such when planning a production rollout.
:::

The only backend currently supported is **S3**, including S3-compatible
stores such as MinIO and RustFS.

---

## When a payload is offloaded

Zigflow measures the size of each serialised payload and compares it against
the payload size threshold. A payload whose serialised size is greater than or
equal to the threshold is written to external storage. Anything smaller is
sent to Temporal inline as usual.

The threshold is set in bytes with
`--external-storage-payload-size-threshold`. When it is not set, the Temporal
SDK default of 256 KiB applies.

The size compared is the serialised Temporal payload, which includes its
metadata, rather than the length of the value you wrote in YAML. Setting the
threshold to `1` therefore offloads every payload, which is useful for
demonstrations and testing.

### Keeping all payload values outside Temporal

The Claim Check guarantee applies to the payloads that are actually offloaded.
For every payload value to stay out of Temporal, both of the following must be
true.

**The threshold is low enough that all relevant payloads are offloaded.** The
default of 256 KiB leaves everything smaller than that inline. Set
`--external-storage-payload-size-threshold` to `1` to offload every payload.

**Every client and worker submitting those payloads uses the same external
storage configuration.** Offloading is performed by whichever process encodes
the payload. A client that starts a workflow, or sends it a signal or an
update, without the same external storage configuration sends those values to
Temporal inline, whatever the worker is configured to do.

Both are needed. A worker that offloads everything does not stop an
unconfigured client writing an inline payload into the same workflow history.

---

## Enabling external storage

Select the backend with `--external-storage`, then supply the settings for
that backend. The value must be `s3`, or left unset to disable the feature.
Any other value is rejected before the worker starts.

```sh
zigflow run \
  -f workflow.yaml \
  --external-storage s3 \
  --external-storage-s3-bucket my-payload-bucket \
  --external-storage-s3-region eu-west-2
```

As with every Zigflow flag, each of these can be supplied as an environment
variable instead. See
[environment variables](/docs/deployment/intro#environment-variables).

---

## S3 configuration

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--external-storage` | `EXTERNAL_STORAGE` | (none) | Backend to use. `s3`, or unset to disable |
| `--external-storage-payload-size-threshold` | `EXTERNAL_STORAGE_PAYLOAD_SIZE_THRESHOLD` | 256 KiB | Size in bytes at or above which a payload is offloaded |
| `--external-storage-s3-bucket` | `EXTERNAL_STORAGE_S3_BUCKET` | (none) | Bucket that payloads are written to. Required |
| `--external-storage-s3-region` | `EXTERNAL_STORAGE_S3_REGION` | (none) | AWS region the bucket is in |
| `--external-storage-s3-driver-name` | `EXTERNAL_STORAGE_S3_DRIVER_NAME` | `aws.s3driver` | Identifier recorded in the history alongside each reference |
| `--external-storage-s3-max_payload_size` | `EXTERNAL_STORAGE_S3_MAX_PAYLOAD_SIZE` | 50 MiB | Largest payload in bytes the driver will accept |
| `--external-storage-s3-endpoint` | `EXTERNAL_STORAGE_S3_ENDPOINT` | (none) | Override the endpoint to target an S3-compatible store |
| `--external-storage-s3-use-path-style` | `EXTERNAL_STORAGE_S3_USE_PATH_STYLE` | `false` | Address the bucket in the request path rather than the hostname |
| `--external-storage-s3-access-key-id` | `EXTERNAL_STORAGE_S3_ACCESS_KEY_ID` | (none) | Access key ID |
| `--external-storage-s3-secret-access-key` | `EXTERNAL_STORAGE_S3_SECRET_ACCESS_KEY` | (none) | Secret access key |
| `--external-storage-s3-session-token` | `EXTERNAL_STORAGE_S3_SESSION_TOKEN` | (none) | Session token for temporary credentials |

:::info
`--external-storage-s3-max_payload_size` uses underscores rather than hyphens
in its final segment. The environment variable follows the usual convention.
:::

A payload larger than the maximum payload size is rejected by the driver
rather than truncated or sent inline, so the operation that produced it
fails. Raise the limit if you legitimately need to store larger values.

### Credentials

The credential flags are optional. When the access key ID and secret access
key are both empty, the AWS SDK resolves credentials through its default
chain, so a worker running under an EC2 instance role, an EKS service account
or any other workload identity needs none of them.

For convenience the three credential settings also accept the standard AWS
variable names:

| Setting | Zigflow variable | AWS fallback |
| --- | --- | --- |
| Access key ID | `EXTERNAL_STORAGE_S3_ACCESS_KEY_ID` | `AWS_ACCESS_KEY_ID` |
| Secret access key | `EXTERNAL_STORAGE_S3_SECRET_ACCESS_KEY` | `AWS_SECRET_ACCESS_KEY` |
| Session token | `EXTERNAL_STORAGE_S3_SESSION_TOKEN` | `AWS_SESSION_TOKEN` |

The Zigflow-prefixed variable takes precedence when both are set.

A session token may only be supplied alongside both an access key ID and a
secret access key. Supplying it on its own fails at startup.

### S3-compatible storage

To use a store other than S3 itself, set the endpoint and enable path-style
addressing:

```sh
zigflow run \
  -f workflow.yaml \
  --external-storage s3 \
  --external-storage-s3-bucket zigflow \
  --external-storage-s3-region us-east-1 \
  --external-storage-s3-endpoint http://minio:9000 \
  --external-storage-s3-use-path-style
```

Most S3-compatible servers cannot resolve the virtual-host form
(`bucket.host`), so path-style addressing is usually required. Omitting the
endpoint sends the requests to the real AWS endpoint for the configured
region, which is a common cause of unexpected `403 Forbidden` responses
against a bucket you do not own.

---

## Reading offloaded payloads

:::tip
The
[hello-world-external-storage example](https://github.com/zigflow/zigflow/tree/main/examples/hello-world-external-storage)
shows both halves: a worker that offloads payloads to a local S3-compatible
store, and a Go client configured to read them back.
:::

A payload that has been offloaded is only readable by something that has the
same storage driver configured. The worker started by `zigflow run` has it, so
workflows and activities see their data exactly as they would without external
storage.

Anything else reading the workflow sees the reference rather than the value.
That includes the Temporal UI, the `temporal` CLI and any application client
that starts workflows or reads their results. A client written with a Temporal
SDK needs the same bucket, endpoint and credentials configured before it can
resolve a result.

---

## Operational considerations

**The bucket must already exist.** Zigflow does not create it. The driver is
built when the worker starts, but nothing contacts the bucket at that point,
so a missing bucket, a wrong endpoint or invalid credentials only surface when
the first payload is offloaded.

**Offloaded objects have no managed lifecycle.** Temporal's external storage
mechanism does not currently manage the lifecycle of offloaded objects, so
Zigflow does not delete or prune the objects written to your store. They
remain there until your own lifecycle policy or cleanup process removes them,
and without one, storage usage keeps growing for as long as workflows keep
running.

Configure an S3 lifecycle policy, or the equivalent for your store, and make
its expiry compatible with how long the corresponding workflow histories need
to remain usable. There is no retention period worth recommending here: the
right value falls out of your Temporal namespace retention, archival,
debugging and compliance requirements, so derive it from those.

:::warning
Do not expire an object that is still referenced by a workflow history you
need to inspect, replay or otherwise retain. An offloaded payload exists only
in the bucket, so deleting it makes that part of the history unreadable and
the workflow unreplayable. This is not recoverable.
:::

**Do not change the driver name after deployment.** The name is recorded in
the history next to every reference and is used to pick the driver on the way
back. Renaming it makes previously stored payloads unreadable.

**The bucket holds your workflow data.** Anything you would otherwise rely on
Temporal to protect is now your responsibility: bucket policy, encryption at
rest, versioning and access logging. Grant the worker only the S3 permissions
it needs on that bucket.

**A history can hold a mix of inline and offloaded payloads.** Only the
payloads that met the threshold, and that were submitted by a process
configured for external storage, are in the bucket. If you are relying on
external storage to keep values away from Temporal, confirm both conditions in
[keeping all payload values outside Temporal](#keeping-all-payload-values-outside-temporal).

**Credentials are hidden from help output.** When a credential is resolved
from the environment, `zigflow run --help` displays `***` in place of the
value rather than echoing the secret.

---

## Troubleshooting

**`403 Forbidden` on a bucket you do own.** The endpoint is probably not set,
so the request went to the real AWS endpoint for the configured region rather
than to your S3-compatible store. Set `--external-storage-s3-endpoint` and
`--external-storage-s3-use-path-style`.

**Nothing appears in the bucket.** Check that `--external-storage` is set to
`s3`. Without it the worker starts normally and sends every payload to
Temporal inline. If the backend is set, the payloads may simply be below the
threshold; set `--external-storage-payload-size-threshold` to `1` to confirm
the wiring before tuning it. If the worker is offloading but the payloads a
client submits are not, that client needs the same external storage
configuration.

**`invalid external storage type`.** The value of `--external-storage` is
neither `s3` nor empty. The check runs before any worker starts.

**`SessionToken requires AccessKeyID and SecretAccessKey to also be set`.**
A session token was supplied on its own. Provide all three credential values
or none of them.

**`payload size N exceeds maximum M`.** The payload is larger than
`--external-storage-s3-max_payload_size`, which defaults to 50 MiB. The
payload is not sent inline as a fallback, so raise the limit if the value is
legitimate.

**A client cannot read the workflow result.** Clients need the same storage
driver as the worker. See
[reading offloaded payloads](#reading-offloaded-payloads).

---

## Related pages

- [Deploying Zigflow](/docs/deployment/intro): connection flags and
  environment variables
- [CLI reference](/docs/cli/commands/zigflow_run): the complete
  `zigflow run` flag list
- [Observability](/docs/deployment/observability): health checks, metrics and
  CloudEvents

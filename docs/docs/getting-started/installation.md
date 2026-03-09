---
sidebar_position: 1
---
# Installation

Zigflow is a single Go binary with prebuilt options.

## What you will learn

- How to install the Zigflow binary on your platform
- How to run Zigflow from a Docker image
- How to install from source

## Binary Releases

Every [release](https://github.com/zigflow/zigflow/releases) of Zigflow provides
binary releases for a variety of OSes. These binary versions can be manually
downloaded and installed.

1. Download your [desired version](https://github.com/zigflow/zigflow/releases)
2. Make it executable `chmod +x ./path/to/binary`
3. (Optional) Move to your `$PATH` directory

## Docker Image

Every [release](https://github.com/zigflow/zigflow/pkgs/container/zigflow) of
Zigflow provides a Docker image. The binary is set as the
[entrypoint](https://docs.docker.com/reference/dockerfile/#entrypoint), so you
can use the image as a replacement for the binary.

A `latest` tag is maintained for the most recent tag, or you can use the version
as the tag (eg, `0.1.0`).

```sh
docker run -it --rm \
  -v /path/to/workflow.yaml:/app/workflow.yaml \
  ghcr.io/zigflow/zigflow \
  run
```

## Go Install

If you already have [Go](https://go.dev/doc/install) installed, you can use the
Go package manager to install the binary. This will be installed under your
`$GOPATH`.

```sh
go install github.com/zigflow/zigflow@latest
```

You can specify a version by changing `@latest` to the desired version.

### From Source

:::tip
You will need to install [Go](https://go.dev/doc/install)
:::

Building from source is useful for testing unreleased versions.

```sh
git clone https://github.com/zigflow/zigflow.git
cd zigflow
go build .
```

This will fetch the dependencies and build the binary. It will compile it to
`./zigflow`.

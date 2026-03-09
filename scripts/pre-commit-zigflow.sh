#!/usr/bin/env bash
# Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

# pre-commit provides the rev (e.g. v0.8.1)
VERSION="${PRE_COMMIT_REV}"
VERSION="${VERSION#v}"   # strip leading v

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Match goreleaser name_template
case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  amd64) ARCH="x86_64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  i386|i686) ARCH="i386" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux)   OS="linux" ;;
  darwin)  OS="darwin" ;;
  msys*|mingw*|cygwin*)
    OS="windows"
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

CACHE_DIR="${PRE_COMMIT_HOME:-$HOME/.cache/pre-commit}/zigflow"
BIN_NAME="zigflow_${OS}_${ARCH}"
BIN_PATH="${CACHE_DIR}/${BIN_NAME}-${VERSION}"

if [ ! -f "$BIN_PATH" ]; then
  mkdir -p "$CACHE_DIR"

  URL="https://github.com/zigflow/zigflow/releases/download/v${VERSION}/${BIN_NAME}"

  echo "Downloading Zigflow ${VERSION} (${OS}/${ARCH})..."
  curl -sSL -o "$BIN_PATH" "$URL"

  chmod +x "$BIN_PATH"
fi

exec "$BIN_PATH" graph inject "$@"

#!/usr/bin/env sh
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

set -eu

# This script installs Zigflow from GitHub Releases.
#
# It is intended as a convenience installer for local development, CI runners,
# containers and short-lived environments. For long-lived production systems,
# prefer a package manager where available, such as Homebrew on macOS.
#
# Usage:
#
#   curl -fsSL https://get.zigflow.dev -o get-zigflow.sh
#   sh get-zigflow.sh
#
# Or:
#
#   curl -fsSL https://get.zigflow.dev | sh
#
# Optional environment variables:
#
#   ZIGFLOW_VERSION=v0.12.0
#     Install a specific Zigflow release. Defaults to "latest".
#
#   ZIGFLOW_INSTALL_DIR="$HOME/.local/bin"
#     Install Zigflow into a custom directory. Defaults to "/usr/local/bin".
#
#   ZIGFLOW_SKIP_CHECKSUM=true
#     Skip SHA-256 checksum verification. Not recommended.
#
#   ZIGFLOW_REPO=zigflow/zigflow
#     Override the GitHub repository used for releases.
#
# Examples:
#
#   curl -fsSL https://get.zigflow.dev | sh
#   curl -fsSL https://get.zigflow.dev | ZIGFLOW_VERSION=v0.12.0 sh
#   curl -fsSL https://get.zigflow.dev | ZIGFLOW_INSTALL_DIR="$HOME/.local/bin" sh
#
# The script detects the current operating system and architecture, downloads
# the matching Zigflow binary, verifies its checksum and installs it as
# "zigflow".

REPO="${ZIGFLOW_REPO:-zigflow/zigflow}"
VERSION="${ZIGFLOW_VERSION:-latest}"
INSTALL_DIR="${ZIGFLOW_INSTALL_DIR:-/usr/local/bin}"
SKIP_CHECKSUM="${ZIGFLOW_SKIP_CHECKSUM:-false}"

BINARY_NAME="zigflow"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'zigflow install error: %s\n' "$*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

need curl
need uname
need chmod
need mkdir
need cp

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m | tr '[:upper:]' '[:lower:]')"

case "$OS" in
  linux)
    OS="linux"
    ;;
  darwin)
    OS="darwin"
    ;;
  mingw*|msys*|cygwin*)
    OS="windows"
    ;;
  *)
    fail "unsupported operating system: $OS"
    ;;
esac

case "$ARCH" in
  x86_64|amd64)
    ARCH="x86_64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  i386|i686)
    ARCH="i386"
    ;;
  *)
    fail "unsupported architecture: $ARCH"
    ;;
esac

if [ "$OS" = "darwin" ] && [ "$ARCH" = "i386" ]; then
  fail "unsupported platform: darwin/i386"
fi

EXT=""
if [ "$OS" = "windows" ]; then
  EXT=".exe"
fi

ASSET="zigflow_${OS}_${ARCH}${EXT}"
TARGET="$INSTALL_DIR/$BINARY_NAME$EXT"

if [ "$VERSION" = "latest" ]; then
  BASE_URL="https://github.com/$REPO/releases/latest/download"
else
  case "$VERSION" in
    v*)
      ;;
    *)
      VERSION="v$VERSION"
      ;;
  esac

  BASE_URL="https://github.com/$REPO/releases/download/$VERSION"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

download() {
  url="$1"
  out="$2"

  curl -fsSL "$url" -o "$out"
}

log "Installing Zigflow"
log "Repository: $REPO"
log "Version:    $VERSION"
log "Platform:   $OS/$ARCH"
log "Asset:      $ASSET"

download "$BASE_URL/$ASSET" "$TMP_DIR/$BINARY_NAME$EXT"

if [ "$SKIP_CHECKSUM" = "true" ] || [ "$SKIP_CHECKSUM" = "1" ]; then
  log "Skipping checksum verification"
else
  need grep
  need awk

  if command -v sha256sum >/dev/null 2>&1; then
    SHA256_CMD="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    SHA256_CMD="shasum -a 256"
  else
    fail "missing checksum tool: install sha256sum or shasum, or set ZIGFLOW_SKIP_CHECKSUM=true"
  fi

  log "Downloading checksums"
  download "$BASE_URL/checksums.txt" "$TMP_DIR/checksums.txt"

  EXPECTED="$(grep "  $ASSET\$" "$TMP_DIR/checksums.txt" | awk '{print $1}')"

  if [ -z "$EXPECTED" ]; then
    fail "checksum not found for $ASSET"
  fi

  ACTUAL="$($SHA256_CMD "$TMP_DIR/$BINARY_NAME$EXT" | awk '{print $1}')"

  if [ "$EXPECTED" != "$ACTUAL" ]; then
    fail "checksum mismatch for $ASSET"
  fi

  log "Checksum verified"
fi

chmod +x "$TMP_DIR/$BINARY_NAME$EXT"

mkdir -p "$INSTALL_DIR"

if ! cp "$TMP_DIR/$BINARY_NAME$EXT" "$TARGET" 2>/dev/null; then
  fail "could not install to $TARGET. Try running with sudo or set ZIGFLOW_INSTALL_DIR to a writable directory"
fi

log "Installed Zigflow to $TARGET"

if command -v "$TARGET" >/dev/null 2>&1; then
  "$TARGET" version || true
else
  "$TARGET" version || true
fi

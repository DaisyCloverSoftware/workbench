#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

mkdir -p "$HOME/.local/bin"

have_usable_go() {
  command -v go >/dev/null 2>&1 || return 1
  local v
  v="$(go env GOVERSION 2>/dev/null || true)"
  printf '%s\n' "$v" | grep -Eq '^go([2-9]|1\.(2[3-9]|[3-9][0-9]))([.]|$)'
}

download() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fL --retry 3 --connect-timeout 15 "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -O "$dest" "$url"
  else
    echo "Workbench Runner needs curl or wget for one-time Go bootstrap." >&2
    exit 1
  fi
}

bootstrap_go() {
  local version arch machine base tmp archive
  if command -v curl >/dev/null 2>&1; then
    version="$(curl -fsSL --retry 3 --connect-timeout 15 'https://go.dev/VERSION?m=text' | head -n 1 | tr -d '\r\n')"
  elif command -v wget >/dev/null 2>&1; then
    version="$(wget -qO- 'https://go.dev/VERSION?m=text' | head -n 1 | tr -d '\r\n')"
  else
    echo "Workbench Runner needs curl or wget for one-time Go bootstrap." >&2
    exit 1
  fi

  case "$version" in
    go1.*) ;;
    *) echo "Could not determine a stable Go toolchain from go.dev (got: $version)." >&2; exit 1 ;;
  esac

  machine="$(uname -m)"
  case "$machine" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    armv6l|armv7l) arch="armv6l" ;;
    *) echo "Unsupported CPU architecture for automatic Go bootstrap: $machine" >&2; exit 1 ;;
  esac

  base="$HOME/.local/share/workbench/toolchains/$version"
  if [ ! -x "$base/go/bin/go" ]; then
    echo "Go 1.23+ was not found; bootstrapping $version locally (no sudo required)..."
    tmp="$(mktemp -d)"
    trap 'rm -rf "${tmp:-}"' EXIT
    archive="$tmp/$version.linux-$arch.tar.gz"
    download "https://go.dev/dl/$version.linux-$arch.tar.gz" "$archive"
    rm -rf "$base"
    mkdir -p "$base"
    tar -xzf "$archive" -C "$base"
  fi
  export PATH="$base/go/bin:$PATH"
}

if ! have_usable_go; then
  bootstrap_go
fi

echo "Using $(go version)"
echo "Testing Workbench..."
go test ./...

echo "Building cluster runner..."
go build -trimpath -o "$HOME/.local/bin/workbench-runner" ./cmd/workbench-runner
chmod 0755 "$HOME/.local/bin/workbench-runner"

echo
"$HOME/.local/bin/workbench-runner" doctor

echo
printf 'Installed: %s\n' "$HOME/.local/bin/workbench-runner"
echo "Workbench desktop can now use this node over the same SSH host configured for OpenClaw."

#!/usr/bin/env bash
set -euo pipefail

for name in workbench-relay workbench-runner workbench-server; do
  bin="$HOME/.local/bin/$name"
  printf '=== %s ===\n' "$name"
  if [[ ! -x "$bin" ]]; then
    echo 'missing'
    continue
  fi
  stat -c 'mtime=%y size=%s sha256=' "$bin" | tr -d '\n'
  sha256sum "$bin" | awk '{print $1}'
  if command -v go >/dev/null 2>&1; then
    go version -m "$bin" || true
  fi
done

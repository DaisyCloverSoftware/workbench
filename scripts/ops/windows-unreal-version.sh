#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: windows-unreal-version.sh <windows-host-id>" >&2
  exit 2
fi

host_id="$1"
if [[ ! "$host_id" =~ ^windows_[a-z0-9_-]{8,95}$ ]]; then
  echo "invalid Windows host id" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

if command -v go >/dev/null 2>&1; then
  go_bin="$(command -v go)"
else
  go_bin="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
fi
[ -n "${go_bin:-}" ] && [ -x "$go_bin" ] || {
  echo "Workbench Go toolchain is unavailable" >&2
  exit 1
}

exec "$go_bin" run ./cmd/workbench-unreal-version-submit "$host_id"

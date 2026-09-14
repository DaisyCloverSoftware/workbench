#!/usr/bin/env bash
# Only a registered host id is accepted. All build inputs are compiled constants.
set -euo pipefail
[[ $# -eq 1 && "$1" =~ ^windows_[a-z0-9_-]{8,87}$ ]] || { echo 'one registered Windows host id is required' >&2; exit 2; }
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
if command -v go >/dev/null 2>&1; then
  go_bin="$(command -v go)"
else
  go_bin="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
fi
[[ -n "${go_bin:-}" && -x "$go_bin" ]] || { echo 'Workbench Go toolchain is unavailable' >&2; exit 1; }
exec "$go_bin" run ./cmd/workbench-override-pr97-submit "$1"

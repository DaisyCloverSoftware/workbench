#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

find_tool() {
  local name="$1"
  local found=""
  found="$(command -v "$name" 2>/dev/null || true)"
  if [[ -n "$found" ]]; then
    printf '%s\n' "$found"
    return 0
  fi
  for candidate in "/usr/local/go/bin/$name" "/usr/lib/go/bin/$name" "/usr/bin/$name"; do
    if [[ -x "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  return 1
}

GO_BIN="$(find_tool go)" || { echo 'Go compiler not found on Workbench host' >&2; exit 127; }
GOFMT_BIN="$(find_tool gofmt)" || { echo 'gofmt not found on Workbench host' >&2; exit 127; }

format_diff="$("$GOFMT_BIN" -d \
  internal/core/machine_operations.go \
  internal/core/owner_gated_product_mutations.go \
  internal/core/owner_gated_product_mutations_test.go \
  internal/core/operations_script.go \
  internal/core/owner_gated_operations_source.go \
  internal/core/owner_gated_operations_source_test.go)"
if [[ -n "$format_diff" ]]; then
  printf '%s\n' "$format_diff" >&2
  echo 'gofmt check failed' >&2
  exit 1
fi

"$GO_BIN" test ./internal/core

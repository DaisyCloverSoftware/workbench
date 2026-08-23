#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

format_diff="$(gofmt -d internal/core/machine_operations.go internal/core/owner_gated_product_mutations.go internal/core/owner_gated_product_mutations_test.go)"
if [[ -n "$format_diff" ]]; then
  printf '%s\n' "$format_diff" >&2
  echo 'gofmt check failed' >&2
  exit 1
fi

go test ./internal/core

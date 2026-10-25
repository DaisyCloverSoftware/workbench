#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-sha>" >&2
  exit 64
fi
SHA="$1"
[[ "$SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase SHA required" >&2; exit 64; }
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-api-ci-check.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "RUM_API_BOUNDED_CHECK_BLOCKED=SOURCE_UNAVAILABLE" >&2; exit 78; }

log="$(mktemp)"
trap 'rm -f "$log"' EXIT HUP INT TERM
if bash "$SOURCE" "$SHA" >"$log" 2>&1; then
  tail -n 80 "$log"
  printf 'RUM_API_BOUNDED_CHECK_STATUS=PASS\n'
else
  status=$?
  printf 'RUM_API_BOUNDED_CHECK_STATUS=FAIL exit=%s\n' "$status" >&2
  tail -n 120 "$log" >&2
  exit "$status"
fi

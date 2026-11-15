#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat171-judge-api-isolated.sh scripts/ops/deploy-rat171-judge-frontend-isolated.sh scripts/ops/verify-rat171-community-verdict-browser-20261115.sh; do
  [[ -f "$script" && ! -L "$script" ]] || exit 66
  bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat171-community-verdict-browser-20261115.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
echo RAT171_SCRIPT_SYNTAX_PASS=true

#!/usr/bin/env bash
set -euo pipefail
# Read-only source syntax check. Does not touch RAT data or environments.
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat171-judge-api-isolated.sh scripts/ops/deploy-rat171-judge-frontend-isolated.sh scripts/ops/verify-rat171-community-verdict-browser-20261115.sh;do
 [[ -f "$script" && ! -L "$script" ]] || exit 66
 bash -n "$script"
done
temp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$temp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat171-community-verdict-browser-20261115.sh | sed '$d' > "$temp"
[[ -s "$temp" ]] || exit 65
node --check "$temp"
echo RAT171_JUDGE_DELIVERY_SCRIPT_SYNTAX_OK=true

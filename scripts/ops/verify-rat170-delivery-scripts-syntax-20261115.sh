#!/usr/bin/env bash
set -euo pipefail
# Source-only admission check; no Kubernetes, network or application mutations.
[[ $# -eq 0 ]] || exit 64
for path in scripts/ops/deploy-rat-search-api-repair.sh scripts/ops/deploy-rat-thingtype-frontend-isolated.sh scripts/ops/verify-rat170-direct-community-images-browser-20261115.sh scripts/ops/inspect-rat170-community-owner-images-readonly-20261115.sh scripts/ops/verify-rat170-avatar-readonly-browser-20261115.sh scripts/ops/verify-rat170-replacement-rollback-browser-20261115.sh scripts/ops/verify-rat170-processed-image-reuse-browser-20261115.sh scripts/ops/verify-rat170-no-match-regression-20261115.sh; do
  [[ -f "$path" && ! -L "$path" ]] || exit 66
  bash -n "$path"
done
target="scripts/ops/verify-rat170-direct-community-images-browser-20261115.sh"
temp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$temp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' "$target" | sed '$d' > "$temp"
[[ -s "$temp" ]] || { echo "JavaScript heredoc extraction failed" >&2; exit 65; }
command -v node >/dev/null || exit 69
node --check "$temp"
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat170-replacement-rollback-browser-20261115.sh | sed '$d' > "$temp"
[[ -s "$temp" ]] || exit 65
node --check "$temp"
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat170-processed-image-reuse-browser-20261115.sh | sed '$d' > "$temp"
[[ -s "$temp" ]] || exit 65
node --check "$temp"
echo "RAT170_DELIVERY_SCRIPTS_SYNTAX_PASS=true"

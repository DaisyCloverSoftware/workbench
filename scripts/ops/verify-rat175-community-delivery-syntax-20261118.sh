#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only syntax for the two guarded, fail-closed API/RAT-only rollout scripts.
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat175-community-api-isolated-20261118.sh scripts/ops/deploy-rat175-community-frontend-isolated-20261118.sh scripts/ops/verify-rat175-community-cj-browser-20261118.sh scripts/ops/verify-rat175-community-qa-image-browser-20261118.sh scripts/ops/export-rat175-community-qa-screenshot-readonly-20261118.sh scripts/ops/export-rat175-community-cj-error-readonly-20261118.sh; do
 [[ -f "$script" && ! -L "$script" ]] || exit 66
 bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
for browser in scripts/ops/verify-rat175-community-cj-browser-20261118.sh scripts/ops/verify-rat175-community-qa-image-browser-20261118.sh; do
 sed -n '/^import assert from /,/^NODE$/p' "$browser" | sed '$d' > "$tmp"
 [[ -s "$tmp" ]] || exit 65
 node --check "$tmp"
done
python3 - <<'PY'
from pathlib import Path
for path in ("scripts/ops/deploy-rat175-community-api-isolated-20261118.sh", "scripts/ops/deploy-rat175-community-frontend-isolated-20261118.sh", "scripts/ops/export-rat175-community-qa-screenshot-readonly-20261118.sh", "scripts/ops/export-rat175-community-cj-error-readonly-20261118.sh"):
  source=Path(path).read_text()
  for block in source.split("<<'PY'\n")[1:]:
    compile(block.split("\nPY\n",1)[0],path,'exec')
  if "deploy-rat175-community" in path:
    assert "9c8af9fe07e9a0c4b0b1896f4be1903719b781e3" in source
    assert "rollback()" in source
print("RAT175_COMMUNITY_DELIVERY_PYTHON_SYNTAX_PASS=true")
PY
echo RAT175_COMMUNITY_DELIVERY_SYNTAX_PASS=true

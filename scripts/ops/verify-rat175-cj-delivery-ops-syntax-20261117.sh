#!/usr/bin/env bash
set -Eeuo pipefail
# Source-only syntax of reviewed RAT175 deployment+browser operations.
[[ $# -eq 0 ]] || exit 64
for script in \
 scripts/ops/deploy-rat175-cj-image-api-isolated-20261117.sh \
 scripts/ops/deploy-rat175-cj-image-frontend-isolated-20261117.sh \
 scripts/ops/verify-rat175-cj-image-invalid-url-browser-20261117.sh; do
 [[ -f "$script" && ! -L "$script" ]] || exit 66
 bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat175-cj-image-invalid-url-browser-20261117.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PYCHECK'
from pathlib import Path
for path in [
"scripts/ops/deploy-rat175-cj-image-api-isolated-20261117.sh",
"scripts/ops/deploy-rat175-cj-image-frontend-isolated-20261117.sh",
"scripts/ops/verify-rat175-cj-image-invalid-url-browser-20261117.sh",
]:
 src=Path(path).read_text()
 for block in src.split("<<'PY'\n")[1:]:
  compile(block.split("\nPY\n",1)[0],path,'exec')
print("RAT175_CJ_OPS_PY_SYNTAX_PASS=true")
PYCHECK
echo RAT175_CJ_OPS_SYNTAX_PASS=true

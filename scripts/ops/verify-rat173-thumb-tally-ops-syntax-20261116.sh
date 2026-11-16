#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat173-judge-thumb-tallies-frontend-isolated-20261116.sh scripts/ops/verify-rat173-judge-thumb-tallies-browser-20261116.sh; do
  [[ -f "$script" && ! -L "$script" ]] || exit 66
  bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat173-judge-thumb-tallies-browser-20261116.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PYCHECK'
from pathlib import Path
for path in ['scripts/ops/deploy-rat173-judge-thumb-tallies-frontend-isolated-20261116.sh','scripts/ops/verify-rat173-judge-thumb-tallies-browser-20261116.sh']:
    source=Path(path).read_text()
    blocks=source.split("<<'PY'\n")[1:]
    assert blocks, f'No Python health guard found: {path}'
    for block in blocks:
        compile(block.split("\nPY\n",1)[0],path,'exec')
print('RAT173_THUMB_TALLY_PYTHON_SYNTAX_PASS=true')
PYCHECK
echo RAT173_THUMB_TALLY_OPS_SYNTAX_PASS=true

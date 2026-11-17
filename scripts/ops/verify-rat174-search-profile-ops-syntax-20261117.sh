#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat174-search-profile-api-isolated-20261117.sh scripts/ops/verify-rat174-search-profile-browser-20261117.sh; do
  [[ -f "$script" && ! -L "$script" ]] || exit 66
  bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)";trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat174-search-profile-browser-20261117.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PY'
from pathlib import Path
for path in ('scripts/ops/deploy-rat174-search-profile-api-isolated-20261117.sh','scripts/ops/verify-rat174-search-profile-browser-20261117.sh'):
    data=Path(path).read_text()
    for block in data.split("<<'PY'\n")[1:]:
        compile(block.split('\nPY\n',1)[0],path,'exec')
print('RAT174_SEARCH_PROFILE_PYTHON_SYNTAX_PASS=true')
PY
echo RAT174_SEARCH_PROFILE_OPS_SYNTAX_PASS=true

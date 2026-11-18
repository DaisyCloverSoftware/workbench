#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
for path in \
 scripts/ops/deploy-rat178-semantic-linked-categories-api-isolated-20261118.sh \
 scripts/ops/verify-rat178-semantic-linked-search-browser-20261118.sh \
 scripts/ops/export-rat178-semantic-browser-screenshot-readonly-20261118.sh;do
 [[ -f "$path" && ! -L "$path" ]] || exit 66
 bash -n "$path"
done
temp="$(mktemp --suffix=.mjs)";trap 'rm -f "$temp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat178-semantic-linked-search-browser-20261118.sh | sed '$d' > "$temp"
[[ -s "$temp" ]] || exit 66
node --check "$temp"
python3 - <<'PY'
from pathlib import Path
for p in ('deploy-rat178-semantic-linked-categories-api-isolated-20261118.sh','verify-rat178-semantic-linked-search-browser-20261118.sh','export-rat178-semantic-browser-screenshot-readonly-20261118.sh'):
 src=Path('scripts/ops/'+p).read_text()
 for part in src.split("<<'PY'\n")[1:]:
  compile(part.split('\nPY\n',1)[0],p,'exec')
print('RAT178_GUARDED_RELEASE_SCRIPTS_SYNTAX_PASS=true')
PY

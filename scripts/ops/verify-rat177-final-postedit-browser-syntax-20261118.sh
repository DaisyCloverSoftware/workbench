#!/usr/bin/env bash
set -Eeuo pipefail
# Pure syntax check for scoped final PR177 browser and screenshot exporter.
[[ $# -eq 0 ]] || exit 64
for src in scripts/ops/verify-rat177-cj-postedit-browser-final-20261118.sh scripts/ops/export-rat177-cj-accepted-screenshot-readonly-20261118.sh; do
 [[ -f "$src" && ! -L "$src" ]] || exit 66
 bash -n "$src"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat177-cj-postedit-browser-final-20261118.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PY'
from pathlib import Path
for name in ('verify-rat177-cj-postedit-browser-final-20261118.sh','export-rat177-cj-accepted-screenshot-readonly-20261118.sh'):
 s=Path('scripts/ops/'+name).read_text()
 for block in s.split("<<'PY'\n")[1:]:
  compile(block.split('\nPY\n',1)[0],name,'exec')
assert '887b5e02f4be6ec7e52371666effa9eb47892d6e' in Path('scripts/ops/verify-rat177-cj-postedit-browser-final-20261118.sh').read_text()
print('RAT177_FINAL_BROWSER_SYNTAX_PASS=true')
PY

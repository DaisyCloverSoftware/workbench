#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
script="scripts/ops/verify-rat177-public-status-browser-readonly-20261118.sh"
[[ -f "$script" && ! -L "$script" ]] || exit 66
bash -n "$script"
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' "$script" | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - "$script" <<'PY'
import sys,pathlib
s=pathlib.Path(sys.argv[1]).read_text()
for p in s.split("<<'PY'\n")[1:]:
 compile(p.split("\nPY\n",1)[0],sys.argv[1],'exec')
assert "712cbd7ba1a77c4b723cf312e59f1265f4cf5df6" in s
assert "r.abort('blockedbyclient')" in s
print("RAT177_PUBLIC_READONLY_CHROMIUM_SYNTAX_PASS=true")
PY
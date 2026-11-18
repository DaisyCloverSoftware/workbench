#!/usr/bin/env bash
set -Eeuo pipefail
# Syntax-check source-pinned, fail-closed RAT176 deployment and browser, no user data.
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat176-compact-vote-frontend-isolated-20261118.sh scripts/ops/verify-rat176-compact-vote-browser-20261118.sh; do
 [[ -f "$script" && ! -L "$script" ]] || exit 66
 bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat176-compact-vote-browser-20261118.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PY'
from pathlib import Path
p=Path("scripts/ops/deploy-rat176-compact-vote-frontend-isolated-20261118.sh")
s=p.read_text()
for block in s.split("<<'PY'\n")[1:]:
  compile(block.split("\nPY\n",1)[0],str(p),"exec")
assert "67d19d785c752c9a692a8e0a1aa67da9f7d4d508" in s
assert "ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71" in s
assert "rollback()" in s and "protected_snapshot()" in s
print("RAT176_COMPACT_OPS_SAFE_PYTHON_SYNTAX_PASS=true")
PY
echo RAT176_COMPACT_OPS_SYNTAX_PASS=true

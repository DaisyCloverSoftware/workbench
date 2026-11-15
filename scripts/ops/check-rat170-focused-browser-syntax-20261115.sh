#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
script="$(dirname "$0")/verify-rat170-focused-search-security-20261115.sh"
bash -n "$script"
temp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$temp"' EXIT
python3 - "$script" "$temp" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text()
assert s.count("<<'NODE'")==1
js=s.split("<<'NODE'",1)[1].split("\nNODE\n",1)[0]
assert "RAT170_FOCUSED_SECURITY_SEARCH_PASS" in js
Path(sys.argv[2]).write_text(js)
PY
node --check "$temp"
echo RAT170_FOCUSED_BROWSER_STATIC_SYNTAX_PASS=true

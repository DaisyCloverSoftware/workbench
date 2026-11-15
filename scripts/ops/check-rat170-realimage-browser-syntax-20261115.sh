#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
source="$(dirname "$0")/verify-rat170-2mib-realimage-browser-20261115.sh"
bash -n "$source"
temp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$temp"' EXIT
python3 - "$source" "$temp" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text()
assert s.count("<<'NODE'")==1
js=s.split("<<'NODE'",1)[1].split("\nNODE\n",1)[0]
assert "RAT170_UPLOAD2MIB_REAL_BROWSER_PASS=true" in js
assert "12*1024*1024" in js
assert "report.newMedia.push" in js
Path(sys.argv[2]).write_text(js)
PY
node --check "$temp"
echo RAT170_REAL_2MIB_ACCEPTANCE_BROWSER_STATIC_SYNTAX_PASS=true

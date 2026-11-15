#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 0 ]] || exit 64
script="$(dirname "$0")/verify-rat170-owner-profile-readonly-browser-20261115.sh"
bash -n "$script"
python3 - "$script" <<'PY'
import pathlib,sys
s=pathlib.Path(sys.argv[1]).read_text()
assert s.count("<<'NODE'")==1 and "\nNODE\n" in s
code=s.split("<<'NODE'",1)[1].split("\nNODE\n",1)[0]
assert len(code)>2000
pathlib.Path('/tmp/rat170-owner-static-check.mjs').write_text(code)
PY
node --check /tmp/rat170-owner-static-check.mjs
rm -f /tmp/rat170-owner-static-check.mjs
echo RAT170_OWNER_READONLY_STATIC_SYNTAX_PASS=true

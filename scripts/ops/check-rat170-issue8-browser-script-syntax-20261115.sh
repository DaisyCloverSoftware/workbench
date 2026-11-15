#!/usr/bin/env bash
set -euo pipefail
# Exact review: parse the already-authored browser verifier's Node heredoc only.
# No browser, network, mutation, credentials, DEV or LIVE access.
[[ $# -eq 0 ]] || exit 64
source_file="$(dirname "$0")/verify-rat170-issue8-withdraw-proxy-browser-20261115.sh"
[[ -f "$source_file" ]] || exit 66
python3 - "$source_file" <<'PY'
from pathlib import Path
import re
import subprocess
import sys
import tempfile
script=Path(sys.argv[1]).read_text(encoding="utf-8")
assert "f4e05b91df0f9b6a3d9c665db851446c6d77dccf" in script
match=re.search(r"node --input-type=module <<'NODE'\n(.*?)\nNODE", script, re.DOTALL)
assert match, "missing browser code block"
with tempfile.TemporaryDirectory() as directory:
    file=Path(directory)/"browser.mjs"
    file.write_text(match.group(1),encoding="utf-8")
    subprocess.run(["node","--check",str(file)],check=True)
print("RAT170_ISSUE8_BROWSER_SCRIPT_SYNTAX_PASS=true")
PY

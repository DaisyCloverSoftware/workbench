#!/usr/bin/env bash
set -euo pipefail
# Source-only check of preauthored Playwright verifier; no network/browser/DEV data changes.
[[ $# -eq 0 ]] || exit 64
source_file="$(dirname "$0")/verify-rat170-thing-images-browser-20261115.sh"
[[ -f "$source_file" ]] || exit 66
python3 - "$source_file" <<'PY'
from pathlib import Path
import re,subprocess,sys,tempfile
s=Path(sys.argv[1]).read_text(encoding='utf-8')
assert '56a2fdfe9ae885f2e8889b5251f0b3c598eab27f' in s
m=re.search(r"node --input-type=module <<'NODE'\n(.*?)\nNODE",s,re.DOTALL)
assert m
with tempfile.TemporaryDirectory() as dir:
 f=Path(dir)/'browser.mjs'
 f.write_text(m.group(1),encoding='utf-8')
 subprocess.run(['node','--check',str(f)],check=True)
print('RAT170_THING_MEDIA_BROWSER_VERIFIER_SYNTAX_PASS=true')
PY

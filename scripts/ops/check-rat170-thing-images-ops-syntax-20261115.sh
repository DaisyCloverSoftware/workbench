#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 0 ]] || exit 64
dir="$(cd "$(dirname "$0")" && pwd)"
for f in deploy-rat170-thing-images-api-migration-20261115.sh deploy-rat170-thing-images-frontend-20261115.sh verify-rat170-thing-images-browser-20261115.sh; do
  bash -n "$dir/$f"
  echo "BASH_SYNTAX_OK=$f"
done
python3 - "$dir/verify-rat170-thing-images-browser-20261115.sh" <<'PY'
import sys,pathlib
text=pathlib.Path(sys.argv[1]).read_text()
assert text.count("<<'NODE'") == 1
start=text.index("<<'NODE'")+len("<<'NODE'")
end=text.index("\nNODE\n",start)
js=text[start:end].lstrip("\n")
assert len(js)>5000
pathlib.Path("/tmp/rat170-thing-images-acceptance-check.mjs").write_text(js)
PY
node --check /tmp/rat170-thing-images-acceptance-check.mjs
rm -f /tmp/rat170-thing-images-acceptance-check.mjs
echo 'RAT170_IMAGE_OPS_STATIC_SYNTAX_PASS=true'

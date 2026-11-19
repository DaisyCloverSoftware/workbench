#!/usr/bin/env bash
set -Eeuo pipefail
# Export only checksum-validated accepted QA screenshots after real Chromium pass.
[[ $# -eq 2 ]] || exit 64
run="$1";name="$2"
[[ "$run" =~ ^ratcj-public-accepted-qa-browser-20261119-r[0-9]+$ ]] || exit 64
case "$name" in
 ratcj-accepted-qa-1440-refreshed.jpg|ratcj-accepted-qa-768-refreshed.jpg|ratcj-accepted-qa-390-refreshed.jpg);;
 *) exit 78;;
esac
dir="/tmp/$run"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$run" "$name" "$dir/$name" <<'PY'
import hashlib,json,sys
from pathlib import Path
a=json.loads(Path(sys.argv[1]).read_text())
assert a["status"]=="PASS" and a["run"]==sys.argv[2] and not a["errors"]
assert len(a["screenshots"])==3 and len(a["checks"])==4
assert a["qaRating"] and a["qaEntity"]
for width in (1440,768,390):assert any(x["name"]=="ratcj-accepted-qa-"+str(width)+"-refreshed.jpg" for x in a["screenshots"])
row=[x for x in a["screenshots"] if x["name"]==sys.argv[3]]
assert len(row)==1
path=Path(sys.argv[4])
assert path.is_file() and not path.is_symlink()
contents=path.read_bytes()
assert 12000<len(contents)<950000 and len(contents)==row[0]["bytes"]
assert hashlib.sha256(contents).hexdigest()==row[0]["sha256"]
print("RATCJ_ACCEPTED_QA_SCREENSHOT_VERIFIED="+row[0]["name"])
print("RATCJ_ACCEPTED_QA_SCREENSHOT_BYTES="+str(row[0]["bytes"]))
print("RATCJ_ACCEPTED_QA_SCREENSHOT_SHA256="+row[0]["sha256"])
PY
printf 'RATCJ_ACCEPTED_QA_SCREENSHOT_BASE64='
base64 -w0 "$dir/$name"
printf '\n'

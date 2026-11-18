#!/usr/bin/env bash
set -Eeuo pipefail
# Export accepted and SHA-checked real Chromium screenshot as base64; read-only.
[[ $# -eq 2 && "$1" == 'ratcj-clean-3width-20261118-r2' ]] || exit 64
case "$2" in
 cj-investigates-1440-search-after-refresh.jpg|cj-investigates-768-search-after-refresh.jpg|cj-investigates-390-search-after-refresh.jpg|cj-investigates-1440-workspace-after-refresh.jpg|cj-investigates-768-workspace-after-refresh.jpg|cj-investigates-390-workspace-after-refresh.jpg) ;;
 *) exit 78 ;;
esac
dir="/tmp/$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$1" "$2" "$dir/$2" <<'PY'
import hashlib,json,sys
from pathlib import Path
report=json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert report["status"]=="PASS" and not report.get("errors") and report["run"]==sys.argv[2]
assert report["source"]=="887b5e02f4be6ec7e52371666effa9eb47892d6e"
assert report["id"]=="01m07d2f9fj05f4ypdbwp087qj"
assert len(report["screenshots"])==6 and len(report["checks"])==7
for width in (1440,768,390):
 assert "search-"+str(width)+"-no-fixtures-and-stable-after-reload" in report["checks"]
 assert "workspace-"+str(width)+"-zero-public-connections-published-images-and-stable-after-reload" in report["checks"]
records=[row for row in report["screenshots"] if row["name"]==sys.argv[3]]
assert len(records)==1
path=Path(sys.argv[4])
assert path.is_file() and not path.is_symlink()
actual=path.read_bytes()
assert len(actual)==records[0]["bytes"] and 12000<len(actual)<900000
assert hashlib.sha256(actual).hexdigest()==records[0]["sha256"]
print("RATCJ_SCREENSHOT_VERIFIED="+records[0]["name"])
print("RATCJ_SCREENSHOT_BYTES="+str(records[0]["bytes"]))
print("RATCJ_SCREENSHOT_SHA256="+records[0]["sha256"])
PY
printf 'RATCJ_SCREENSHOT_BASE64='
base64 -w0 "$dir/$2"
printf '\n'

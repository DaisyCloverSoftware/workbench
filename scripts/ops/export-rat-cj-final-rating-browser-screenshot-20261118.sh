#!/usr/bin/env bash
set -Eeuo pipefail
# Export only accepted real CJ rating browser JPEG screenshots verified by the full acceptance manifest.
[[ $# -eq 2 && "$1" == 'ratcj-owner-rating-final-browser-20261118-r2' ]] || exit 64
case "$2" in
 cj-owner-rating-1440-search-refreshed.jpg|cj-owner-rating-1440-workspace-refreshed.jpg|cj-owner-rating-1440-see-ratings-refreshed.jpg|cj-owner-rating-390-search-refreshed.jpg|cj-owner-rating-390-workspace-refreshed.jpg|cj-owner-rating-390-see-ratings-refreshed.jpg) ;;
 *) exit 78;;
esac
dir="/tmp/$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$1" "$2" "$dir/$2" <<'PY'
import hashlib,json,sys
from pathlib import Path
report=json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert report["status"]=="PASS" and report["run"]==sys.argv[2] and not report["errors"]
assert report["id"]=="01m07d2f9fj05f4ypdbwp087qj"
assert report["keep"]=="01m0mvrxv8c3s12q0w7gwmzt03"
assert len(report["checks"])==7 and len(report["screenshots"])==6
for width in (1440,390):
 for check in ("search-"+str(width)+"-canonical-CJ-no-QA-links-after-refresh",
 "workspace-"+str(width)+"-reputation-one-no-QA-and-photos-after-refresh",
 "see-ratings-"+str(width)+"-anchor-reputation-and-community-verdict-eligibility"):
  assert check in report["checks"]
files=[r for r in report["screenshots"] if r["name"]==sys.argv[3]]
assert len(files)==1
f=Path(sys.argv[4])
assert f.is_file() and not f.is_symlink()
b=f.read_bytes()
assert 12000<len(b)<900000 and len(b)==files[0]["bytes"]
assert hashlib.sha256(b).hexdigest()==files[0]["sha256"]
print("RATCJ_FINAL_RATING_SCREENSHOT="+files[0]["name"])
print("RATCJ_FINAL_RATING_SCREENSHOT_BYTES="+str(len(b)))
print("RATCJ_FINAL_RATING_SCREENSHOT_SHA256="+files[0]["sha256"])
PY
printf 'RATCJ_FINAL_RATING_SCREENSHOT_BASE64='
base64 -w0 "$dir/$2"
printf '\n'

#!/usr/bin/env bash
set -Eeuo pipefail
# Export only accepted real Chromium JPG bytes after all canonical rating checks pass.
[[ $# -eq 2 ]] || exit 64
run="$1";name="$2"
[[ "$run" =~ ^ratcj-rating-final-20261118-r[0-9]+$ ]] || exit 64
case "$name" in
 cj-rating-clean-1440-search-refreshed.jpg|cj-rating-clean-1440-thing-see-ratings-refreshed.jpg|cj-rating-clean-768-search-refreshed.jpg|cj-rating-clean-768-thing-see-ratings-refreshed.jpg|cj-rating-clean-390-search-refreshed.jpg|cj-rating-clean-390-thing-see-ratings-refreshed.jpg) ;;
 *) exit 78;;
esac
dir="/tmp/$run"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$run" "$name" "$dir/$name" <<'PY'
import sys,json,hashlib
from pathlib import Path
report=json.loads(Path(sys.argv[1]).read_text())
assert report["status"]=="PASS" and report["run"]==sys.argv[2] and report["issues"]==[]
assert report["entityId"]=="01m07d2f9fj05f4ypdbwp087qj"
assert report["originalRating"]=="01m0mvrxv8c3s12q0w7gwmzt03"
assert len(report["checks"])==9 and len(report["screenshots"])==6
for width in [1440,768,390]:
 assert 'search-'+str(width)+'-no-ineligible-score-or-test-links-and-refresh' in report["checks"]
 assert 'thing-'+str(width)+'-one-5of5-review-0votes-pending-visible-and-refresh' in report["checks"]
records=[x for x in report["screenshots"] if x["name"]==sys.argv[3]]
assert len(records)==1
path=Path(sys.argv[4]);assert path.is_file() and not path.is_symlink()
blob=path.read_bytes()
assert len(blob)==records[0]["bytes"] and 12000<len(blob)<950000
assert hashlib.sha256(blob).hexdigest()==records[0]["sha256"]
print("RATCJ_VERIFIED_CAPTURE="+records[0]["name"])
print("RATCJ_CAPTURE_BYTES="+str(len(blob)))
print("RATCJ_CAPTURE_SHA256="+records[0]["sha256"])
PY
printf 'RATCJ_CAPTURE_BASE64='
base64 -w0 "$dir/$name"
printf '\n'

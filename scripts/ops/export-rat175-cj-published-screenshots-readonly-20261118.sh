#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only export of genuine, accepted, public CJ desktop/mobile screenshots.
# No credentials, browser storage or unpublished media are returned.
[[ $# -eq 2 && "$1" == "rat175-cj-published-20261118-r1" ]] || exit 64
case "$2" in
 cj-1440-thing-full|cj-1440-thing-avatar|cj-1440-search-full|cj-1440-search-avatar|cj-390-thing-full|cj-390-thing-avatar|cj-390-search-full|cj-390-search-avatar) ;;
 *) exit 78 ;;
esac
dir="/tmp/rat175-cj-published-browser-$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1],encoding="utf-8"))
assert r["status"]=="PASS" and r["run"]=="rat175-cj-published-20261118-r1" and r["errors"]==[]
checks=set(r["checks"])
for needed in ("original-sanitised-CJ-public-160px-webp-loaded-from-canonical-media",
"cj-1440-thing-photo-persists-after-refresh",
"cj-1440-search-photo-persists-without-initials-or-broken-images",
"cj-390-thing-photo-persists-after-refresh",
"cj-390-search-photo-persists-without-initials-or-broken-images",
"independent-new-browser-session-uses-same-real-CJ-photo"):
  assert needed in checks
PY
file="$dir/$2.jpg"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -ge 1200 && "$bytes" -lt 750000 ]] || exit 78
echo "CJ_REAL_SCREENSHOT=$2.jpg"
echo "CJ_REAL_SCREENSHOT_BYTES=$bytes"
echo "CJ_REAL_SCREENSHOT_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf "CJ_REAL_SCREENSHOT_B64="
base64 -w0 "$file"
printf '\n'

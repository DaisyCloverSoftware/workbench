#!/usr/bin/env bash
set -Eeuo pipefail
# Export only verified PR177 Chromium evidence, never arbitrary paths or user data.
[[ $# -eq 2 && "$1" == 'rat177-cj-final-20261118-r1' ]] || exit 64
case "$2" in
 cj-1440-main-card.png|cj-768-main-card.png|cj-390-main-card.png|cj-1440-thing-full.jpg|cj-768-thing-full.jpg|cj-390-thing-full.jpg|cj-1440-search-full.jpg|cj-768-search-full.jpg|cj-390-search-full.jpg) ;;
 *) exit 78 ;;
esac
dir="/tmp/rat177-cj-type-browser-$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$2" <<'PY'
import json,sys
r=json.load(open(sys.argv[1],encoding='utf-8'))
assert r.get('status')=='PASS' and r.get('errors')==[]
assert r.get('run')=='rat177-cj-final-20261118-r1'
assert isinstance(r.get('revisionId'),str) and len(r['revisionId'])==26
assert sys.argv[2] in {s['name'] for s in r.get('screenshots',[])}
assert all(any(x.startswith('CJ-'+w+'-thing-one-RUM-status') for x in r['checks']) for w in ('1440','768','390'))
assert all(any(x.startswith('CJ-'+w+'-search-shows-same-channel') for x in r['checks']) for w in ('1440','768','390'))
PY
file="$dir/$2"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1300 && "$bytes" -lt 800000 ]] || exit 78
echo "RAT177_ACCEPTED_SCREENSHOT=$2"
echo "RAT177_ACCEPTED_SCREENSHOT_BYTES=$bytes"
echo "RAT177_ACCEPTED_SCREENSHOT_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT177_ACCEPTED_BASE64='
base64 -w0 "$file"
printf '\n'

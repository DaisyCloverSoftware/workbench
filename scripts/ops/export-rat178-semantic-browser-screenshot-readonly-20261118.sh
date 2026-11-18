#!/usr/bin/env bash
set -Eeuo pipefail
# Export only accepted real Chromium screenshots from pinned PR178 acceptance.
[[ $# -eq 2 && "$1" == 'rat178-linked-channel-final-20261118-r1' ]] || exit 64
case "$2" in
 professional-idiots-1440-search.jpg|professional-idiots-768-search.jpg|professional-idiots-390-search.jpg|professional-idiots-1440-linked-active.jpg|professional-idiots-768-linked-active.jpg|professional-idiots-390-linked-active.jpg) ;;
 *) exit 78;;
esac
dir="/tmp/rat178-linked-category-browser-$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$1" "$2" <<'PY'
import sys,json
r=json.load(open(sys.argv[1],encoding='utf-8'))
assert r['status']=='PASS' and not r.get('errors') and r['run']==sys.argv[2]
assert r['source']=='e54abfaabde5bbd69b0b97e55e8770754fea3401'
assert 'canonical-api-primary-youtube-channel-categorised-once-with-unchanged-relationship-and-top-linked' in r['checks']
for width in ('1440','768','390'):
 assert 'search-'+width+'-Channels-not-Projects-Thumbnails-Thumb-TopLinked-and-no-overflow' in r['checks']
 assert 'rebase-'+width+'-click-promotes-existing-target-and-persists-after-refresh' in r['checks']
assert sys.argv[3] in {s['name'] for s in r['screenshots']}
PY
file="$dir/$2"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 12000 && "$bytes" -lt 800000 ]] || exit 78
echo "RAT178_VERIFIED_BROWSER_SCREENSHOT=$2"
echo "RAT178_SCREENSHOT_BYTES=$bytes"
echo "RAT178_SCREENSHOT_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT178_SCREENSHOT_BASE64='
base64 -w0 "$file"
printf '\n'

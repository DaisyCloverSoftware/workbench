#!/usr/bin/env bash
set -Eeuo pipefail
# Bounded export of real accepted PR177 Chromium screenshots only.
[[ $# -eq 2 ]] || exit 64
case "$1" in
 rat177-public-header-20261118-r4)
  dir="/tmp/rat177-public-header-$1"
  case "$2" in cj-1440-public-header.png|cj-768-public-header.png|cj-390-public-header.png) ;; *) exit 78;; esac
  ;;
 rat177-cj-postedit-20261118-r1)
  dir="/tmp/rat177-cj-postedit-$1"
  case "$2" in
   cj-1440-public-header.png|cj-768-public-header.png|cj-390-public-header.png|cj-1440-thing-full.jpg|cj-768-thing-full.jpg|cj-390-thing-full.jpg|cj-1440-search-full.jpg|cj-768-search-full.jpg|cj-390-search-full.jpg|cj-1440-saved-type-editor.png) ;;
   *) exit 78;;
  esac
  ;;
 *) exit 78;;
esac
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" "$1" "$2" <<'PY'
import json,sys
r=json.load(open(sys.argv[1],encoding='utf-8'))
assert r.get('status')=='PASS' and not r.get('errors')
assert r.get('run')==sys.argv[2]
assert sys.argv[3] in {x['name'] for x in r.get('screenshots',[])}
if 'postedit' in sys.argv[2]:
 assert 'canonical-CJ-digital-resource-with-primary-YouTube-Channel' in r['checks']
 assert 'new-community-member-editor-shows-saved-type-and-attributable-revision-without-mutating-CJ' in r['checks']
 assert all('CJ-'+w+'-header-about-search-images-refresh-no-overflow' in r['checks'] for w in ('1440','768','390'))
else:
 assert all(any(x.startswith('CJ-'+w+'-one-branded') for x in r['checks']) for w in ('1440','768','390'))
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

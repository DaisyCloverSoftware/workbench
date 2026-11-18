#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only retrieval of accepted real CJ public header captures, no accounts.
[[ $# -eq 2 && "$1" == 'rat177-public-header-20261118-r3' ]] || exit 64
case "$2" in
 cj-1440-public-header.png|cj-768-public-header.png|cj-390-public-header.png) ;;
 *) exit 78 ;;
esac
dir="/tmp/rat177-public-header-$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" <<'PY'
import json,sys
report=json.load(open(sys.argv[1],encoding="utf-8"))
assert report['status']=='PASS' and report['errors']==[]
assert report['run']=='rat177-public-header-20261118-r3'
assert set(report['checks']) >= {
 'CJ-1440-one-branded-pending-or-appropriate-public-empty-state-no-overflow',
 'CJ-768-one-branded-pending-or-appropriate-public-empty-state-no-overflow',
 'CJ-390-one-branded-pending-or-appropriate-public-empty-state-no-overflow',
 'independent-public-browser-context-preserves-header'
}
PY
file="$dir/$2"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1500 && "$bytes" -lt 650000 ]] || exit 78
echo "RAT177_PUBLIC_SCREENSHOT=$2"
echo "RAT177_PUBLIC_BYTES=$bytes"
echo "RAT177_PUBLIC_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf "RAT177_PUBLIC_BASE64="
base64 -w0 "$file"
printf '\n'
#!/usr/bin/env bash
set -Eeuo pipefail
# Bounded, read-only export of accepted, real signed-in RAT Judge QA screenshots.
[[ $# -eq 2 && "$1" == 'rat176-compact-vote-20261118-r1' ]] || exit 64
case "$2" in
 02-voter-compact-card-1440.png|02b-voter-compact-card-768.png|02c-voter-compact-card-390.png|09-owner-compact-card-desktop.png|10-owner-compact-card-mobile.png|11-owner-compact-card-tablet.png|02b-voter-tablet-768.jpg|02c-voter-mobile-390.jpg|06-owner-eligible-voter-desktop-readonly.jpg|07-owner-eligible-voter-mobile-readonly.jpg|08-owner-eligible-voter-tablet-readonly.jpg) ;;
 *) echo 'Screenshot not on RAT176 review allowlist' >&2; exit 78 ;;
esac
dir="/tmp/rat176-compact-vote-browser-$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" <<'PY'
import sys,json
r=json.load(open(sys.argv[1],encoding="utf-8"))
assert r["run"]=="rat176-compact-vote-20261118-r1" and r["status"]=="PASS" and not r["errors"]
steps=set(r["steps"])
required={
 "native-RUM-22px-thumbs-44px-hit-targets-anchored-bottom-right-desktop",
 "compact-vote-controls-inside-card-padding-no-overlap-no-overflow-at-1440-768-390",
 "canonical-Judge-vote-withdrawal-and-refresh-persisted",
 "owner-canonical-vote-totals-unchanged-across-desktop-mobile-tablet"
}
assert required.issubset(steps)
PY
file="$dir/$2"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1000 && "$bytes" -lt 750000 ]] || exit 78
echo "RAT176_ACTUAL_SCREENSHOT=$2"
echo "RAT176_SCREENSHOT_BYTES=$bytes"
echo "RAT176_SCREENSHOT_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT176_SCREENSHOT_B64='
base64 -w0 "$file"
printf '\n'

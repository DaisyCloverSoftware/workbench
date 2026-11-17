#!/usr/bin/env bash
set -Eeuo pipefail
# Read only a pinned, accepted RAT175 QA browser screenshot; no customer credentials.
[[ $# -eq 2 ]] || { echo 'usage: accepted-run screenshot-stem' >&2; exit 64; }
[[ "$1" == 'rat175-cj-invalid-url-20261117-r1' ]] || exit 78
case "$2" in
 cj-20261117-specific-image-error|cj-20261117-image-editor-page) ;;
 *) exit 78 ;;
esac
dir="/tmp/rat175-cj-browser-$1"
[[ -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" <<'PY'
import sys,json
data=json.load(open(sys.argv[1],encoding="utf-8"))
assert data["status"]=="PASS"
assert "real-browser-422-with-actionable-link-validation-and-correlation-id" in data["checked"]
assert "CJ-image-history-readable-and-still-empty-after-refresh" in data["checked"]
assert data["errors"]==[]
PY
file="$dir/$2.jpg"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1000 && "$bytes" -lt 750000 ]] || exit 78
echo "RAT175_CJ_SCREENSHOT_FILENAME=$2.jpg"
echo "RAT175_CJ_SCREENSHOT_BYTES=$bytes"
echo "RAT175_CJ_SCREENSHOT_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT175_CJ_SCREENSHOT_B64='
base64 -w0 "$file"
printf '\n'

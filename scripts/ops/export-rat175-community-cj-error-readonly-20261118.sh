#!/usr/bin/env bash
set -Eeuo pipefail
# Export signed-in CJ's invalid-URL error diagnostic only; no legitimate owner image.
[[ $# -eq 2 && "$1" == 'rat175-community-cj-20261118-r1' ]] || exit 64
case "$2" in
 cj-20261117-specific-image-error|cj-20261117-image-editor-page) ;;
 *) exit 78 ;;
esac
dir="/tmp/rat175-cj-browser-$1"
[[ -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1],encoding="utf-8"))
assert obj["status"]=="PASS" and not obj["errors"]
assert "real-browser-422-with-actionable-link-validation-and-correlation-id" in obj["checked"]
assert "CJ-image-history-readable-without-unauthorised-change-after-refresh" in obj["checked"]
PY
file="$dir/$2.jpg"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1500 && "$bytes" -lt 750000 ]] || exit 78
echo "RAT175_CJ_ERROR_IMAGE=$2.jpg"
echo "RAT175_CJ_ERROR_BYTES=$bytes"
echo "RAT175_CJ_ERROR_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT175_CJ_ERROR_SCREENSHOT_B64='
base64 -w0 "$file"
printf '\n'
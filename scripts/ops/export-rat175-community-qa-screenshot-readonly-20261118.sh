#!/usr/bin/env bash
set -Eeuo pipefail
# Export accepted synthetic QA screenshots only after upload, refresh, search and rollback passed.
[[ $# -eq 2 && "$1" == 'rat175-community-qa-20261118-r1' ]] || exit 64
case "$2" in
 01-qa-thing-profile-published|02-qa-search-published-desktop|03-qa-search-published-mobile) ;;
 *) exit 78 ;;
esac
dir="/tmp/rat175-community-qa-browser-$1"
[[ -f "$dir/report.json" && ! -L "$dir/report.json" ]] || exit 66
python3 - "$dir/report.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1],encoding="utf-8"))
assert obj["status"]=="PASS" and obj.get("qaRolledBack") is True
required={"unclaimed-persona-profile-upload-directly-published-with-canonical-revision","profile-image-persisted-after-reload-and-rendered-in-canonical-thing","same-published-canonical-image-in-rum-rat-search-desktop-mobile","qa-image-rollback-restores-original-blank-profile-and-hides-old-media"}
assert required.issubset(set(obj["steps"])) and obj["errors"]==[]
PY
file="$dir/$2.jpg"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1500 && "$bytes" -lt 650000 ]] || exit 78
echo "RAT175_QA_IMAGE=$2.jpg"
echo "RAT175_QA_BYTES=$bytes"
echo "RAT175_QA_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT175_QA_SCREENSHOT_B64='
base64 -w0 "$file"
printf '\n'
#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 2 ]] || { echo 'usage: <accepted-run-id> <screenshot-stem>' >&2; exit 64; }
[[ "$1" == 'rat173-20261116-parity-thumb-r1' ]] || exit 78
case "$2" in
  01-author-pending-desktop|02-independent-branded-thumbs-unselected|03-independent-sinbin-selected|04-independent-keep-selected|05-independent-withdrawn|06-owner-eligible-voter-desktop-readonly|07-owner-eligible-voter-mobile-readonly|08-owner-eligible-voter-tablet-readonly) ;;
  *) exit 78 ;;
esac
file="/tmp/rat171-judge-browser-$1/$2.jpg"
[[ -f "$file" && ! -L "$file" ]] || { echo 'Screenshot unavailable in accepted Workbench run' >&2; exit 66; }
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1000 && "$bytes" -lt 250000 ]] || exit 78
echo "RAT173_SCREENSHOT_FILENAME=$2.jpg"
echo "RAT173_SCREENSHOT_BYTES=$bytes"
printf 'RAT173_SCREENSHOT_B64='
base64 -w0 "$file"
printf '\n'

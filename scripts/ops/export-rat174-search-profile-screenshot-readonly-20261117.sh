#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only export of accepted public RAT174 browser screenshots, never private data.
[[ $# -eq 2 ]] || { echo 'usage: accepted-run-id screenshot-stem' >&2; exit 64; }
[[ "$1" == 'rat174-search-profile-20261117-r1' ]] || { echo 'Unknown immutable browser acceptance run' >&2; exit 78; }
case "$2" in
  rat174-search-1440-full|rat174-search-1440-main-card|rat174-search-1440-linked-cards|rat174-search-390-full|rat174-search-390-main-card|rat174-search-390-linked-cards) ;;
  *) echo 'Screenshot not allowlisted' >&2; exit 78 ;;
esac
dir="/tmp/rat174-search-profile-$1"
[[ -d "$dir" && ! -L "$dir" && -f "$dir/accepted.txt" && ! -L "$dir/accepted.txt" ]] || exit 66
grep -Fxq 'source=cd1910ba808919d74f748456fd04202da052ddf6' "$dir/accepted.txt" || exit 78
grep -Fxq 'api=ghcr.io/daisycloversoftware/rum-api@sha256:5fc614b352a596e41d35a33f55b9f4b01d4c6c2bfbe2d7dda700d92267761a13' "$dir/accepted.txt" || exit 78
grep -Fxq 'rat=ghcr.io/daisycloversoftware/rum-rate-anything@sha256:31651a79f1bd59c8ff6c04a627e392a10b7756a7579633be81230cf96673ad7f' "$dir/accepted.txt" || exit 78
grep -Fxq 'status=PASS' "$dir/accepted.txt" || exit 78
file="$dir/$2.jpg"
[[ -f "$file" && ! -L "$file" ]] || exit 66
bytes="$(wc -c < "$file" | tr -d ' ')"
[[ "$bytes" -gt 1500 && "$bytes" -lt 750000 ]] || exit 78
echo "RAT174_SCREENSHOT_FILENAME=$2.jpg"
echo "RAT174_SCREENSHOT_BYTES=$bytes"
echo "RAT174_SCREENSHOT_SHA256=$(sha256sum "$file" | cut -d' ' -f1)"
printf 'RAT174_SCREENSHOT_B64='
base64 -w0 "$file"
printf '\n'

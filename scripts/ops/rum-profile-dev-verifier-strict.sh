#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-profile-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "profile verifier unavailable" >&2; exit 78; }

out="$(mktemp)"
trap 'rm -f "$out"' EXIT HUP INT TERM
set +e
bash "$SOURCE" "$1" >"$out" 2>&1
status=$?
set -e
# Preserve all assertions and diagnostics while replacing the potentially large
# screenshot payload with a bounded proof marker for the private relay.
awk '
  /^profile_owner_mobile_png_base64=/ { print "profile_owner_mobile_screenshot_captured=yes"; next }
  { print }
' "$out"
exit "$status"

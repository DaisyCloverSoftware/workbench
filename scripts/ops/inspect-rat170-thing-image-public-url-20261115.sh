#!/usr/bin/env bash
set -euo pipefail
# Bounded read-only public test fixture probe; no user data or state.
[[ $# -eq 0 ]] || exit 64
url='https://upload.wikimedia.org/wikipedia/commons/a/a9/Example.jpg'
result="$(curl -sSI --max-time 12 -H 'Accept: image/jpeg,image/png,image/webp' -H 'User-Agent: RUM-RAT-Safe-Image-Import/1.0' "$url")" || exit 2
printf '%s\n' "$result" | tr -d '\r' | grep -Ei '^(HTTP/|content-type:|content-length:|location:)'
python3 - "$result" <<'PY'
import sys,re
s=sys.argv[1]
assert re.search(r'^HTTP/[^\n]+ 200\b',s,re.M|re.I),s
assert re.search(r'^content-type:\s*image/(?:jpeg|png|webp)',s,re.M|re.I),s
assert not re.search(r'^location:',s,re.M|re.I),s
print('RAT170_PUBLIC_TEST_IMAGE_READY=true')
PY

#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
for u in 'https://httpbin.org/redirect/1' 'https://httpbingo.org/redirect/1'; do
 code="$(curl -sS -I --connect-timeout 5 --max-time 15 -o /dev/null -w '%{http_code}' "$u" || true)"
 echo "RAT170_REDIRECT_FIXTURE=$u STATUS=$code"
done

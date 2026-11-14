#!/usr/bin/env bash
set -euo pipefail
# Scoped, unauthenticated canonical DEV read-only diagnosis after later browser
# Add-entry checks returned no action. No LIVE, account or database mutation.
[[ $# -eq 0 ]] || exit 64
uri='https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/search'
for q in 'The Professional Idiots' 'Violet Glacier Semaphore rat169-youtube-reuse-b06' 'zzzz-no-such-rat-entity-zzzz'; do
  tmp="$(mktemp)"
  trap 'rm -f "$tmp"' EXIT
  code="$(curl -sS --max-time 25 --retry 2 --retry-delay 1 -G --data-urlencode "q=$q" -o "$tmp" -w '%{http_code}' "$uri")"
  python3 - "$q" "$code" "$tmp" <<'PY'
import json,sys
query,code,path=sys.argv[1:]
try:
    with open(path,encoding="utf-8") as f: data=json.load(f)
    meta=data.get("meta") or {}
    active=((data.get("data") or {}).get("active") or {})
    result={"query":query,"http_status":int(code),"found":meta.get("found"),
      "source":meta.get("source"),"privacy":meta.get("privacy"),
      "active_name":active.get("name")}
except Exception as exc:
    result={"query":query,"http_status":int(code),"parse_error":str(exc)}
print("RAT169_CURRENT_CANONICAL_SEARCH="+json.dumps(result,ensure_ascii=False))
PY
  rm -f "$tmp"; trap - EXIT
done
echo 'RAT169_SEARCH_DIAG_READ_ONLY=true'

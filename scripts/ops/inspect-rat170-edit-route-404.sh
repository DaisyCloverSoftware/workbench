#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 1 && "$1" =~ ^[a-z0-9]{26}$ ]] || { echo 'usage: <separate-test-entity-ulid>'; exit 64; }
entity="$1"
host='dev-rum-ra.daisycloversoftware.uk'
canonical='dev-rum.daisycloversoftware.uk'
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-rate-anything-preview get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == "$host" ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == "$canonical" ]] || exit 78

# Anonymous requests differentiate auth middleware (401) from an absent
# Laravel or proxy route (404); no mutation, cookies or credentials used.
for suffix in "/things/$entity/edits" "/things/$entity/connections/search?q=Observed" "/me"; do
  base="https://$host/api/v1/rate-anything"
  tmp="$(mktemp)"
  code="$(curl -sS --max-time 22 -H 'Accept: application/json' -o "$tmp" -w '%{http_code}' "$base$suffix")"
  python3 - "$suffix" "$code" "$tmp" <<'PY'
import json,sys
path,code,file=sys.argv[1:]
data=open(file,'rb').read(3000).decode('utf-8','replace')
try:
    obj=json.loads(data)
    detail=(obj.get('message') or obj.get('error') or str(obj))[:240]
except Exception:
    detail=data[:160]
print('RAT170_UNAUTH_ROUTE='+json.dumps({'path':path,'status':int(code),'message':detail}))
PY
  rm -f "$tmp"
done
echo 'RAT170_CANONICAL_ROUTE_REGISTERED_BEGIN'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php artisan route:list --path='api/v1/rate-anything/things' --json | python3 -c '
import json,sys
data=json.load(sys.stdin)
for x in data:
    print("RAT170_ROUTE="+json.dumps({"method":x.get("method"),"uri":x.get("uri"),"domain":x.get("domain"),"action":x.get("action")},sort_keys=True))
'
echo 'RAT170_CANONICAL_ROUTE_REGISTERED_END'
echo 'RAT170_LIVE_RAT_NGINX_EDIT_LOCATION_BEGIN'
kctl -n rum-rate-anything-preview exec deployment/rum-rate-anything -c rate-anything -- sh -c "nginx -T 2>&1 | grep -A 12 'rate-anything/things/' | head -16" || true
echo 'RAT170_LIVE_RAT_NGINX_EDIT_LOCATION_END'
echo 'RAT170_ROUTE_DIAGNOSTIC_READ_ONLY=true'

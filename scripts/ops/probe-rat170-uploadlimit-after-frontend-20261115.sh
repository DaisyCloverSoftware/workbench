#!/usr/bin/env bash
set -Eeuo pipefail
# No credentials, no user-record mutation. Confirm 2 MiB reaches API auth gate.
[[ $# -eq 2 ]] || exit 64
source="$1"; candidate="$2"
[[ "$source" =~ ^[0-9a-f]{40}$ && "$candidate" =~ ^ghcr[.]io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api_source='17cef49b962943095df5374f65de7c74cda640dd'
api_digest='ghcr.io/daisycloversoftware/rum-api@sha256:0fc1525d9184bdb140bb543baae817fc54bf3237629ceec38429fd3e2301f521'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api_digest" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$candidate" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
config="$(kctl -n rum-rate-anything-preview exec deployment/rum-rate-anything -c rate-anything -- nginx -T 2>&1)"
grep -q 'client_max_body_size 16m' <<<"$config" || exit 78
echo 'RAT170_UPLOADLIMIT_RUNTIME_16M_PASS=true'
rat_version="$(curl -fsS --retry 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')"
[[ "$rat_version" == "rat-candidate+$source" ]] || exit 78
api_health="$(curl -fsS --retry 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$api_health" "$api_source" <<'PY'
import sys,json
j=json.loads(sys.argv[1])
assert j.get('status')=='ok' and j.get('service')=='rum-api' and sys.argv[2] in str(j.get('version',''))
PY
echo 'RAT170_UPLOADLIMIT_API_SOURCE_STILL_UNCHANGED=true'
url='https://dev-rum-ra.daisycloversoftware.uk/api/v1/rate-anything/things/01m7fvzcen292rvqeqj03evtxe/images/upload'
for kb in 512 2048; do
 f="$(mktemp)"
 trap 'rm -f "$f"' EXIT
 truncate -s "$kb"K "$f"
 code="$(curl -sS --connect-timeout 5 --max-time 45 --output /dev/null --write-out '%{http_code}' \
    -H 'Accept: application/json' -F 'slot=profile' -F "file=@$f;filename=probe.png;type=image/png" "$url")" || {
    echo "RAT170_UPLOADLIMIT_TRANSPORT_ERROR_KiB=$kb";exit 2;
  }
 echo "RAT170_UPLOADLIMIT_PROBE_KiB=$kb HTTP=$code"
 [[ "$code" =~ ^[0-9]{3}$ && "$code" != 413 && "$code" != 200 && "$code" != 201 && "$code" != 000 ]] || {
   echo 'Unexpected size rejection or unauthenticated write' >&2;exit 2;
 }
 rm -f "$f";trap - EXIT
done
echo 'RAT170_UPLOADLIMIT_2MiB_PROXY_NO_LONGER_REJECTS=true'
echo 'RAT170_UPLOADLIMIT_NO_DATA_MUTATION=true'

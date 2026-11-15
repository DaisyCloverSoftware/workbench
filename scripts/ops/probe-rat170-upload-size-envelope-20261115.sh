#!/usr/bin/env bash
set -Eeuo pipefail
# No authentication or mutation authority; only a disposable synthetic Thing.
# Probe the preview proxy size boundary without modifying user/owner records.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
target='https://dev-rum-ra.daisycloversoftware.uk/api/v1/rate-anything/things/01m7fvzcen292rvqeqj03evtxe/images/upload'
for kb in 2048 13312; do
 file="$(mktemp)"
 trap 'rm -f "$file"' EXIT
 truncate -s "$kb"K "$file"
 code="$(curl -sS --connect-timeout 5 --max-time 30 --output /dev/null --write-out '%{http_code}' \
  -H 'Accept: application/json' -F 'slot=profile' -F "file=@$file;filename=invalid-image.png;type=image/png" "$target")" || {
    echo "RAT170_UPLOAD_SIZE_PROBE_$kb-KiB=HTTP_ERROR"; exit 2;
  }
 [[ "$code" =~ ^[0-9]{3}$ ]] || exit 2
 echo "RAT170_UPLOAD_SIZE_PROBE_$kb-KiB_HTTP=$code"
 [[ "$code" != 200 && "$code" != 201 ]] || { echo 'Unexpected unauthenticated upload accepted' >&2; exit 2; }
 rm -f "$file"
 trap - EXIT
done
echo RAT170_UPLOAD_SIZE_PROXY_DIAGNOSTIC_COMPLETE=true

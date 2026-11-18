#!/usr/bin/env bash
set -Eeuo pipefail
# Pure read-only exact workload and public version check before/after PR177 frontend rollout.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
namespace='rum-rate-anything-preview'
rat="$(kctl -n "$namespace" get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')"
api="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
ratready="$(kctl -n "$namespace" get deployment rum-rate-anything -o 'jsonpath={.status.readyReplicas}')"
apiready="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.status.readyReplicas}')"
live_host="$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\\n"}{end}' | sort -u)"
rat_host="$(kctl -n "$namespace" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\\n"}{end}' | sort -u)"
dev_host="$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\\n"}{end}' | sort -u)"
version="$(curl -fsS --retry 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\\r\\n')"
health="$(curl -fsS --retry 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
[[ "$rat_host" == 'dev-rum-ra.daisycloversoftware.uk' && "$dev_host" == 'dev-rum.daisycloversoftware.uk' && "$live_host" == 'rateurmate.online' ]] || exit 78
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c' ]] || exit 78
case "$rat" in
  'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:6a506b08359eea423340c467ee6f330e22556c1a29002d0351323528123c8518'|'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003') ;;
 *) echo 'Unexpected RAT digest'; exit 78;;
esac
[[ "$ratready" == 1 && "$apiready" =~ ^[1-9] ]] || exit 78
printf 'RAT177_RUNTIME_STATE rat_image=%s api_image=%s rat_ready=%s api_ready=%s version=%s\\n' "$rat" "$api" "$ratready" "$apiready" "$version"
python3 - "$health" <<'PY'
import json,sys
h=json.loads(sys.argv[1])
assert h.get('status')=='ok' and h.get('service')=='rum-api'
assert '628296e5c57665814799bfedf1ab3ef8f6f163df' in str(h.get('version',''))
print('RAT177_CANONICAL_API_HEALTH_PASS=true')
PY
echo RAT177_ISOLATED_RUNTIME_READONLY_PASS=true

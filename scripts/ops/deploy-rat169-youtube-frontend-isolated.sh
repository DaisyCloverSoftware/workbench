#!/usr/bin/env bash
set -euo pipefail

# PR169 YouTube guidance: exactly one isolated RAT frontend image promotion. Canonical API must match
# exact source; no Helm, datastore, fixture, worker or LIVE operations.
[[ $# -eq 3 ]] || {
  echo 'usage: deploy-rat-thingtype-frontend-isolated.sh <source-sha> <rat-image@sha256> <canonical-dev-api@sha256>' >&2
  exit 64
}
source_sha="$1"
candidate="$2"
canonical_dev_api="$3"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$candidate" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$canonical_dev_api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64

namespace='rum-rate-anything-preview'
host='dev-rum-ra.daisycloversoftware.uk'
baseline='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9e9feff67611009f1f9193345754fd929cd9dd1449e6d3b30ae3a5486c085638'
kctl() {
  if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi
}
image() {
  kctl -n "$namespace" get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}'
}
for ns in rum-rate-anything-preview rum-dev-isolated rum-dev; do
  kctl get namespace "$ns" >/dev/null
done
rat_hosts="$(kctl -n "$namespace" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
dev_hosts="$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
live_hosts="$(kctl -n rum-dev get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$rat_hosts" == "$host" ]] || { echo 'Blocked: isolated RAT host mismatch' >&2; exit 78; }
[[ "$dev_hosts" == "dev-rum.daisycloversoftware.uk" ]] || { echo 'Blocked: canonical DEV host mismatch' >&2; exit 78; }
[[ "$live_hosts" == "rateurmate.online" ]] || { echo 'Blocked: LIVE host mismatch' >&2; exit 78; }

before="$(image)"
[[ "$before" == "$baseline" || "$before" == "$candidate" ]] || {
  echo "Blocked: isolated RAT frontend changed since read-only checkpoint: $before" >&2; exit 78;
}
deployed_api="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$deployed_api" == "$canonical_dev_api" ]] || { echo 'Blocked: canonical DEV API candidate mismatch' >&2; exit 78; }

health="$(curl -fsS --retry 4 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$source_sha" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get('status') == 'ok' and obj.get('service') == 'rum-api'
assert sys.argv[2] in str(obj.get('version','')), obj
PY

# Hash every object/spec other than this one RAT frontend Deployment to prove
# the migration, API, other workloads, configurations and LIVE are unchanged
# by this specific frontend rollout.
protected_snapshot() {
  for ns in rum-rate-anything-preview rum-dev-isolated rum-dev; do
    kctl -n "$ns" get deployment,statefulset,cronjob,configmap,persistentvolumeclaim,ingress -o json | python3 -c '
import json,sys
items=json.load(sys.stdin)["items"]
rows=[]
for item in items:
    metadata=item["metadata"]
    if metadata["namespace"] == "rum-rate-anything-preview" and item["kind"] == "Deployment" and metadata["name"] == "rum-rate-anything":
        continue
    rows.append({"namespace":metadata["namespace"],"kind":item["kind"],"name":metadata["name"],"spec":item.get("spec"),"data":item.get("data")})
print(json.dumps(sorted(rows,key=lambda x:(x["kind"],x["name"])),sort_keys=True,separators=(",",":")))' 
  done | sha256sum | cut -d' ' -f1
}
protected_before="$(protected_snapshot)"
changed=false
rollback() {
  local code=$?
  trap - ERR
  if [[ "$changed" == true ]]; then
    kctl -n "$namespace" set image deployment/rum-rate-anything "rate-anything=$before" >/dev/null || true
    kctl -n "$namespace" rollout status deployment/rum-rate-anything --timeout=240s || true
  fi
  echo 'RAT_THINGTYPE_FRONTEND_ROLLOUT_FAILED; reverted RAT frontend only' >&2
  exit "$code"
}
trap rollback ERR

if [[ "$before" != "$candidate" ]]; then
  changed=true
  kctl -n "$namespace" set image deployment/rum-rate-anything "rate-anything=$candidate" >/dev/null
fi
kctl -n "$namespace" rollout status deployment/rum-rate-anything --timeout=300s
[[ "$(image)" == "$candidate" ]]
version="$(curl -fsS --retry 6 --retry-delay 2 --max-time 20 -H 'Cache-Control: no-cache' "https://$host/VERSION" | tr -d '\r\n')"
[[ "$version" == "rat-candidate+$source_sha" ]]
protected_after="$(protected_snapshot)"
[[ "$protected_before" == "$protected_after" ]]
trap - ERR
printf 'RAT169_YOUTUBE_ISOLATED_FRONTEND_DEPLOYED\nsource_sha=%s\nrat_image=%s\ncanonical_dev_api=%s\nserved_version=%s\nprevious_frontend=%s\nprotected_before=%s\nprotected_after=%s\nno_reseed_or_reset=true\nlive_unchanged=true\n' "$source_sha" "$candidate" "$canonical_dev_api" "$version" "$before" "$protected_before" "$protected_after"

#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <40-char-source-sha> <api-image@sha256:...> <web-image@sha256:...> <rat-image@sha256:...>" >&2
  exit 64
fi

SOURCE_SHA="$1"
API_IMAGE="$2"
WEB_IMAGE="$3"
RAT_IMAGE="$4"

[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid source SHA" >&2; exit 64; }
[[ "$API_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable API image" >&2; exit 64; }
[[ "$WEB_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-web@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable web image" >&2; exit 64; }
[[ "$RAT_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable RAT image" >&2; exit 64; }

RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
PUBLIC_NS="rum-dev"
RUM_HOST="dev-rum.daisycloversoftware.uk"
RAT_HOST="dev-rum-ra.daisycloversoftware.uk"
PUBLIC_HOST="rateurmate.online"

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

for ns in "$RUM_NS" "$RAT_NS" "$PUBLIC_NS"; do
  kctl get namespace "$ns" >/dev/null
 done

host_set() {
  local ns="$1"
  kctl -n "$ns" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}

[[ "$(host_set "$RUM_NS")" == "$RUM_HOST" ]] || { echo "blocked: RUM isolated ingress host mismatch" >&2; exit 78; }
[[ "$(host_set "$RAT_NS")" == "$RAT_HOST" ]] || { echo "blocked: RAT isolated ingress host mismatch" >&2; exit 78; }
public_hosts_before="$(host_set "$PUBLIC_NS")"
grep -Fxq "$PUBLIC_HOST" <<<"$public_hosts_before" || { echo "blocked: public host not found in public namespace" >&2; exit 78; }

deployment_image() {
  local ns="$1" deployment="$2" container="$3"
  kctl -n "$ns" get deployment "$deployment" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}

public_web_before="$(deployment_image "$PUBLIC_NS" rum-web web)"
public_api_before="$(deployment_image "$PUBLIC_NS" rum-api php-fpm)"
public_worker_before="$(deployment_image "$PUBLIC_NS" rum-worker worker)"

rum_api_before="$(deployment_image "$RUM_NS" rum-api php-fpm)"
rum_web_before="$(deployment_image "$RUM_NS" rum-web web)"
rat_api_before="$(deployment_image "$RAT_NS" rum-api php-fpm)"
rat_web_before="$(deployment_image "$RAT_NS" rum-rate-anything rate-anything)"

restore_isolated() {
  local rc=$?
  trap - ERR
  echo "isolated candidate deployment failed; restoring prior isolated images" >&2
  kctl -n "$RUM_NS" set image deployment/rum-api "php-fpm=$rum_api_before" --record=false >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" set image deployment/rum-web "web=$rum_web_before" --record=false >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" set image deployment/rum-api "php-fpm=$rat_api_before" --record=false >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" set image deployment/rum-rate-anything "rate-anything=$rat_web_before" --record=false >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" rollout status deployment/rum-api --timeout=180s >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" rollout status deployment/rum-web --timeout=180s >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" rollout status deployment/rum-api --timeout=180s >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" rollout status deployment/rum-rate-anything --timeout=180s >/dev/null 2>&1 || true
  exit "$rc"
}
trap restore_isolated ERR

kctl -n "$RUM_NS" set image deployment/rum-api "php-fpm=$API_IMAGE" --record=false >/dev/null
kctl -n "$RUM_NS" set image deployment/rum-web "web=$WEB_IMAGE" --record=false >/dev/null
kctl -n "$RAT_NS" set image deployment/rum-api "php-fpm=$API_IMAGE" --record=false >/dev/null
kctl -n "$RAT_NS" set image deployment/rum-rate-anything "rate-anything=$RAT_IMAGE" --record=false >/dev/null

kctl -n "$RUM_NS" rollout status deployment/rum-api --timeout=300s
kctl -n "$RUM_NS" rollout status deployment/rum-web --timeout=300s
kctl -n "$RAT_NS" rollout status deployment/rum-api --timeout=300s
kctl -n "$RAT_NS" rollout status deployment/rum-rate-anything --timeout=300s

[[ "$(deployment_image "$RUM_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image "$RUM_NS" rum-web web)" == "$WEB_IMAGE" ]]
[[ "$(deployment_image "$RAT_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image "$RAT_NS" rum-rate-anything rate-anything)" == "$RAT_IMAGE" ]]

[[ "$(host_set "$RUM_NS")" == "$RUM_HOST" ]]
[[ "$(host_set "$RAT_NS")" == "$RAT_HOST" ]]
[[ "$(host_set "$PUBLIC_NS")" == "$public_hosts_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-web web)" == "$public_web_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-api php-fpm)" == "$public_api_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-worker worker)" == "$public_worker_before" ]]

for origin in "https://$RUM_HOST" "https://$RAT_HOST"; do
  version="$(curl -fsS -H 'Cache-Control: no-cache' --retry 6 --retry-delay 2 --max-time 20 "$origin/VERSION" | tr -d '\r\n')"
  grep -Fq "$SOURCE_SHA" <<<"$version" || { echo "exact-head VERSION mismatch at $origin: $version" >&2; false; }
done

catalogue="$(curl -fsS -H 'Cache-Control: no-cache' --retry 6 --retry-delay 2 --max-time 20 "https://$RUM_HOST/api/v1/public/entities/rat-catalogue/search?q=CJ")"
python3 - "$catalogue" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get('meta',{}).get('source') == 'rum-dev-canonical', obj
active=obj.get('data',{}).get('active') or {}
assert active.get('name') == 'CJ Investigates', obj
print('canonical_cj_id=' + str(active.get('id')))
PY

trap - ERR
printf 'RUM160_ISOLATED_DEPLOYED source_sha=%s\n' "$SOURCE_SHA"
printf 'rum_dev=%s\nrat_dev=%s\npublic_host_unchanged=%s\n' "$RUM_HOST" "$RAT_HOST" "$PUBLIC_HOST"
printf 'api_image=%s\nweb_image=%s\nrat_image=%s\n' "$API_IMAGE" "$WEB_IMAGE" "$RAT_IMAGE"

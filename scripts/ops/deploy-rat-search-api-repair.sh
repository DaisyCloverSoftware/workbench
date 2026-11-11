#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <40-char-source-sha> <api-image@sha256:...>" >&2
  exit 64
fi

SOURCE_SHA="$1"
API_IMAGE="$2"
DEV_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
LIVE_NS="rum-dev"
DEV_HOST="dev-rum.daisycloversoftware.uk"
RAT_HOST="dev-rum-ra.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"

[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid source SHA" >&2; exit 64; }
[[ "$API_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable API image" >&2; exit 64; }

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

image() {
  local ns="$1" deploy="$2" container="$3"
  kctl -n "$ns" get deployment "$deploy" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}

hosts() {
  local ns="$1"
  kctl -n "$ns" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}

for ns in "$DEV_NS" "$RAT_NS" "$LIVE_NS"; do
  kctl get namespace "$ns" >/dev/null
done

[[ "$(hosts "$DEV_NS")" == "$DEV_HOST" ]] || { echo "blocked: isolated DEV host mismatch" >&2; exit 78; }
[[ "$(hosts "$RAT_NS")" == "$RAT_HOST" ]] || { echo "blocked: RAT preview host mismatch" >&2; exit 78; }
live_hosts_before="$(hosts "$LIVE_NS")"
grep -Fxq "$LIVE_HOST" <<<"$live_hosts_before" || { echo "blocked: LIVE host missing" >&2; exit 78; }

dev_api_before="$(image "$DEV_NS" rum-api php-fpm)"
dev_web_before="$(image "$DEV_NS" rum-web web)"
dev_worker_before="$(image "$DEV_NS" rum-worker worker)"
rat_front_before="$(image "$RAT_NS" rum-rate-anything rate-anything)"
rat_api_before="$(image "$RAT_NS" rum-api php-fpm)"
live_api_before="$(image "$LIVE_NS" rum-api php-fpm)"
live_web_before="$(image "$LIVE_NS" rum-web web)"
live_worker_before="$(image "$LIVE_NS" rum-worker worker)"

rollback() {
  local rc=$?
  trap - ERR
  echo "RAT search API candidate verification failed; restoring prior isolated DEV API" >&2
  kctl -n "$DEV_NS" set image deployment/rum-api "php-fpm=$dev_api_before" --record=false >/dev/null 2>&1 || true
  kctl -n "$DEV_NS" rollout status deployment/rum-api --timeout=240s >/dev/null 2>&1 || true
  exit "$rc"
}
trap rollback ERR

kctl -n "$DEV_NS" set image deployment/rum-api "php-fpm=$API_IMAGE" --record=false >/dev/null
kctl -n "$DEV_NS" rollout status deployment/rum-api --timeout=300s

[[ "$(image "$DEV_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(image "$DEV_NS" rum-web web)" == "$dev_web_before" ]]
[[ "$(image "$DEV_NS" rum-worker worker)" == "$dev_worker_before" ]]
[[ "$(image "$RAT_NS" rum-rate-anything rate-anything)" == "$rat_front_before" ]]
[[ "$(image "$RAT_NS" rum-api php-fpm)" == "$rat_api_before" ]]
[[ "$(image "$LIVE_NS" rum-api php-fpm)" == "$live_api_before" ]]
[[ "$(image "$LIVE_NS" rum-web web)" == "$live_web_before" ]]
[[ "$(image "$LIVE_NS" rum-worker worker)" == "$live_worker_before" ]]
[[ "$(hosts "$LIVE_NS")" == "$live_hosts_before" ]]

health="$(curl -fsS --retry 6 --retry-delay 2 --max-time 20 "https://$DEV_HOST/api/v1/health")"
python3 - "$health" "$SOURCE_SHA" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get("status") == "ok", obj
assert obj.get("service") == "rum-api", obj
assert sys.argv[2] in str(obj.get("version","")), obj
PY

verify_search() {
  local query="$1" expected="${2:-}"
  local headers body
  headers="$(mktemp)"
  body="$(mktemp)"
  curl -fsS --retry 6 --retry-delay 2 --max-time 20 \
    -D "$headers" \
    -H "Origin: https://$RAT_HOST" \
    -H 'Accept: application/json' \
    "https://$DEV_HOST/api/v1/public/entities/rat-catalogue/search?q=$query" \
    -o "$body"
  tr -d '\r' <"$headers" | grep -Fiqx "Access-Control-Allow-Origin: https://$RAT_HOST"
  python3 - "$body" "$query" "$expected" <<'PY'
import json,sys
with open(sys.argv[1], encoding="utf-8") as fh:
    obj=json.load(fh)
meta=obj.get("meta",{})
assert meta.get("source") == "rum-dev-canonical", obj
assert meta.get("privacy") == "private-rum-people-excluded", obj
assert isinstance(meta.get("found"), bool), obj
found=meta["found"]
data=obj.get("data")
if found:
    active=(data or {}).get("active") or {}
    assert active.get("id"), obj
    if sys.argv[3]:
        assert active.get("name") == sys.argv[3], obj
    print("verified_search=" + sys.argv[2] + " found=true name=" + str(active.get("name")))
else:
    assert data is None, obj
    assert not sys.argv[3], obj
    print("verified_search=" + sys.argv[2] + " found=false")
PY
  rm -f "$headers" "$body"
}

verify_search DJI
verify_search CJ "CJ Investigates"

trap - ERR
echo "RAT_SEARCH_API_REPAIR_DEPLOYED source_sha=$SOURCE_SHA"
echo "api_image=$API_IMAGE"
echo "dev_api_before=$dev_api_before"
echo "rat_frontend_preserved=$rat_front_before"
echo "rat_preview_api_preserved=$rat_api_before"
echo "live_unchanged=true"

#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <40-char-source-sha> <api-image@sha256:...>" >&2
  exit 64
fi

SOURCE_SHA="$1"
API_IMAGE="$2"
DEV_NS="rum-dev-isolated"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_NS="rum-dev"
LIVE_HOST="rateurmate.online"
RAT_HOST="dev-rum-ra.daisycloversoftware.uk"
CONFIGMAP="rum-config"
API_DEPLOYMENT="rum-api"

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

kctl get namespace "$DEV_NS" >/dev/null
kctl get namespace "$LIVE_NS" >/dev/null
[[ "$(hosts "$DEV_NS")" == "$DEV_HOST" ]] || { echo "blocked: isolated DEV host mismatch" >&2; exit 78; }
live_hosts_before="$(hosts "$LIVE_NS")"
grep -Fxq "$LIVE_HOST" <<<"$live_hosts_before" || { echo "blocked: LIVE host missing" >&2; exit 78; }

dev_api_before="$(image "$DEV_NS" "$API_DEPLOYMENT" php-fpm)"
live_api_before="$(image "$LIVE_NS" rum-api php-fpm)"
live_web_before="$(image "$LIVE_NS" rum-web web)"
live_worker_before="$(image "$LIVE_NS" rum-worker worker)"
[[ "$dev_api_before" == "$API_IMAGE" ]] || { echo "blocked: isolated DEV API image mismatch" >&2; exit 79; }

app_url_before="$(kctl -n "$DEV_NS" get configmap "$CONFIGMAP" -o jsonpath='{.data.APP_URL}')"
[[ "$app_url_before" == "https://$DEV_HOST" ]] || { echo "blocked: isolated DEV APP_URL mismatch" >&2; exit 79; }

guard_before="$(kctl -n "$DEV_NS" get configmap "$CONFIGMAP" -o jsonpath='{.data.RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED}' 2>/dev/null || true)"
if [[ -n "$guard_before" && "$guard_before" != "true" ]]; then
  echo "blocked: existing canonical DEV fixture guard has unexpected value" >&2
  exit 79
fi

rollback() {
  local rc=$?
  trap - ERR
  if [[ -z "$guard_before" ]]; then
    kctl -n "$DEV_NS" patch configmap "$CONFIGMAP" --type=json -p='[{"op":"remove","path":"/data/RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED"}]' >/dev/null 2>&1 || true
  fi
  kctl -n "$DEV_NS" rollout restart deployment/"$API_DEPLOYMENT" >/dev/null 2>&1 || true
  kctl -n "$DEV_NS" rollout status deployment/"$API_DEPLOYMENT" --timeout=240s >/dev/null 2>&1 || true
  echo "RAT DJI canonical DEV repair failed; restored prior guard state" >&2
  exit "$rc"
}
trap rollback ERR

if [[ "$guard_before" != "true" ]]; then
  kctl -n "$DEV_NS" patch configmap "$CONFIGMAP" --type=merge -p='{"data":{"RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED":"true"}}' >/dev/null
  kctl -n "$DEV_NS" rollout restart deployment/"$API_DEPLOYMENT" >/dev/null
  kctl -n "$DEV_NS" rollout status deployment/"$API_DEPLOYMENT" --timeout=300s
fi

runtime_app_url="$(kctl -n "$DEV_NS" exec deployment/"$API_DEPLOYMENT" -c php-fpm -- printenv APP_URL | tr -d '\r\n')"
runtime_guard="$(kctl -n "$DEV_NS" exec deployment/"$API_DEPLOYMENT" -c php-fpm -- printenv RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED | tr -d '\r\n')"
[[ "$runtime_app_url" == "https://$DEV_HOST" ]] || { echo "blocked: runtime APP_URL mismatch" >&2; exit 80; }
[[ "$runtime_guard" == "true" ]] || { echo "blocked: runtime fixture seed guard not true" >&2; exit 80; }

kctl -n "$DEV_NS" exec deployment/"$API_DEPLOYMENT" -c php-fpm -- php artisan db:seed --class='Database\Seeders\RateAnythingCanonicalDevSeeder' --force

verify_search() {
  local query="$1" expected="$2"
  local body
  body="$(curl -fsS --retry 6 --retry-delay 2 --max-time 20 -H "Origin: https://$RAT_HOST" -H 'Accept: application/json' "https://$DEV_HOST/api/v1/public/entities/rat-catalogue/search?q=$query")"
  python3 - "$body" "$expected" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
expected=sys.argv[2]
meta=obj.get("meta",{})
assert meta.get("source") == "rum-dev-canonical", obj
assert meta.get("privacy") == "private-rum-people-excluded", obj
if expected:
    assert meta.get("found") is True, obj
    active=(obj.get("data") or {}).get("active") or {}
    assert active.get("name") == expected, obj
    print("verified="+expected+" id="+str(active.get("id")))
else:
    assert meta.get("found") is False, obj
    assert obj.get("data") is None, obj
    print("verified=no-match")
PY
}

verify_search DJI DJI
verify_search CJ "CJ Investigates"
verify_search owner-no-match-dji-recovery-7cc91 ""

[[ "$(image "$LIVE_NS" rum-api php-fpm)" == "$live_api_before" ]]
[[ "$(image "$LIVE_NS" rum-web web)" == "$live_web_before" ]]
[[ "$(image "$LIVE_NS" rum-worker worker)" == "$live_worker_before" ]]
[[ "$(hosts "$LIVE_NS")" == "$live_hosts_before" ]]

trap - ERR
echo "RAT_DJI_CANONICAL_DEV_REPAIRED source_sha=$SOURCE_SHA"
echo "api_image=$API_IMAGE"
echo "guard_after=true"
echo "live_unchanged=true"

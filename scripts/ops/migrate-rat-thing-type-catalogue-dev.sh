#!/usr/bin/env bash
set -euo pipefail

# Additive canonical RUM DEV classification-mapping migration only.
# Never invokes a seed/reset, Helm, worker, LIVE API, preview API or frontend.
if [[ $# -ne 2 ]]; then
  echo 'usage: migrate-rat-thing-type-catalogue-dev.sh <exact-source-sha> <api-image@sha256:...>' >&2
  exit 64
fi
SOURCE_SHA="$1"
API_IMAGE="$2"
DEV_NS=rum-dev-isolated
RAT_NS=rum-rate-anything-preview
LIVE_NS=rum-dev
DEV_HOST=dev-rum.daisycloversoftware.uk
RAT_HOST=dev-rum-ra.daisycloversoftware.uk
LIVE_HOST=rateurmate.online
MIGRATION=2026_11_13_000001_map_thing_classifications_to_entity_types

[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid source SHA' >&2; exit 64; }
[[ "$API_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || { echo 'Invalid API digest' >&2; exit 64; }

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
  kctl -n "$1" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}

for ns in "$DEV_NS" "$RAT_NS" "$LIVE_NS"; do
  kctl get namespace "$ns" >/dev/null
done
[[ "$(hosts "$DEV_NS")" == "$DEV_HOST" ]] || { echo 'Blocked: canonical DEV ingress mismatch' >&2; exit 78; }
[[ "$(hosts "$RAT_NS")" == "$RAT_HOST" ]] || { echo 'Blocked: isolated RAT ingress mismatch' >&2; exit 78; }
live_hosts="$(hosts "$LIVE_NS")"
[[ "$live_hosts" == "$LIVE_HOST" ]] || { echo 'Blocked: LIVE ingress mismatch' >&2; exit 78; }

dev_api_before="$(image "$DEV_NS" rum-api php-fpm)"
rat_front_before="$(image "$RAT_NS" rum-rate-anything rate-anything)"
rat_api_before="$(image "$RAT_NS" rum-api php-fpm)"
live_api_before="$(image "$LIVE_NS" rum-api php-fpm)"
live_web_before="$(image "$LIVE_NS" rum-web web)"
live_worker_before="$(image "$LIVE_NS" rum-worker worker)"
dev_web_before="$(image "$DEV_NS" rum-web web)"
dev_worker_before="$(image "$DEV_NS" rum-worker worker)"
[[ "$dev_api_before" == "$API_IMAGE" ]] || { echo 'Blocked: candidate API not deployed in canonical DEV' >&2; exit 78; }
app_url="$(kctl -n "$DEV_NS" get configmap rum-config -o jsonpath='{.data.APP_URL}')"
[[ "$app_url" == "https://$DEV_HOST" ]] || { echo 'Blocked: DEV APP_URL mismatch' >&2; exit 78; }
runtime_url="$(kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm -- printenv APP_URL | tr -d '\r\n')"
[[ "$runtime_url" == "https://$DEV_HOST" ]] || { echo 'Blocked: runtime DEV URL mismatch' >&2; exit 78; }

health="$(curl -fsS --retry 4 --retry-delay 2 --max-time 20 "https://$DEV_HOST/api/v1/health")"
python3 - "$health" "$SOURCE_SHA" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get('status')=='ok', obj
assert sys.argv[2] in str(obj.get('version','')), obj
PY

# Only the named, reversible, nullable-column migration. No general migrate
# invocation, data fixtures, arbitrary SQL, or historical schema updates.
kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm --   php artisan migrate --force --no-interaction     --path="database/migrations/$MIGRATION.php"

status="$(kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm --   php artisan migrate:status --path="database/migrations/$MIGRATION.php" --no-ansi)"
printf '%s\n' "$status" | grep -F "$MIGRATION" | grep -Eq 'Ran|Yes|\[[0-9]+\]'
[[ "$(image "$DEV_NS" rum-api php-fpm)" == "$dev_api_before" ]]
[[ "$(image "$DEV_NS" rum-web web)" == "$dev_web_before" ]]
[[ "$(image "$DEV_NS" rum-worker worker)" == "$dev_worker_before" ]]
[[ "$(image "$RAT_NS" rum-rate-anything rate-anything)" == "$rat_front_before" ]]
[[ "$(image "$RAT_NS" rum-api php-fpm)" == "$rat_api_before" ]]
[[ "$(image "$LIVE_NS" rum-api php-fpm)" == "$live_api_before" ]]
[[ "$(image "$LIVE_NS" rum-web web)" == "$live_web_before" ]]
[[ "$(image "$LIVE_NS" rum-worker worker)" == "$live_worker_before" ]]
[[ "$(hosts "$LIVE_NS")" == "$live_hosts" ]]

echo "RAT_THING_TYPE_CANONICAL_DEV_MIGRATION_OK source_sha=$SOURCE_SHA"
echo "migration=$MIGRATION"
echo "canonical_dev_api=$dev_api_before"
echo 'no_fixture_seed=true'
echo 'no_datastore_reset=true'
echo 'rat_preview_api_frontend_unchanged=true'
echo 'live_unchanged=true'

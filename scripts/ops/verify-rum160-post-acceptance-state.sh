#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 0 ]]; then
  echo "usage: $0" >&2
  exit 64
fi

SOURCE_SHA="226f898fa0508b9f5b41b8d0651bab315b7dd204"
API_IMAGE="ghcr.io/daisycloversoftware/rum-api@sha256:8daee721ada11f1ffcaf89a1c874319ed307ce51884153b8e7a2298b4a66936e"
WEB_IMAGE="ghcr.io/daisycloversoftware/rum-web@sha256:77f83cf631918cc72179cc8e57a7b12b1ff8884ef6efdde58306d7e3cabf401e"
RAT_IMAGE="ghcr.io/daisycloversoftware/rum-rate-anything@sha256:619130a9843cbd2c38d22e591019b3196c5960eea3583cdff01dd3524086dcbb"
PUBLIC_API="ghcr.io/daisycloversoftware/rum-api:sha-8106675"
PUBLIC_WEB="ghcr.io/daisycloversoftware/rum-web:sha-8106675"
SENTINEL_SLUG="rum160-private-acceptance-${SOURCE_SHA:0:12}"

kctl() {
  if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi
}
deployment_image() {
  local ns="$1" deployment="$2" container="$3"
  kctl -n "$ns" get deployment "$deployment" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}

[[ "$(deployment_image rum-dev-isolated rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image rum-dev-isolated rum-web web)" == "$WEB_IMAGE" ]]
[[ "$(deployment_image rum-rate-anything-preview rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image rum-rate-anything-preview rum-rate-anything rate-anything)" == "$RAT_IMAGE" ]]

[[ "$(deployment_image rum-dev rum-api php-fpm)" == "$PUBLIC_API" ]]
[[ "$(deployment_image rum-dev rum-web web)" == "$PUBLIC_WEB" ]]
[[ "$(deployment_image rum-dev rum-worker worker)" == "$PUBLIC_API" ]]

sentinel_count="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- \
  env RUM160_SENTINEL_SLUG="$SENTINEL_SLUG" php -r '
require "vendor/autoload.php";
$app = require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
echo App\Models\Entity::query()->where("canonical_slug", getenv("RUM160_SENTINEL_SLUG"))->count();
' | tr -d '\r\n')"
[[ "$sentinel_count" == "0" ]] || { echo "temporary privacy sentinel was not cleaned up" >&2; exit 10; }

rat_version="$(kctl -n rum-rate-anything-preview exec deployment/rum-rate-anything -c rate-anything -- cat /usr/share/nginx/html/VERSION | tr -d '\r\n')"
grep -Fq "$SOURCE_SHA" <<<"$rat_version"

printf 'RUM160_POST_ACCEPTANCE_STATE_VERIFIED source_sha=%s\n' "$SOURCE_SHA"
printf 'isolated_api=%s\nisolated_web=%s\nisolated_rat=%s\n' "$API_IMAGE" "$WEB_IMAGE" "$RAT_IMAGE"
printf 'privacy_sentinel_cleanup=true\npublic_images_unchanged=true\n'

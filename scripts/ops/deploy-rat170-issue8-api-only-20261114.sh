#!/usr/bin/env bash
set -euo pipefail
# Exact PR170 Issue8 canonical DEV API-only promotion; no Helm, migrations, seeds,
# RAT frontend, admin, worker, datastore, production or unrelated DEV updates.
[[ $# -eq 2 ]] || exit 64
sha="$1"; candidate="$2"
[[ "$sha" == '06da14c2ebf7be2fb28b83113ea6a4bf41b373e7' ]] || exit 78
[[ "$candidate" =~ ^ghcr[.]io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
current_expected='ghcr.io/daisycloversoftware/rum-api@sha256:e0f3de7f64cc0e54d7a0a812b47eccfbd9f2c8c97f623ac2aa72749665be9983'
rat_expected='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a7a8d6b7fb2dbefef30384b9b0d6658762a8ab74001617883660229755cf68c'
live_api_expected='ghcr.io/daisycloversoftware/rum-api:sha-54d31b8'
live_web_expected='ghcr.io/daisycloversoftware/rum-web:sha-8106675'
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
img(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
hosts(){ kctl -n "$1" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
[[ "$(hosts rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts rum-rate-anything-preview)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts rum-dev)" == 'rateurmate.online' ]] || exit 78
old="$(img rum-dev-isolated rum-api php-fpm)"
[[ "$old" == "$current_expected" || "$old" == "$candidate" ]] || { echo 'API deployment differs from baseline; refuse' >&2; exit 78; }
[[ "$(img rum-rate-anything-preview rum-rate-anything rate-anything)" == "$rat_expected" ]] || exit 78
[[ "$(img rum-dev rum-api php-fpm)" == "$live_api_expected" ]] || exit 78
[[ "$(img rum-dev rum-web web)" == "$live_web_expected" ]] || exit 78
[[ "$(img rum-dev rum-worker worker)" == "$live_api_expected" ]] || exit 78
bash "$(dirname "$0")/deploy-rat-search-api-repair.sh" "$sha" "$candidate"
[[ "$(img rum-dev-isolated rum-api php-fpm)" == "$candidate" ]] || exit 1
[[ "$(img rum-rate-anything-preview rum-rate-anything rate-anything)" == "$rat_expected" ]] || exit 1
[[ "$(img rum-dev rum-api php-fpm)" == "$live_api_expected" ]] || exit 1
[[ "$(img rum-dev rum-web web)" == "$live_web_expected" ]] || exit 1
[[ "$(img rum-dev rum-worker worker)" == "$live_api_expected" ]] || exit 1
echo 'RAT170_ISSUE8_API_ONLY_CONFIRMED'
echo "source_sha=$sha"
echo "api_image=$candidate"
echo "rat_frontend_preserved=$rat_expected"
echo 'live_unchanged=true'
echo 'no_migrations_seeds_or_reset=true'

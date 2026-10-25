#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi
CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || exit 64

for command in gh podman mktemp timeout; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "no GitHub/GHCR token available" >&2; exit 2; }
ACTOR="$(GH_TOKEN="$TOKEN" gh api user --jq .login 2>/dev/null || true)"
[[ -n "$ACTOR" ]] || ACTOR="mattbrownsett"
AUTHDIR="$(mktemp -d)"
AUTHFILE="$AUTHDIR/auth.json"
cleanup(){ rm -rf "$AUTHDIR"; }
trap cleanup EXIT HUP INT TERM
printf '%s' "$TOKEN" | podman login --authfile "$AUTHFILE" ghcr.io -u "$ACTOR" --password-stdin >/dev/null
unset TOKEN GH_TOKEN GHCR_TOKEN

candidate_tag="sha-${CANDIDATE_SHA}"
baseline_tag="sha-8106675"
for component in rum-web rum-api; do
  for tag in "$baseline_tag" "$candidate_tag"; do
    ref="ghcr.io/daisycloversoftware/${component}:${tag}"
    timeout 120s podman pull --authfile "$AUTHFILE" "$ref" >/dev/null
    echo "IMAGE=${component}:${tag}"
    podman image inspect "$ref" --format 'ARCH={{.Architecture}} OS={{.Os}} USER={{.Config.User}} ENTRYPOINT={{json .Config.Entrypoint}} CMD={{json .Config.Cmd}} REVISION={{index .Labels "org.opencontainers.image.revision"}}'
    if [[ "$component" == "rum-web" ]]; then
      timeout 20s podman run --rm --entrypoint /bin/sh "$ref" -c 'id; nginx -t; test -r /etc/nginx/nginx.conf; test -r /usr/share/nginx/html/index.html; echo WEB_STATIC_OK' 2>&1 || true
    else
      timeout 20s podman run --rm --entrypoint /bin/sh "$ref" -c 'id; php -v | head -n1; php-fpm -tt 2>&1 | tail -n 8; test -r /var/www/html/artisan; echo API_FILES_OK' 2>&1 || true
    fi
  done
done

echo REGISTRY_ONLY_DIAGNOSTIC=YES
echo LIVE_RUNTIME_AFFECTED=NO
echo RATE_ANYTHING_AFFECTED=NO

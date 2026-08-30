#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-sha>" >&2
  exit 64
fi
SHA="$1"
[[ "$SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
API_IMAGE="ghcr.io/daisycloversoftware/rum-api@sha256:f4e700314dd2fe2b8b6dabee5f640f35b9463d914f8a2be79ff781862a80e978"
POSTGRES_IMAGE="docker.io/library/postgres:18"
VALKEY_IMAGE="docker.io/valkey/valkey:8-alpine"

for command in gh git mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }
if command -v podman >/dev/null 2>&1; then RUNTIME=podman; elif command -v docker >/dev/null 2>&1; then RUNTIME=docker; else echo "no container runtime" >&2; exit 2; fi

tmp="$(mktemp -d)"
suffix="${SHA:0:8}-$$"
network="rum-api-check-$suffix"
pg="rum-api-pg-$suffix"
valkey="rum-api-valkey-$suffix"
api="rum-api-app-$suffix"
cleanup() {
  "$RUNTIME" rm -f "$api" "$pg" "$valkey" >/dev/null 2>&1 || true
  "$RUNTIME" network rm "$network" >/dev/null 2>&1 || true
  rm -rf "$tmp"
}
trap cleanup EXIT HUP INT TERM

GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$SHA" >/dev/null
[[ "$(git -C "$tmp/rum" rev-parse HEAD)" == "$SHA" ]] || { echo "checkout mismatch" >&2; exit 78; }

"$RUNTIME" pull "$API_IMAGE" >/dev/null
"$RUNTIME" pull "$POSTGRES_IMAGE" >/dev/null
"$RUNTIME" pull "$VALKEY_IMAGE" >/dev/null
"$RUNTIME" network create "$network" >/dev/null
"$RUNTIME" run -d --name "$pg" --network "$network" \
  -e POSTGRES_USER=rum -e POSTGRES_PASSWORD=rum -e POSTGRES_DB=rum_test \
  "$POSTGRES_IMAGE" >/dev/null
"$RUNTIME" run -d --name "$valkey" --network "$network" "$VALKEY_IMAGE" >/dev/null

ready=0
for _ in $(seq 1 40); do
  if "$RUNTIME" exec "$pg" pg_isready -U rum -d rum_test >/dev/null 2>&1; then ready=1; break; fi
  sleep 1
done
[[ "$ready" == 1 ]] || { echo "Postgres test service did not become ready" >&2; exit 70; }

"$RUNTIME" create --name "$api" --user 0:0 --network "$network" --entrypoint tail "$API_IMAGE" -f /dev/null >/dev/null
"$RUNTIME" start "$api" >/dev/null
"$RUNTIME" exec --user 0:0 "$api" mkdir -p /repo
# Preserve the repository-relative layout because API contract tests intentionally
# read Helm and runtime-contract files outside apps/api.
"$RUNTIME" cp "$tmp/rum/." "$api:/repo"

API_WORKDIR=/repo/apps/api
"$RUNTIME" exec --user 0:0 -w "$API_WORKDIR" "$api" sh -lc 'composer install --no-interaction --prefer-dist --no-progress >/dev/null'
"$RUNTIME" exec --user 0:0 -w "$API_WORKDIR" "$api" vendor/bin/pint --test
"$RUNTIME" exec --user 0:0 -w "$API_WORKDIR" "$api" vendor/bin/phpstan analyse --memory-limit=1G --no-progress
"$RUNTIME" exec --user 0:0 -w "$API_WORKDIR" \
  -e APP_ENV=testing \
  -e APP_KEY='base64:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=' \
  -e MODERATION_EVIDENCE_KEY='base64:ZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWVlZWU=' \
  -e MODERATION_AUDIT_IP_HASH_KEY='base64:YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=' \
  -e DB_CONNECTION=pgsql \
  -e DB_HOST="$pg" \
  -e DB_PORT=5432 \
  -e DB_DATABASE=rum_test \
  -e DB_USERNAME=rum \
  -e DB_PASSWORD=rum \
  -e REDIS_HOST="$valkey" \
  -e REDIS_PORT=6379 \
  -e CACHE_STORE=array \
  -e QUEUE_CONNECTION=sync \
  -e SESSION_DRIVER=array \
  -e MAIL_MAILER=array \
  "$api" vendor/bin/phpunit

printf 'RUM_API_EXACT_CI_CHECK_OK=%s\n' "$SHA"

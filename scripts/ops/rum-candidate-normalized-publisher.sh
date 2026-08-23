#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi
CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "candidate SHA must be 40 lowercase hex characters" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR="153"
TAG="sha-${CANDIDATE_SHA:0:8}"
WEB_IMAGE="ghcr.io/daisycloversoftware/rum-web:${TAG}"
API_IMAGE="ghcr.io/daisycloversoftware/rum-api:${TAG}"

for command in gh git podman mktemp timeout chmod; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done

umask 077
TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "no GitHub/GHCR token available" >&2; exit 2; }
branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || { echo "PUBLISH BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "PUBLISH BLOCKED: PR is not open/draft/unmerged exact head" >&2; exit 78; }
ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "CI" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || { echo "PUBLISH BLOCKED: exact-head CI success missing" >&2; exit 78; }

ACTOR="$(GH_TOKEN="$TOKEN" gh api user --jq .login 2>/dev/null || true)"
[[ -n "$ACTOR" ]] || ACTOR="mattbrownsett"
AUTHDIR="$(mktemp -d)"
AUTHFILE="$AUTHDIR/auth.json"
WORKDIR="$(mktemp -d)"
cleanup(){ rm -rf "$AUTHDIR" "$WORKDIR"; }
trap cleanup EXIT HUP INT TERM
printf '%s' "$TOKEN" | podman login --authfile "$AUTHFILE" ghcr.io -u "$ACTOR" --password-stdin >/dev/null

# Build context files must match normal Git checkout readability. Workbench's
# secret-protective umask is appropriate for auth/scratch roots but must not be
# inherited into Docker COPY source modes for non-root runtime containers.
umask 022
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$WORKDIR/rum" -- --no-checkout >/dev/null
git -C "$WORKDIR/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null
[[ "$(git -C "$WORKDIR/rum" rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || { echo "PUBLISH BLOCKED: checkout mismatch" >&2; exit 78; }
chmod -R u+rwX,go+rX "$WORKDIR/rum"
APP_VERSION="$(cat "$WORKDIR/rum/VERSION")+${CANDIDATE_SHA}"

# Stage external/unqualified images for Buildah short-name resolution.
podman pull docker.io/library/composer:2 >/dev/null
podman tag docker.io/library/composer:2 composer:2
podman pull docker.io/library/nginx:1.29.4-alpine3.23 >/dev/null
podman tag docker.io/library/nginx:1.29.4-alpine3.23 nginx:1.29.4-alpine3.23

podman build --pull=newer \
  --label "org.opencontainers.image.revision=${CANDIDATE_SHA}" \
  --build-arg VITE_DEMO_MODE=false \
  --build-arg VITE_ENTITY_BRIDGE=true \
  --build-arg VITE_UNIVERSAL_ENTITIES=false \
  --build-arg "APP_VERSION=${APP_VERSION}" \
  -t "$WEB_IMAGE" "$WORKDIR/rum/apps/web"

podman build --pull=newer \
  --label "org.opencontainers.image.revision=${CANDIDATE_SHA}" \
  --label "org.opencontainers.image.version=$(cat "$WORKDIR/rum/VERSION")" \
  --build-arg "APP_VERSION=${APP_VERSION}" \
  -t "$API_IMAGE" "$WORKDIR/rum/apps/api"

# Verify the exact non-root runtime users can read their copied configuration
# before anything is pushed.
timeout 30s podman run --rm --add-host rum-api:127.0.0.1 --entrypoint /bin/sh "$WEB_IMAGE" -c 'test -r /etc/nginx/nginx.conf; nginx -t; test -r /usr/share/nginx/html/index.html; echo WEB_RUNTIME_CONFIG_OK'
timeout 30s podman run --rm --entrypoint /bin/sh "$API_IMAGE" -c 'test -r /usr/local/etc/php-fpm.d/zz-rum.conf; php-fpm -tt >/dev/null; test -r /var/www/html/artisan; echo API_RUNTIME_CONFIG_OK'

podman push --authfile "$AUTHFILE" "$WEB_IMAGE" >/dev/null
podman push --authfile "$AUTHFILE" "$API_IMAGE" >/dev/null
podman pull --authfile "$AUTHFILE" "$WEB_IMAGE" >/dev/null
podman pull --authfile "$AUTHFILE" "$API_IMAGE" >/dev/null
web_revision="$(podman image inspect "$WEB_IMAGE" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
api_revision="$(podman image inspect "$API_IMAGE" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
web_digest="$(podman image inspect "$WEB_IMAGE" --format '{{.Digest}}')"
api_digest="$(podman image inspect "$API_IMAGE" --format '{{.Digest}}')"
[[ "$web_revision" == "$CANDIDATE_SHA" && "$api_revision" == "$CANDIDATE_SHA" ]] || { echo "PUBLISH BLOCKED: revision label mismatch" >&2; exit 78; }
[[ "$web_digest" == sha256:* && "$api_digest" == sha256:* ]] || { echo "PUBLISH BLOCKED: missing digest" >&2; exit 78; }

unset TOKEN GH_TOKEN GHCR_TOKEN
printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_DEPLOY_TAG=%s\n' "$TAG"
printf 'RUM_WEB_IMAGE=%s\n' "$WEB_IMAGE"
printf 'RUM_WEB_DIGEST=%s\n' "$web_digest"
printf 'RUM_API_IMAGE=%s\n' "$API_IMAGE"
printf 'RUM_API_DIGEST=%s\n' "$api_digest"
printf 'SOURCE_PERMISSIONS_NORMALIZED=YES\n'
printf 'RUNTIME_CONFIG_SMOKE=PASS\n'
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'

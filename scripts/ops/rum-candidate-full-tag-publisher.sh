#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || {
  echo "candidate SHA must be a full 40-character lowercase hexadecimal commit" >&2
  exit 64
}

REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR="153"
EXPECTED_CI_NAME="CI"
SHORT_TAG="sha-${CANDIDATE_SHA:0:7}"
FULL_TAG="sha-${CANDIDATE_SHA}"
SHORT_WEB="ghcr.io/daisycloversoftware/rum-web:${SHORT_TAG}"
SHORT_API="ghcr.io/daisycloversoftware/rum-api:${SHORT_TAG}"
FULL_WEB="ghcr.io/daisycloversoftware/rum-web:${FULL_TAG}"
FULL_API="ghcr.io/daisycloversoftware/rum-api:${FULL_TAG}"

for command in gh git podman mktemp timeout; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "required command unavailable: $command" >&2
    exit 2
  }
done

TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || {
  echo "no GitHub/GHCR token is available" >&2
  exit 2
}

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || {
  echo "PUBLISH BLOCKED: requested SHA is not the exact current RUM owner-candidate branch head." >&2
  exit 78
}

pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "PUBLISH BLOCKED: RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate." >&2
  exit 78
}

ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "'"${EXPECTED_CI_NAME}"'" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || {
  echo "PUBLISH BLOCKED: no successful exact-head ${EXPECTED_CI_NAME} workflow run exists for ${CANDIDATE_SHA}." >&2
  exit 78
}

main_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/main" --jq '.object.sha')"
[[ "$main_sha" != "$CANDIDATE_SHA" ]] || {
  echo "PUBLISH BLOCKED: candidate unexpectedly equals RUM main." >&2
  exit 78
}

ACTOR="$(GH_TOKEN="$TOKEN" gh api user --jq .login 2>/dev/null || true)"
[[ -n "$ACTOR" ]] || ACTOR="mattbrownsett"
AUTHDIR="$(mktemp -d)"
AUTHFILE="$AUTHDIR/auth.json"
WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$AUTHDIR" "$WORKDIR"
}
trap cleanup EXIT HUP INT TERM

printf '%s' "$TOKEN" | podman login --authfile "$AUTHFILE" ghcr.io -u "$ACTOR" --password-stdin >/dev/null

inspect_exact() {
  local ref="$1"
  if ! timeout 90s podman pull --authfile "$AUTHFILE" "$ref" >/dev/null 2>&1; then
    return 1
  fi
  local revision
  revision="$(podman image inspect "$ref" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
  [[ "$revision" == "$CANDIDATE_SHA" ]] || {
    echo "PUBLISH BLOCKED: ${ref} revision ${revision} != ${CANDIDATE_SHA}" >&2
    exit 78
  }
}

# Web is already published by the guarded short-tag publisher. Require that
# exact revision, then freeze it under the race-free full-source-SHA tag.
inspect_exact "$SHORT_WEB" || {
  echo "PUBLISH BLOCKED: exact short-tag RUM web image is not available." >&2
  exit 78
}
podman tag "$SHORT_WEB" "$FULL_WEB"
podman push --authfile "$AUTHFILE" "$FULL_WEB" >/dev/null

# If the exact short-tag API image has appeared, freeze it too. Otherwise build
# only API from the exact candidate source on this independent Podman host.
if inspect_exact "$SHORT_API"; then
  podman tag "$SHORT_API" "$FULL_API"
  podman push --authfile "$AUTHFILE" "$FULL_API" >/dev/null
  printf 'RUM_FULL_TAG_API_SOURCE=SHORT_TAG_REUSE\n'
else
  GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$WORKDIR/rum" -- --no-checkout >/dev/null
  git -C "$WORKDIR/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null
  [[ "$(git -C "$WORKDIR/rum" rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || {
    echo "PUBLISH BLOCKED: disposable checkout is not the exact candidate SHA." >&2
    exit 78
  }
  APP_VERSION="$(cat "$WORKDIR/rum/VERSION")+${CANDIDATE_SHA}"
  podman build \
    --pull=always \
    --label "org.opencontainers.image.revision=${CANDIDATE_SHA}" \
    --label "org.opencontainers.image.version=$(cat "$WORKDIR/rum/VERSION")" \
    --build-arg "APP_VERSION=${APP_VERSION}" \
    -t "$FULL_API" \
    "$WORKDIR/rum/apps/api"
  podman push --authfile "$AUTHFILE" "$FULL_API" >/dev/null
  printf 'RUM_FULL_TAG_API_SOURCE=EXACT_SOURCE_BUILD\n'
fi

# Re-pull both full tags from GHCR and verify the immutable final objects.
podman pull --authfile "$AUTHFILE" "$FULL_WEB" >/dev/null
podman pull --authfile "$AUTHFILE" "$FULL_API" >/dev/null
web_revision="$(podman image inspect "$FULL_WEB" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
api_revision="$(podman image inspect "$FULL_API" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
web_digest="$(podman image inspect "$FULL_WEB" --format '{{.Digest}}')"
api_digest="$(podman image inspect "$FULL_API" --format '{{.Digest}}')"
[[ "$web_revision" == "$CANDIDATE_SHA" && "$api_revision" == "$CANDIDATE_SHA" ]] || {
  echo "PUBLISH BLOCKED: full-tag image revision labels do not match exact candidate." >&2
  exit 78
}
[[ "$web_digest" == sha256:* && "$api_digest" == sha256:* ]] || {
  echo "PUBLISH BLOCKED: full-tag images have no immutable digests." >&2
  exit 78
}

unset TOKEN GH_TOKEN GHCR_TOKEN
printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_CANDIDATE_FULL_TAG=%s\n' "$FULL_TAG"
printf 'RUM_WEB_IMAGE=%s\n' "$FULL_WEB"
printf 'RUM_WEB_DIGEST=%s\n' "$web_digest"
printf 'RUM_API_IMAGE=%s\n' "$FULL_API"
printf 'RUM_API_DIGEST=%s\n' "$api_digest"
printf 'RATE_ANYTHING_AFFECTED=NO\n'
printf 'LIVE_RUNTIME_AFFECTED=NO\n'

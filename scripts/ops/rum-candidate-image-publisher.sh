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
TAG="sha-${CANDIDATE_SHA:0:7}"
WEB_IMAGE="ghcr.io/daisycloversoftware/rum-web:${TAG}"
API_IMAGE="ghcr.io/daisycloversoftware/rum-api:${TAG}"

for command in gh git docker mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command unavailable: $command" >&2
    exit 2
  }
done

TOKEN="${GHCR_TOKEN:-${GH_TOKEN:-}}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || {
  echo "ERROR: no GitHub/GHCR token is available" >&2
  exit 2
}

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || {
  echo "PUBLISH BLOCKED: requested SHA is not the exact current RUM owner-candidate branch head." >&2
  echo "expected=${branch_sha} requested=${CANDIDATE_SHA}" >&2
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

tmp_root="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_root"
}
trap cleanup EXIT HUP INT TERM

GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp_root/rum" -- --no-checkout --filter=blob:none >/dev/null
cd "$tmp_root/rum"
git checkout --detach "$CANDIDATE_SHA" >/dev/null
[[ "$(git rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || {
  echo "PUBLISH BLOCKED: disposable checkout is not the exact candidate SHA." >&2
  exit 78
}

APP_VERSION="$(cat VERSION)+${CANDIDATE_SHA}"
ACTOR="${GITHUB_ACTOR:-}"
if [[ -z "$ACTOR" ]]; then
  ACTOR="$(GH_TOKEN="$TOKEN" gh api user --jq .login 2>/dev/null || true)"
fi
[[ -n "$ACTOR" ]] || ACTOR="mattbrownsett"
printf '%s' "$TOKEN" | docker login ghcr.io -u "$ACTOR" --password-stdin >/dev/null

verify_existing_image() {
  local image="$1"
  if ! docker manifest inspect "$image" >/dev/null 2>&1; then
    return 1
  fi

  docker pull "$image" >/dev/null
  local revision digest
  revision="$(docker image inspect "$image" --format '{{index .Config.Labels "org.opencontainers.image.revision"}}')"
  digest="$(docker image inspect "$image" --format '{{index .RepoDigests 0}}')"
  [[ "$revision" == "$CANDIDATE_SHA" ]] || {
    echo "PUBLISH BLOCKED: existing ${image} has revision ${revision}, expected ${CANDIDATE_SHA}." >&2
    exit 78
  }
  [[ "$digest" == *@sha256:* ]] || {
    echo "PUBLISH BLOCKED: existing ${image} has no immutable repository digest." >&2
    exit 78
  }
  printf 'EXISTING_IMAGE=%s REVISION=%s\n' "$digest" "$revision"
  return 0
}

if ! verify_existing_image "$WEB_IMAGE"; then
  docker build \
    --pull \
    --label "org.opencontainers.image.revision=${CANDIDATE_SHA}" \
    --build-arg VITE_DEMO_MODE=false \
    --build-arg VITE_ENTITY_BRIDGE=true \
    --build-arg VITE_UNIVERSAL_ENTITIES=false \
    --build-arg "APP_VERSION=${APP_VERSION}" \
    -t "$WEB_IMAGE" \
    apps/web
  docker push "$WEB_IMAGE"
fi

if ! verify_existing_image "$API_IMAGE"; then
  docker build \
    --pull \
    --label "org.opencontainers.image.revision=${CANDIDATE_SHA}" \
    --label "org.opencontainers.image.version=$(cat VERSION)" \
    --build-arg "APP_VERSION=${APP_VERSION}" \
    -t "$API_IMAGE" \
    apps/api
  docker push "$API_IMAGE"
fi

docker pull "$WEB_IMAGE" >/dev/null
docker pull "$API_IMAGE" >/dev/null

web_revision="$(docker image inspect "$WEB_IMAGE" --format '{{index .Config.Labels "org.opencontainers.image.revision"}}')"
api_revision="$(docker image inspect "$API_IMAGE" --format '{{index .Config.Labels "org.opencontainers.image.revision"}}')"
web_digest="$(docker image inspect "$WEB_IMAGE" --format '{{index .RepoDigests 0}}')"
api_digest="$(docker image inspect "$API_IMAGE" --format '{{index .RepoDigests 0}}')"

[[ "$web_revision" == "$CANDIDATE_SHA" && "$api_revision" == "$CANDIDATE_SHA" ]] || {
  echo "PUBLISH BLOCKED: final image revision labels do not match exact RUM candidate." >&2
  exit 78
}
[[ "$web_digest" == *@sha256:* && "$api_digest" == *@sha256:* ]] || {
  echo "PUBLISH BLOCKED: final image digests are not immutable repository digests." >&2
  exit 78
}

printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_WEB_IMAGE=%s\n' "$WEB_IMAGE"
printf 'RUM_WEB_DIGEST=%s\n' "$web_digest"
printf 'RUM_API_IMAGE=%s\n' "$API_IMAGE"
printf 'RUM_API_DIGEST=%s\n' "$api_digest"
printf 'RUM_IMAGE_REVISION=%s\n' "$CANDIDATE_SHA"
printf 'RATE_ANYTHING_AFFECTED=NO\n'
printf 'LIVE_RUNTIME_AFFECTED=NO\n'

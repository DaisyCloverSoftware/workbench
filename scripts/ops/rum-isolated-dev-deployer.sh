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
TAG="sha-${CANDIDATE_SHA:0:8}"
DEPLOY_SCRIPT="scripts/ops/deploy-rum-dev.sh"

for command in gh git mktemp bash helm; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "DEV DEPLOY BLOCKED: required command unavailable: $command" >&2
    exit 2
  }
done

TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || {
  echo "DEV DEPLOY BLOCKED: no GitHub token is available to verify the exact candidate" >&2
  exit 2
}

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || {
  echo "DEV DEPLOY BLOCKED: requested SHA is not the exact current RUM owner-candidate branch head." >&2
  exit 78
}

pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "DEV DEPLOY BLOCKED: RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate." >&2
  exit 78
}

ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "'"${EXPECTED_CI_NAME}"'" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || {
  echo "DEV DEPLOY BLOCKED: no successful exact-head ${EXPECTED_CI_NAME} workflow run exists for ${CANDIDATE_SHA}." >&2
  exit 78
}

main_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/main" --jq '.object.sha')"
[[ "$main_sha" != "$CANDIDATE_SHA" ]] || {
  echo "DEV DEPLOY BLOCKED: candidate unexpectedly equals RUM main." >&2
  exit 78
}

tmp_root="$(mktemp -d)"
cleanup() { rm -rf "$tmp_root"; }
trap cleanup EXIT HUP INT TERM

GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp_root/rum" -- --no-checkout --filter=blob:none >/dev/null
cd "$tmp_root/rum"
git checkout --detach "$CANDIDATE_SHA" >/dev/null
[[ "$(git rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || {
  echo "DEV DEPLOY BLOCKED: disposable checkout is not the exact candidate SHA." >&2
  exit 78
}
[[ -f "$DEPLOY_SCRIPT" && ! -L "$DEPLOY_SCRIPT" ]] || {
  echo "DEV DEPLOY BLOCKED: candidate isolated-DEV deploy script is missing or not a regular file." >&2
  exit 78
}

grep -Fq 'NAMESPACE="rum-dev-isolated"' "$DEPLOY_SCRIPT" || {
  echo "DEV DEPLOY BLOCKED: candidate deploy script is not pinned to rum-dev-isolated." >&2
  exit 78
}
grep -Fq 'DEV_HOST="dev-rum.daisycloversoftware.uk"' "$DEPLOY_SCRIPT" || {
  echo "DEV DEPLOY BLOCKED: candidate deploy script is not pinned to the authorised DEV host." >&2
  exit 78
}
grep -Fq 'PUBLIC_HOST="rateurmate.online"' "$DEPLOY_SCRIPT" || {
  echo "DEV DEPLOY BLOCKED: candidate deploy script lacks the LIVE-host fail-closed guard." >&2
  exit 78
}

unset TOKEN GH_TOKEN GHCR_TOKEN
printf 'RUM_ISOLATED_DEV_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_ISOLATED_DEV_IMAGE_TAG=%s\n' "$TAG"
printf 'RUM_ISOLATED_DEV_NAMESPACE=rum-dev-isolated\n'
printf 'RUM_LIVE_NAMESPACE_MUTATED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'

bash "$DEPLOY_SCRIPT" "$TAG"

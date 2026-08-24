#!/usr/bin/env bash
set -euo pipefail
umask 077

REPOSITORY="DaisyCloverSoftware/rum"
RUN_ID="32680021440"
EXPECTED_NAME="Container release"
EXPECTED_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
EXPECTED_OLD_SHA="a38dcfd49d92e36accea61c69224dc0f4f801892"
CURRENT_CANDIDATE="18a2ce07ae003bb225f356b758cd59b9ba5f3f8c"

command -v gh >/dev/null 2>&1 || { echo "required command unavailable: gh" >&2; exit 2; }
TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "no GitHub token available" >&2; exit 2; }

current_head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${EXPECTED_BRANCH}" --jq '.object.sha')"
[[ "$current_head" == "$CURRENT_CANDIDATE" ]] || { echo "CANCEL BLOCKED: current candidate moved" >&2; exit 78; }

run_json="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs/${RUN_ID}")"
name="$(jq -r '.name' <<<"$run_json")"
branch="$(jq -r '.head_branch' <<<"$run_json")"
head_sha="$(jq -r '.head_sha' <<<"$run_json")"
status="$(jq -r '.status' <<<"$run_json")"
[[ "$name" == "$EXPECTED_NAME" && "$branch" == "$EXPECTED_BRANCH" && "$head_sha" == "$EXPECTED_OLD_SHA" ]] || {
  echo "CANCEL BLOCKED: run identity mismatch" >&2
  exit 78
}
if [[ "$status" == "completed" ]]; then
  printf 'RUN_ALREADY_COMPLETED=%s\n' "$RUN_ID"
  exit 0
fi
[[ "$status" == "queued" || "$status" == "pending" || "$status" == "in_progress" ]] || { echo "CANCEL BLOCKED: unexpected status=$status" >&2; exit 78; }
GH_TOKEN="$TOKEN" gh api --method POST "repos/${REPOSITORY}/actions/runs/${RUN_ID}/cancel" >/dev/null
printf 'CANCEL_REQUESTED=%s old_sha=%s current_candidate=%s\n' "$RUN_ID" "$EXPECTED_OLD_SHA" "$CURRENT_CANDIDATE"

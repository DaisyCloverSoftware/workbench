#!/usr/bin/env bash
set -euo pipefail
umask 077

REPOSITORY="DaisyCloverSoftware/rum"
EXPECTED_BRANCH="ops/rum-candidate-image-publisher-20260823"
EXPECTED_NAMES=(
  "RUM owner candidate image publication"
  "RUM owner candidate full-SHA publication"
)
RUN_IDS=(
  32664302960
  32664312225
  32666304703
  32666319315
  32666319350
)

command -v gh >/dev/null 2>&1 || { echo "required command unavailable: gh" >&2; exit 2; }
TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || { echo "no GitHub token available" >&2; exit 2; }

name_allowed() {
  local candidate="$1" expected
  for expected in "${EXPECTED_NAMES[@]}"; do
    [[ "$candidate" == "$expected" ]] && return 0
  done
  return 1
}

for run_id in "${RUN_IDS[@]}"; do
  run_json="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs/${run_id}")"
  name="$(jq -r '.name' <<<"$run_json")"
  branch="$(jq -r '.head_branch' <<<"$run_json")"
  status="$(jq -r '.status' <<<"$run_json")"
  conclusion="$(jq -r '.conclusion // ""' <<<"$run_json")"

  name_allowed "$name" || {
    echo "CLEANUP BLOCKED: run ${run_id} has unexpected workflow name ${name}." >&2
    exit 78
  }
  [[ "$branch" == "$EXPECTED_BRANCH" ]] || {
    echo "CLEANUP BLOCKED: run ${run_id} belongs to unexpected branch ${branch}." >&2
    exit 78
  }

  if [[ "$status" == "completed" ]]; then
    printf 'RUN_ALREADY_COMPLETED=%s conclusion=%s\n' "$run_id" "$conclusion"
    continue
  fi
  [[ "$status" == "in_progress" || "$status" == "queued" || "$status" == "pending" ]] || {
    echo "CLEANUP BLOCKED: run ${run_id} has unexpected active status ${status}." >&2
    exit 78
  }

  GH_TOKEN="$TOKEN" gh api --method POST "repos/${REPOSITORY}/actions/runs/${run_id}/cancel" >/dev/null
  printf 'CANCEL_REQUESTED=%s workflow=%s\n' "$run_id" "$name"
done

printf 'RUM_STALE_PUBLISHER_CLEANUP_COMPLETE=1\n'

#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 owner/repository PR_NUMBER [EXPECTED_HEAD_SHA]" >&2
  exit 64
}

[ "$#" -ge 2 ] && [ "$#" -le 3 ] || usage
repository="$(printf '%s' "$1" | xargs)"
pr_number="$2"
expected_head_sha="${3:-}"

if [[ ! "$repository" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || [ "${#repository}" -gt 200 ]; then
  echo "repository must be an owner/name GitHub repository" >&2
  exit 64
fi
if [[ ! "$pr_number" =~ ^[1-9][0-9]*$ ]]; then
  echo "PR number must be a positive integer" >&2
  exit 64
fi
if [ -n "$expected_head_sha" ] && [[ ! "$expected_head_sha" =~ ^[0-9a-fA-F]{40}$ ]]; then
  echo "expected head SHA must be a full 40-character commit SHA" >&2
  exit 64
fi

owner="${repository%%/*}"
name="${repository#*/}"

if ! command -v gh >/dev/null 2>&1 || ! gh auth status --hostname github.com >/dev/null 2>&1; then
  echo "GitHub authentication unavailable" >&2
  exit 69
fi

query='query PullRequestReadyPreflight($owner:String!,$name:String!,$number:Int!) { repository(owner:$owner,name:$name) { pullRequest(number:$number) { id state isDraft merged headRefOid } } }'
mutation='mutation MarkPullRequestReadyForReview($pullRequestId:ID!) { markPullRequestReadyForReview(input:{pullRequestId:$pullRequestId}) { pullRequest { id } } }'

read_pr() {
  gh api graphql \
    -f "query=$query" \
    -f "owner=$owner" \
    -f "name=$name" \
    -F "number=$pr_number" \
    --jq '.data.repository.pullRequest | if . == null then error("pull request not found") else [.id,.state,(.isDraft|tostring),(.merged|tostring),.headRefOid] | @tsv end'
}

if ! before="$(read_pr)"; then
  echo "GitHub pull request preflight read failed" >&2
  exit 65
fi
IFS=$'\t' read -r node_id state is_draft merged head_sha <<<"$before"

if [ "$state" != "OPEN" ] || [ "$merged" != "false" ]; then
  echo "pull request must currently be OPEN + DRAFT" >&2
  exit 65
fi
if [ "$is_draft" != "true" ]; then
  echo "pull request is already ready for review" >&2
  exit 65
fi
if [ -n "$expected_head_sha" ] && [ "${head_sha,,}" != "${expected_head_sha,,}" ]; then
  echo "expected head SHA mismatch: expected $expected_head_sha, found $head_sha" >&2
  exit 65
fi

if ! gh api graphql -f "query=$mutation" -f "pullRequestId=$node_id" >/dev/null; then
  echo "GitHub ready-for-review mutation failed" >&2
  exit 70
fi

if ! after="$(read_pr)"; then
  echo "ready-for-review mutation completed but post-read failed" >&2
  exit 70
fi
IFS=$'\t' read -r _post_node_id post_state post_is_draft post_merged post_head_sha <<<"$after"
if [ "$post_state" != "OPEN" ] || [ "$post_merged" != "false" ] || [ "$post_is_draft" != "false" ]; then
  echo "ready-for-review mutation did not produce an OPEN, unmerged, non-draft pull request" >&2
  exit 70
fi

printf '{"repository":"%s","pr_number":%s,"draft_state":false,"merged_state":false,"head_sha":"%s","resulting_status":"READY_FOR_REVIEW"}\n' \
  "$repository" "$pr_number" "$post_head_sha"

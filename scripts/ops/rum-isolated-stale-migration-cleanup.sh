#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-current-rum-sha>" >&2
  exit 64
fi
CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_PR=153
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
STALE_JOB="rum-migrate-3"
STALE_IMAGE="ghcr.io/daisycloversoftware/rum-api:sha-03265cc"
EXPECTED_TAG="sha-${CANDIDATE_SHA:0:8}"
LIVE_TAG="sha-8106675"

for command in gh kubectl; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done

pr="$(gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}")"
python3 - "$CANDIDATE_SHA" "$CANDIDATE_BRANCH" "$pr" <<'PY'
import json,sys
sha,branch,raw=sys.argv[1:]
pr=json.loads(raw)
if pr.get('state') != 'open' or not pr.get('draft') or pr.get('merged_at') is not None:
    raise SystemExit('cleanup blocked: RUM PR is not open/draft/unmerged')
if pr.get('head',{}).get('ref') != branch or pr.get('head',{}).get('sha') != sha:
    raise SystemExit('cleanup blocked: requested SHA is not exact current RUM candidate head')
PY

job_image="$(kubectl -n "$NAMESPACE" get job "$STALE_JOB" -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)"
if [[ -z "$job_image" ]]; then
  echo "RUM_STALE_MIGRATION_ALREADY_ABSENT=YES"
  echo "RUM_LIVE_NAMESPACE_MUTATED=NO"
  echo "RATE_ANYTHING_AFFECTED=NO"
  exit 0
fi
[[ "$job_image" == "$STALE_IMAGE" ]] || {
  echo "cleanup blocked: $NAMESPACE/$STALE_JOB image is $job_image, expected $STALE_IMAGE" >&2
  exit 78
}

for deployment in rum-web rum-api rum-worker; do
  image="$(kubectl -n "$NAMESPACE" get deployment "$deployment" -o jsonpath='{.spec.template.spec.containers[0].image}')"
  [[ "$image" == *":${EXPECTED_TAG}" ]] || {
    echo "cleanup blocked: isolated $deployment is not on exact candidate $EXPECTED_TAG ($image)" >&2
    exit 78
  }
done
for deployment in rum-web rum-api rum-worker; do
  image="$(kubectl -n "$LIVE_NAMESPACE" get deployment "$deployment" -o jsonpath='{.spec.template.spec.containers[0].image}')"
  [[ "$image" == *":${LIVE_TAG}" ]] || {
    echo "cleanup blocked: LIVE $deployment is not on expected immutable baseline $LIVE_TAG ($image)" >&2
    exit 78
  }
done

kubectl -n "$NAMESPACE" delete job "$STALE_JOB" --wait=true --timeout=60s

if kubectl -n "$NAMESPACE" get job "$STALE_JOB" >/dev/null 2>&1; then
  echo "cleanup failed: stale migration job still exists" >&2
  exit 70
fi

echo "RUM_STALE_MIGRATION_REMOVED=$STALE_JOB"
echo "RUM_STALE_MIGRATION_IMAGE=$STALE_IMAGE"
echo "RUM_ISOLATED_CURRENT_TAG=$EXPECTED_TAG"
echo "RUM_LIVE_NAMESPACE_MUTATED=NO"
echo "RATE_ANYTHING_AFFECTED=NO"

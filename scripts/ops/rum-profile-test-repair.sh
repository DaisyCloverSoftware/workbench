#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi
CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase candidate SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
PR=153
TARGET="apps/api/tests/Feature/Profile/UserProfileTest.php"
for command in gh git python3 base64 mktemp; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; [[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "PATCH BLOCKED: PR not open/draft/unmerged exact head" >&2; exit 78; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null
python3 - "$tmp/rum/$TARGET" <<'PY'
from pathlib import Path
import sys
path=Path(sys.argv[1])
text=path.read_text()
old="$founder->forceFill(['founder_number' => 27, 'founder_alpha_status' => 'active'])->save();"
new="$founder->forceFill(['founder_number' => 27])->save();"
if text.count(old) != 1:
    raise SystemExit('Founder fixture repair marker mismatch')
path.write_text(text.replace(old,new,1))
PY

blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${TARGET}?ref=${BRANCH}" --jq '.sha')"
body="$(base64 -w0 "$tmp/rum/$TARGET")"
response="$(GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${TARGET}" \
  -f message='test: use canonical founder number fixture' -f content="$body" -f sha="$blob" -f branch="$BRANCH")"
next="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["commit"]["sha"])' <<<"$response")"
[[ "$next" =~ ^[0-9a-f]{40}$ ]] || { echo "PATCH BLOCKED: invalid commit response" >&2; exit 70; }
final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$final" == "$next" ]] || { echo "PATCH BLOCKED: final branch head mismatch" >&2; exit 78; }
printf 'RUM_PROFILE_TEST_REPAIR_HEAD=%s\n' "$final"
printf 'FOUNDER_FIXTURE_SOURCE=founder_number\n'

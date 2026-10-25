#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then echo "usage: $0 <exact-rum-candidate-sha>" >&2; exit 64; fi
CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase SHA required" >&2; exit 64; }
REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
PR=153
for command in gh git python3 base64 mktemp; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; [[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"; [[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }
head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "PATCH BLOCKED: PR state mismatch" >&2; exit 78; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null
python3 - "$tmp/rum" <<'PY'
from pathlib import Path
import sys
root=Path(sys.argv[1])
replacements={
 'apps/api/tests/Feature/Entity/EntityBridgeApiTest.php': (
'''            'link_type' => 'controls_identity',
            'status' => 'active',
            'verification_state' => 'claimed',
            'source' => 'claim',
''',
'''            'link_type' => 'controls_identity',
            'status' => 'active',
            'verification_state' => 'self_declared',
            'source' => 'claim',
'''),
 'apps/api/tests/Feature/Entity/PublicPersonEntityBridgeTest.php': (
'''            'link_type' => 'represents_person',
            'status' => 'active',
            'verification_state' => 'claimed',
            'source' => 'claim',
''',
'''            'link_type' => 'represents_person',
            'status' => 'active',
            'verification_state' => 'self_declared',
            'source' => 'claim',
'''),
}
for rel,(old,new) in replacements.items():
    p=root/rel; text=p.read_text(); count=text.count(old)
    if count != 1: raise SystemExit(f'{rel}: expected regression marker once, found {count}')
    p.write_text(text.replace(old,new,1))
PY
publish(){
 local path="$1" message="$2" expected="$3" current blob body response next
 current="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"; [[ "$current" == "$expected" ]] || { echo "PATCH BLOCKED: branch moved before $path" >&2; exit 78; }
 blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${BRANCH}" --jq '.sha')"
 body="$(base64 -w0 "$tmp/rum/$path")"
 response="$(GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${path}" -f message="$message" -f content="$body" -f sha="$blob" -f branch="$BRANCH")"
 next="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["commit"]["sha"])' <<<"$response")"
 [[ "$next" =~ ^[0-9a-f]{40}$ ]] || exit 70; printf '%s' "$next"
}
next="$(publish 'apps/api/tests/Feature/Entity/EntityBridgeApiTest.php' 'test: expect self-declared account link state' "$CANDIDATE_SHA")"
next="$(publish 'apps/api/tests/Feature/Entity/PublicPersonEntityBridgeTest.php' 'test: expect self-declared person link state' "$next")"
printf 'RUM_API_REGRESSION_FIX_HEAD=%s\n' "$next"
printf 'RUM_PR_153_STATE=OPEN_DRAFT_UNMERGED\n'
printf 'LIVE_MUTATED=NO\nRATE_ANYTHING_MUTATED=NO\n'

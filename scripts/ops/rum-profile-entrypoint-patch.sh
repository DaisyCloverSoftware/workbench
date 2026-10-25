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

python3 - "$tmp/rum" <<'PY'
from pathlib import Path
import sys
root=Path(sys.argv[1])

people=root/'apps/web/src/screens/PeopleScreen.tsx'
text=people.read_text()
old="  const scoreVisible = member.profileAccess !== false && typeof member.score === 'number' && typeof member.ratingCount === 'number';\n\n  return <Card className=\"person-card\">"
new="  const scoreVisible = member.profileAccess !== false && typeof member.score === 'number' && typeof member.ratingCount === 'number';\n  const profileHref = member.profileAccess === false ? null : `/people/${encodeURIComponent(member.id)}/profile`;\n\n  return <Card className=\"person-card\">"
if text.count(old) != 1: raise SystemExit('People profileHref insertion marker mismatch')
text=text.replace(old,new,1)
old="    <div className=\"person-card__actions\">\n      {tab === 'requests'"
new="    <div className=\"person-card__actions\">\n      {profileHref ? <Button variant=\"ghost\" icon=\"me\" onClick={() => window.location.assign(profileHref)}>Profile</Button> : null}\n      {tab === 'requests'"
if text.count(old) != 1: raise SystemExit('People profile action insertion marker mismatch')
text=text.replace(old,new,1)
people.write_text(text)

me=root/'apps/web/src/screens/MeScreen.tsx'
text=me.read_text()
old='<div className="profile-hero__actions"><Button variant="secondary" icon="edit" onClick={() => onEdit(\'profile\')}>Edit profile</Button><Button variant="ghost" icon="shield" onClick={() => onEdit(\'privacy\')}>Privacy</Button></div>'
new='<div className="profile-hero__actions"><Button variant="secondary" icon="me" onClick={() => window.location.assign(\'/profile\')}>View profile</Button><Button variant="secondary" icon="edit" onClick={() => onEdit(\'profile\')}>Edit profile</Button><Button variant="ghost" icon="shield" onClick={() => onEdit(\'privacy\')}>Privacy</Button></div>'
if text.count(old) != 1: raise SystemExit('Me profile action marker mismatch')
text=text.replace(old,new,1)
me.write_text(text)
PY

publish_file() {
  local path="$1" message="$2" expected="$3"
  local current blob body response next
  current="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
  [[ "$current" == "$expected" ]] || { echo "PATCH BLOCKED: branch moved before $path" >&2; exit 78; }
  blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${BRANCH}" --jq '.sha')"
  body="$(base64 -w0 "$tmp/rum/$path")"
  response="$(GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${path}" -f message="$message" -f content="$body" -f sha="$blob" -f branch="$BRANCH")"
  next="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["commit"]["sha"])' <<<"$response")"
  [[ "$next" =~ ^[0-9a-f]{40}$ ]] || { echo "PATCH BLOCKED: invalid commit response" >&2; exit 70; }
  printf '%s=%s\n' "$(basename "$path" | tr '.-' '__')_commit" "$next"
  printf '%s' "$next"
}

next="$(publish_file 'apps/web/src/screens/PeopleScreen.tsx' 'feat: link mate cards to profiles' "$CANDIDATE_SHA" | tail -c 40)"
next="$(publish_file 'apps/web/src/screens/MeScreen.tsx' 'feat: add self profile entry point' "$next" | tail -c 40)"
final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$final" == "$next" ]] || { echo "PATCH BLOCKED: final branch head mismatch" >&2; exit 78; }
printf 'RUM_PROFILE_ENTRYPOINT_HEAD=%s\n' "$final"
printf 'RUM_PROFILE_ENTRYPOINTS=PEOPLE_AND_SELF\n'

#!/usr/bin/env bash
set -euo pipefail
umask 077
REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
EXPECTED_HEAD="f19174c95b9a86434e965b1a64c91bec9b8f38d1"
SOURCE_REF="cbc2a860ed66540de01bca6ed56ea1bba476f70f"
PATH_NAME="apps/api/tests/Feature/Entity/PublicPersonEntityBridgeTest.php"
for c in gh jq base64 python3 mktemp; do command -v "$c" >/dev/null 2>&1 || { echo "missing $c" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "no token" >&2; exit 2; }
head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$EXPECTED_HEAD" ]] || { echo "RESTORE BLOCKED expected=$EXPECTED_HEAD actual=$head" >&2; exit 78; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${PATH_NAME}?ref=${SOURCE_REF}" >"$tmp/source.json"
GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${PATH_NAME}?ref=${BRANCH}" >"$tmp/current.json"
jq -r '.content' "$tmp/source.json" | tr -d '\n' | base64 -d >"$tmp/file"
python3 - "$tmp/file" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); s=p.read_text()
old="            'link_type' => 'person',"
new="            'link_type' => 'represents_person',"
if s.count(old) != 1:
    raise SystemExit(f'expected exactly one link_type person assertion, found {s.count(old)}')
p.write_text(s.replace(old,new))
PY
sha="$(jq -r '.sha' "$tmp/current.json")"
encoded="$(base64 -w0 <"$tmp/file")"
GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${PATH_NAME}" \
  -f message="test: use canonical represents-person link type" -f content="$encoded" -f sha="$sha" -f branch="$BRANCH" >/dev/null
final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
printf 'RUM153_PERSON_TEST_RESTORED_HEAD=%s\n' "$final"

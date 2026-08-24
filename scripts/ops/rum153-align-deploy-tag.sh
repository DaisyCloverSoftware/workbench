#!/usr/bin/env bash
set -euo pipefail
umask 077
REPOSITORY="DaisyCloverSoftware/workbench"
BRANCH="ops/rum-candidate-image-publisher-20260823"
EXPECTED_HEAD="4e5245c70419aa09121a1272bb64daf5a4a47684"
PATH_NAME="scripts/ops/rum-isolated-dev-deployer.sh"
for c in gh jq base64 python3 mktemp; do command -v "$c" >/dev/null 2>&1 || { echo "missing $c" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "no token" >&2; exit 2; }
head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$EXPECTED_HEAD" ]] || { echo "ALIGN BLOCKED expected=$EXPECTED_HEAD actual=$head" >&2; exit 78; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${PATH_NAME}?ref=${BRANCH}" >"$tmp/meta.json"
sha="$(jq -r '.sha' "$tmp/meta.json")"
jq -r '.content' "$tmp/meta.json" | tr -d '\n' | base64 -d >"$tmp/file"
python3 - "$tmp/file" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); s=p.read_text()
old='TAG="sha-${CANDIDATE_SHA:0:7}"'
new='TAG="sha-${CANDIDATE_SHA:0:8}"'
if s.count(old) != 1:
    raise SystemExit(f'expected one 7-char deploy tag, found {s.count(old)}')
p.write_text(s.replace(old,new))
PY
encoded="$(base64 -w0 <"$tmp/file")"
GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${PATH_NAME}" \
  -f message="ops: align isolated DEV deploy tag to normalized publisher" -f content="$encoded" -f sha="$sha" -f branch="$BRANCH" >/dev/null
final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
printf 'WORKBENCH_DEPLOY_TAG_ALIGNED_HEAD=%s\n' "$final"

#!/usr/bin/env bash
set -euo pipefail
umask 077

REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
EXPECTED_HEAD="aa787a9f51853f26c71abefbdacd39fec29bf75c"

for command in gh jq python3 base64 mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done

TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || { echo "no GitHub token available" >&2; exit 2; }

current_head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$current_head" == "$EXPECTED_HEAD" ]] || {
  echo "PATCH BLOCKED: RUM PR #153 head moved: expected=${EXPECTED_HEAD} actual=${current_head}" >&2
  exit 78
}

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT HUP INT TERM

update_file() {
  local path="$1"
  local message="$2"
  local transform="$3"
  local json="$tmp_root/file.json"
  local decoded="$tmp_root/file.txt"
  local encoded sha

  GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${BRANCH}" >"$json"
  sha="$(jq -r '.sha' "$json")"
  jq -r '.content' "$json" | tr -d '\n' | base64 -d >"$decoded"
  python3 - "$decoded" "$transform" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
transform = sys.argv[2]
source = path.read_text()

if transform == 'builder-generic':
    old = '    private function applyScope(Builder $query, ?string $scopeKey, ?string $scopeValue): void\n'
    new = '    /** @param Builder<RatingEvent> $query */\n    private function applyScope(Builder $query, ?string $scopeKey, ?string $scopeValue): void\n'
    if source.count(old) != 1:
        raise SystemExit(f'expected exactly one applyScope signature, found {source.count(old)}')
    if new in source:
        raise SystemExit('applyScope generic annotation already present')
    source = source.replace(old, new)
elif transform == 'remove-obsolete-rate-my-baseline':
    lines = source.splitlines(keepends=True)
    match = [i for i, line in enumerate(lines) if 'RateMyService' in line and 'rate\\(' in line and 'return type has no value type specified' in line]
    if len(match) != 1:
        raise SystemExit(f'expected exactly one obsolete RateMyService::rate baseline entry, found {len(match)}')
    i = match[0]
    start = i - 1
    if start < 0 or lines[start].strip() != '-':
        raise SystemExit('unexpected baseline block start')
    end = i
    while end < len(lines) and 'path: app/Services/RateMyService.php' not in lines[end]:
        end += 1
    if end >= len(lines):
        raise SystemExit('unexpected baseline block end')
    end += 1
    if end < len(lines) and lines[end].strip() == '':
        end += 1
    del lines[start:end]
    source = ''.join(lines)
else:
    raise SystemExit(f'unknown transform: {transform}')

path.write_text(source)
PY

  encoded="$(base64 -w0 <"$decoded")"
  GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${path}" \
    -f message="$message" \
    -f content="$encoded" \
    -f sha="$sha" \
    -f branch="$BRANCH" >/dev/null
  printf 'UPDATED=%s\n' "$path"
}

update_file "apps/api/app/Services/RateAnythingRatingService.php" "fix: type canonical rating scope builder" "builder-generic"
update_file "apps/api/phpstan-baseline.neon" "chore: remove obsolete Rate My phpstan suppression" "remove-obsolete-rate-my-baseline"

final_head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
printf 'RUM153_STATIC_ANALYSIS_FIX_HEAD=%s\n' "$final_head"

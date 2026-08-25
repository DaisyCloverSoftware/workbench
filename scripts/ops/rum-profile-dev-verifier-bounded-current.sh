#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-profile-dev-verifier-bounded.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || {
  echo "VERIFY BLOCKED: bounded profile verifier unavailable" >&2
  exit 78
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/rum-profile-dev-verifier-bounded.sh"
cp "$SOURCE" "$COPY"

# The canonical LIVE-derived member-rating flow redirects public submissions
# straight to /judge. The older base profile verifier still waits for the
# linked-Thing success heading. Patch only that assertion, fail-closed, while
# retaining every exact-candidate, isolated-DEV, profile/privacy/Judge/mobile
# assertion from the bounded verifier.
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text()
marker = '''replace_once(
    "import os,base64\\n",
'''
insert = '''replace_once(
    "    page.get_by_role('heading', name='Your verdict is in.').wait_for(state='visible', timeout=30000)",
    "    page.wait_for_url('**/judge', timeout=30000)",
    "canonical public member rating redirect",
)
'''
if text.count(marker) != 1:
    raise SystemExit(f'profile verifier redirect patch blocked: marker count={text.count(marker)}')
text = text.replace(marker, insert + marker, 1)
path.write_text(text)
PY
chmod 0700 "$COPY"
printf 'RUM_PROFILE_RATING_COMPLETION_ASSERTION=CANONICAL_JUDGE_REDIRECT\n'
bash "$COPY" "$@"

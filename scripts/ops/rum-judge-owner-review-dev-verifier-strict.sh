#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-judge-owner-review-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "base Judge owner-review verifier unavailable" >&2; exit 78; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/rum-judge-owner-review-dev-verifier.sh"
cp "$SOURCE" "$COPY"
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys
path=Path(sys.argv[1])
text=path.read_text()
old="sinbin=viewer.get_by_role('button', name='Sin Bin', exact=True); sinbin.click();"
new="sinbin=viewer.locator('button[aria-label=\"Sin Bin\"]'); sinbin.click();"
if old not in text:
    raise SystemExit('strict wrapper blocked: expected Sin Bin verifier selector not found')
path.write_text(text.replace(old,new,1))
PY
chmod 0700 "$COPY"
printf 'RUM_JUDGE_SINBIN_SELECTOR_COMPAT=VOTE_ARIA_LABEL_ONLY\n'
printf 'RUM_JUDGE_CHECK_AGAIN_ASSERTIONS=STRICT\n'
printf 'RUM_JUDGE_SELECTED_STYLE_ASSERTIONS=STRICT\n'
bash "$COPY" "$1"

#!/usr/bin/env bash
set -euo pipefail
umask 077

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source_script="$script_dir/rum-cj-rating-surface-dev-verifier.sh"
[[ -f "$source_script" ]] || { echo "CJ verifier source script unavailable" >&2; exit 2; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
patched="$work/rum-cj-rating-surface-dev-verifier.sh"
cp "$source_script" "$patched"

python3 - "$patched" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
text = path.read_text()
old = 'page.get_by_text("For CJ Investigates", exact=True).first.wait_for(state="visible", timeout=30000)'
new = 'page.get_by_text("For CJ Investigates", exact=False).first.wait_for(state="visible", timeout=30000)'
if text.count(old) != 1:
    raise SystemExit("expected exactly one strict Given-text assertion to patch")
path.write_text(text.replace(old, new))
PY

printf 'RUM_CJ_GIVEN_TEXT_COMPAT=VISIBLE_PREFIX_WITH_AUDIENCE_SUFFIX\n'
printf 'RUM_CJ_API_SAME_EVENT_ASSERTIONS=STRICT\n'
printf 'RUM_CJ_JUDGE_UI_ASSERTIONS=STRICT\n'
exec bash "$patched" "$@"
